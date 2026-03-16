package handler

import (
	"Rx-ui/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService *service.UserService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// RegisterRoutes 注册路由
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
		auth.GET("/me", h.GetCurrentUser)
		auth.PUT("/password", h.ChangePassword)
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token,omitempty"`
}

// Login 用户登录
// @Summary 用户登录
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录信息"
// @Success 200 {object} Response{data=LoginResponse}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请输入用户名和密码")
		return
	}

	user, err := h.userService.Authenticate(req.Username, req.Password)
	if err != nil {
		Unauthorized(c, "用户名或密码错误")
		return
	}

	// TODO: 生成 JWT token 或设置 session
	_ = user

	Success(c, LoginResponse{
		Token: "", // TODO: 返回 token
	})
}

// Logout 用户登出
// @Summary 用户登出
// @Tags Auth
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: 清除 session 或使 token 失效
	SuccessMsg(c, "登出成功", nil)
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Tags Auth
// @Produce json
// @Success 200 {object} Response
// @Failure 401 {object} Response
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// TODO: 从 session/token 获取当前用户
	Unauthorized(c, "未登录")
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "密码信息"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /api/v1/auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请输入旧密码和新密码")
		return
	}

	// TODO: 验证旧密码并更新
	SuccessMsg(c, "密码修改成功", nil)
}
