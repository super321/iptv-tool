package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"iptv-tool-v2/pkg/i18n"
)

var (
	jwtSecret       []byte
	ErrNoToken      = errors.New("error.no_token")
	ErrInvalidToken = errors.New("error.invalid_token")
)

// Claims represents the JWT claims
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// InitJWTSecret generates a random JWT secret on first run, or accepts a provided one.
// Must be called once at startup.
func InitJWTSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
		return
	}
	// Generate a random 32-byte secret
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate JWT secret: " + err.Error())
	}
	jwtSecret = b
}

// GetJWTSecretHex returns the current JWT secret as a hex string (for persistence)
func GetJWTSecretHex() string {
	return hex.EncodeToString(jwtSecret)
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID uint, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)), // Set expiration to 2 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "iptv-tool-v2",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken validates and parses a JWT token string
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// JWTAuthMiddleware is a Gin middleware that checks for a valid JWT token
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.Warn("Unauthorized access attempt: No token", "client_ip", c.ClientIP(), "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T(i18n.Lang(c), "error.no_token")})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			slog.Warn("Unauthorized access attempt: Invalid auth header format", "client_ip", c.ClientIP(), "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_auth_header")})
			c.Abort()
			return
		}

		tokenStr := parts[1]

		claims, err := ParseToken(tokenStr)
		if err != nil {
			slog.Warn("Unauthorized access attempt: Invalid token", "error", err, "client_ip", c.ClientIP(), "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": i18n.T(i18n.Lang(c), "error.invalid_token")})
			c.Abort()
			return
		}

		// Store user info in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
