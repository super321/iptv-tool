package publish

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/pkg/i18n"
)

// requestScheme determines the URL scheme (http or https) from the request.
// It checks X-Forwarded-Proto header first (for reverse proxy setups),
// then falls back to checking if the connection is TLS.
func requestScheme(c *gin.Context) string {
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(proto)
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

// checkUserAgent returns true if reqUA contains at least one of the newline-separated allowed values.
// Returns false when reqUA is empty or no match is found.
// NOTE: newline (\n) is used as separator instead of comma, because UA strings may contain commas.
func checkUserAgent(reqUA, allowedValues string) bool {
	if reqUA == "" {
		return false
	}
	for _, v := range strings.Split(allowedValues, "\n") {
		v = strings.TrimSpace(v)
		if v != "" && strings.Contains(reqUA, v) {
			return true
		}
	}
	return false
}

// Handler serves published subscription endpoints
// GET /sub/live/:path
func LiveHandler(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.not_found"))
		return
	}

	// Look up the publish interface by path and type
	var iface model.PublishInterface
	if err := model.DB.Where("path = ? AND type = ? AND status = ?", path, "live", true).First(&iface).Error; err != nil {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.sub_not_found"))
		return
	}

	// UA validation
	if iface.UACheckEnabled {
		if !checkUserAgent(c.GetHeader("User-Agent"), iface.UAAllowedValues) {
			c.Status(http.StatusForbidden)
			return
		}
	}

	engine, err := NewEngine(iface)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.engine_init_failed"), err.Error())
		return
	}

	requestHost := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		requestHost = fwd
	}
	requestBaseURL := requestScheme(c) + "://" + requestHost

	serveLive(c, engine, iface, requestBaseURL)
}

// EPGHandler serves EPG subscription endpoints
// GET /sub/epg/:path
func EPGHandler(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.not_found"))
		return
	}

	var iface model.PublishInterface
	if err := model.DB.Where("path = ? AND type = ? AND status = ?", path, "epg", true).First(&iface).Error; err != nil {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.sub_not_found"))
		return
	}

	// UA validation
	if iface.UACheckEnabled {
		if !checkUserAgent(c.GetHeader("User-Agent"), iface.UAAllowedValues) {
			c.Status(http.StatusForbidden)
			return
		}
	}

	engine, err := NewEngine(iface)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.engine_init_failed"), err.Error())
		return
	}

	requestHost := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		requestHost = fwd
	}
	requestBaseURL := requestScheme(c) + "://" + requestHost

	serveEPG(c, engine, iface, requestBaseURL)
}

func serveLive(c *gin.Context, engine *Engine, iface model.PublishInterface, requestBaseURL string) {
	channels, err := LoadOrStoreLiveChannels(iface.ID, engine.AggregateLiveChannels)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.channel_agg_failed"), err.Error())
		return
	}

	switch iface.Format {
	case model.PublishFormatM3U:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		// Pass requestBaseURL for logo URL resolution in M3U format
		c.String(http.StatusOK, engine.FormatM3U(channels, requestBaseURL))
	case model.PublishFormatTXT:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, engine.FormatTXT(channels))
	default:
		c.String(http.StatusBadRequest, i18n.T(i18n.Lang(c), "publish_handler.live_format_unsupported"))
	}
}

func serveEPG(c *gin.Context, engine *Engine, iface model.PublishInterface, _ string) { // requestBaseURL unused for EPG
	epg, err := LoadOrStoreEPGPrograms(iface.ID, engine.AggregateEPG)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.epg_agg_failed"), err.Error())
		return
	}

	switch iface.Format {
	case model.PublishFormatXMLTV:
		// Only use gzip when explicitly enabled in the interface settings
		if iface.GzipEnabled {
			if err := engine.FormatXMLTVGzip(epg, c.Writer); err != nil {
				c.String(http.StatusInternalServerError, i18n.T(i18n.Lang(c), "publish_handler.gzip_failed"))
			}
		} else {
			c.Header("Content-Type", "application/xml; charset=utf-8")
			if err := engine.FormatXMLTVToWriter(epg, c.Writer); err != nil {
				c.String(http.StatusInternalServerError, i18n.T(i18n.Lang(c), "publish_handler.xmltv_write_failed"))
			}
		}
	case model.PublishFormatDIYP:
		// DIYP JSON format supports query params: ?ch=频道名&date=2024-01-15
		channelName := c.Query("ch")
		dateStr := c.Query("date")
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.String(http.StatusOK, engine.FormatDIYP(epg, channelName, dateStr))
	default:
		c.String(http.StatusBadRequest, i18n.T(i18n.Lang(c), "publish_handler.epg_format_unsupported"))
	}
}

// ServeLiveOrEPG dispatches to the appropriate serve function based on interface type.
// Exported for use by admin download endpoint (bypasses UA check).
// requestBaseURL should include the scheme, e.g. "https://host:port".
func ServeLiveOrEPG(c *gin.Context, engine *Engine, iface model.PublishInterface, requestBaseURL string) {
	if iface.Type == "live" {
		serveLive(c, engine, iface, requestBaseURL)
	} else {
		serveEPG(c, engine, iface, requestBaseURL)
	}
}
