package huawei

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"iptv-tool-v2/internal/iptv"
	"iptv-tool-v2/pkg/utils"
)

// Client implements the iptv.Client interface for Huawei STBs
type Client struct {
	httpClient *iptv.HTTPClient
	config     *iptv.Config
	host       string            // Stores the host from AuthenticationURL redirect
	headers    map[string]string // Custom headers for requests

	// Authentication state
	Token *Token
}

type Token struct {
	UserToken  string
	Stbid      string
	JSESSIONID string
}

// NewClient creates a new Huawei IPTV client
func NewClient(config *iptv.Config) *Client {
	return &Client{
		httpClient: iptv.NewHTTPClient(nil),
		config:     config,
		headers:    config.Headers,
	}
}

// Authenticate performs the multi-step login to the Huawei IPTV platform
func (c *Client) Authenticate(ctx context.Context) error {
	// Step 1: Visit AuthenticationURL to get redirect and host
	referer, err := c.authenticationURL(ctx, true)
	if err != nil {
		return fmt.Errorf("step 1 (AuthenticationURL) failed: %w", err)
	}

	// Step 2: Access authLoginHWxxx.jsp to get EncryptToken
	encryptToken, err := c.authLoginHW(ctx, referer)
	if err != nil {
		return fmt.Errorf("step 2 (authLogin) failed: %w", err)
	}

	// Step 3: Use 3DES to encrypt data and get JSESSIONID & UserToken
	token, err := c.validAuthenticationHW(ctx, encryptToken)
	if err != nil {
		return fmt.Errorf("step 3 (validAuthentication) failed: %w", err)
	}

	c.Token = token
	return nil
}

func (c *Client) setCommonHeaders(req *http.Request) {
	if c.host != "" {
		req.Header.Set("Host", c.host)
	}
	iptv.SetCommonHeaders(req, c.config)
}

// Step 1
func (c *Client) authenticationURL(ctx context.Context, fccSupport bool) (string, error) {
	if c.config.ServerHost == "" {
		return "", fmt.Errorf("serverHost is not configured")
	}

	// Construct the AuthenticationURL from serverHost, matching old project:
	// http://{serverHost}/EDS/jsp/AuthenticationURL
	authURL := fmt.Sprintf("http://%s/EDS/jsp/AuthenticationURL", c.config.ServerHost)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
	if err != nil {
		return "", err
	}

	params := req.URL.Query()
	params.Add("UserID", c.config.GetAuthParam("UserID"))
	params.Add("Action", "Login")
	if fccSupport {
		params.Add("FCCSupport", "1")
	}
	req.URL.RawQuery = params.Encode()

	c.setCommonHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	// The server usually 302 redirects to a specific EDS server. We need to save the new host.
	c.host = resp.Request.URL.Host

	return resp.Request.URL.String(), nil
}

// Step 2
func (c *Client) authLoginHW(ctx context.Context, referer string) (string, error) {
	data := url.Values{}
	data.Set("UserID", c.config.GetAuthParam("UserID"))

	// Build the path with ProviderSuffix (e.g., authLoginHWCTC.jsp)
	path := fmt.Sprintf("http://%s/EPG/jsp/authLoginHW%s.jsp", c.host, c.config.ProviderSuffix)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	c.setCommonHeaders(req)
	req.Header.Set("Referer", referer)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`EncryptToken\s*=\s*"(.+?)"`)
	matches := regex.FindSubmatch(result)
	if len(matches) != 2 {
		return "", errors.New("failed to parse EncryptToken from response")
	}
	return string(matches[1]), nil
}

// Step 3
func (c *Client) validAuthenticationHW(ctx context.Context, encryptToken string) (*Token, error) {
	random := rand.Intn(90000000) + 10000000 // 8 digit random number

	ipv4Addr, err := c.resolveIP()
	if err != nil {
		return nil, err
	}

	// Format: random$EncryptToken$UserID$STBID$IP$MAC$Reserved$CTC
	plaintext := fmt.Sprintf("%d$%s$%s$%s$%s$%s$$%s",
		random, encryptToken, c.config.GetAuthParam("UserID"), c.config.GetAuthParam("STBID"),
		ipv4Addr, c.config.GetAuthParam("mac"), c.config.ProviderSuffix)

	crypto := utils.NewTripleDESCrypto(c.config.Key)
	authenticator, err := crypto.ECBEncrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authenticator: %w", err)
	}

	data := url.Values{}

	// Set dynamic attributes from config.AuthParams.
	// We only strictly hardcode "Authenticator" and "userToken".
	data.Set("Authenticator", strings.ToUpper(authenticator))
	data.Set("userToken", encryptToken)

	// Loop over user-provided auth params and set them EXACTLY as typed (case-sensitive)
	if c.config.AuthParams != nil {
		for key, val := range c.config.AuthParams {
			// Skip setting Authenticator and userToken as they are securely computed above
			if strings.ToLower(key) == "authenticator" || strings.ToLower(key) == "usertoken" {
				continue
			}
			// Convert the raw value to string format
			var strVal string
			switch v := val.(type) {
			case string:
				strVal = v
			case float64:
				strVal = fmt.Sprintf("%.0f", v)
			case bool:
				strVal = fmt.Sprintf("%t", v)
			default:
				strVal = fmt.Sprintf("%v", v)
			}
			data.Set(key, strVal)
		}
	}

	path := fmt.Sprintf("http://%s/EPG/jsp/ValidAuthenticationHW%s.jsp", c.host, c.config.ProviderSuffix)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	c.setCommonHeaders(req)
	req.Header.Set("Referer", fmt.Sprintf("http://%s/EPG/jsp/authLoginHW%s.jsp", c.host, c.config.ProviderSuffix))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	var jsessionID string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			jsessionID = cookie.Value
			break
		}
	}
	if jsessionID == "" {
		return nil, errors.New("failed to find JSESSIONID in cookies")
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`(?s)"UserToken"\s+value="(.+?)".+?"stbid"\s+value="(.*?)"`)
	matches := regex.FindSubmatch(result)
	if len(matches) != 3 {
		return nil, errors.New("failed to parse UserToken or stbid from final response")
	}

	return &Token{
		UserToken:  string(matches[1]),
		Stbid:      string(matches[2]),
		JSESSIONID: jsessionID,
	}, nil
}

func (c *Client) resolveIP() (string, error) {
	if c.config.InterfaceName != "" {
		iface, err := net.InterfaceByName(c.config.InterfaceName)
		if err != nil {
			return "", err
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	if c.config.IP != "" {
		return c.config.IP, nil
	}
	return "", errors.New("failed to resolve STB IP: interface not found or IP not configured")
}
