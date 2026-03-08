package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"

	"iptv-tool-v2/internal/iptv/huawei"
	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/internal/version"
	"iptv-tool-v2/pkg/auth"
	"iptv-tool-v2/pkg/utils"
)

// captchaStore 验证码存储（内存，默认10分钟过期，自动GC）
var captchaStore = base64Captcha.DefaultMemStore

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

// GetPublicKey returns the RSA public key for frontend password encryption
// GET /api/system/pubkey
func (sc *SystemController) GetPublicKey(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(auth.GetRSAPublicKey()))
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

	plainPassword, err := auth.DecryptRSA(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码解密失败，请刷新页面重试"})
		return
	}

	user, err := sc.userService.Register(req.Username, plainPassword)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrUserExists {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "系统初始化成功",
		"username": user.Username,
	})
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id"`   // 验证码 ID（需要验证码时必填）
	CaptchaCode string `json:"captcha_code"` // 验证码答案（需要验证码时必填）
}

// Login authenticates a user and returns a JWT token
// POST /api/login
func (sc *SystemController) Login(c *gin.Context) {
	// ① IP 频率限制
	clientIP := c.ClientIP()
	if !globalRateLimiter.Allow(clientIP) {
		slog.Warn("Login rate limit exceeded", "client_ip", clientIP)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "登录尝试过于频繁，请稍后再试",
		})
		return
	}

	// ② 解析请求体
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ③ 检查是否需要验证码（连续失败 >= 3 次）
	if globalAttemptTracker.NeedCaptcha(req.Username) {
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			slog.Warn("Captcha required due to multiple failed attempts", "username", req.Username, "client_ip", clientIP)
			c.JSON(http.StatusForbidden, gin.H{
				"error":            "请完成验证码校验",
				"captcha_required": true,
			})
			return
		}
		// 校验验证码（Verify 会自动删除已使用的验证码，防止重放）
		if !captchaStore.Verify(req.CaptchaID, req.CaptchaCode, true) {
			slog.Warn("Invalid captcha attempt", "username", req.Username, "client_ip", clientIP)
			c.JSON(http.StatusForbidden, gin.H{
				"error":            "验证码错误",
				"captcha_required": true,
			})
			return
		}
	}

	// ④ 用户名密码校验
	plainPassword, err := auth.DecryptRSA(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码解密失败，请刷新页面重试"})
		return
	}

	token, err := sc.userService.Login(req.Username, plainPassword)
	if err != nil {
		// 系统未初始化的特殊状态码
		if err == service.ErrSystemNotInit {
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": err.Error()})
			return
		}

		// 记录失败次数并判断是否需要验证码
		globalAttemptTracker.RecordFailure(req.Username)
		needCaptcha := globalAttemptTracker.NeedCaptcha(req.Username)

		slog.Warn("Login failed: invalid credentials", "username", req.Username, "client_ip", clientIP, "need_captcha", needCaptcha)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":            "用户名或密码错误",
			"captcha_required": needCaptcha,
		})
		return
	}

	// ⑤ 登录成功：重置失败计数
	globalAttemptTracker.Reset(req.Username)
	slog.Info("User logged in successfully", "username", req.Username, "client_ip", clientIP)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

// GetCaptcha 生成并返回验证码图片
// GET /api/captcha
func (sc *SystemController) GetCaptcha(c *gin.Context) {
	// 4位数字验证码
	driver := base64Captcha.NewDriverDigit(
		80,  // 高度
		240, // 宽度
		4,   // 位数
		0.7, // 最大倾斜角度
		80,  // 干扰点数量
	)

	captcha := base64Captcha.NewCaptcha(driver, captchaStore)
	id, b64s, _, err := captcha.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"captcha_id":    id,
		"captcha_image": b64s,
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

	plainOldPassword, err := auth.DecryptRSA(req.OldPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "旧密码解密失败，请刷新页面重试"})
		return
	}

	plainNewPassword, err := auth.DecryptRSA(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码解密失败，请刷新页面重试"})
		return
	}

	if err := sc.userService.ChangePassword(userID.(uint), plainOldPassword, plainNewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
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

// GetVersion returns the current application version
// GET /api/system/version
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": version.Version})
}
