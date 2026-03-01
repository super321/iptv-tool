package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/iptv/huawei"
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/pkg/utils"
)

// SystemController handles system initialization and authentication
type SystemController struct {
	userService *service.UserService
}

func NewSystemController() *SystemController {
	return &SystemController{
		userService: service.NewUserService(),
	}
}

// CheckInit returns whether the system has been initialized
// GET /api/init/status
func (sc *SystemController) CheckInit(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"initialized": sc.userService.IsInitialized(),
	})
}

// InitRequest is the request body for system initialization
type InitRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

// Init creates the first admin user
// POST /api/init
func (sc *SystemController) Init(c *gin.Context) {
	var req InitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := sc.userService.Register(req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrUserExists {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "system initialized successfully",
		"username": user.Username,
	})
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user and returns a JWT token
// POST /api/login
func (sc *SystemController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := sc.userService.Login(req.Username, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if err == service.ErrSystemNotInit {
			status = http.StatusPreconditionFailed
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

// ChangePasswordRequest is the request body for changing password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword changes the current user's password
// POST /api/user/password
func (sc *SystemController) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := sc.userService.ChangePassword(userID.(uint), req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// CrackKeyRequest is the request body for cracking the 3DES key
type CrackKeyRequest struct {
	Authenticator string `json:"authenticator" binding:"required"`
}

// CrackKey attempts to brute-force the 3DES key from an authenticator string
// POST /api/crack-key
func (sc *SystemController) CrackKey(c *gin.Context) {
	var req CrackKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	key, err := utils.CrackAuthenticator(ctx, req.Authenticator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "破解失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key})
}

// GetEPGStrategies returns all registered EPG strategy names
// GET /api/epg-strategies
func GetEPGStrategies(c *gin.Context) {
	strategies := huawei.GetAllEPGStrategies()
	result := make([]gin.H, 0, len(strategies)+1)
	result = append(result, gin.H{"value": "auto", "label": "自动检测"})
	for _, s := range strategies {
		result = append(result, gin.H{"value": s.Name(), "label": s.Name()})
	}
	c.JSON(http.StatusOK, result)
}
