package handler

import (
	"strconv"

	"Rx-ui/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理处理器
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// RegisterRoutes 注册路由
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	users := r.Group("/users")
	{
		users.GET("", h.List)
		users.POST("", h.Create)
		users.PUT("/:id/password", h.ChangePassword)
		users.DELETE("/:id", h.Delete)
	}
}

// List 获取所有用户
// @Summary 获取用户列表
// @Tags Users
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.userService.GetAll()
	if err != nil {
		ErrorFromErr(c, err)
		return
	}
	// 不返回密码
	type SafeUser struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Enable   bool   `json:"enable"`
	}
	var result []SafeUser
	for _, u := range users {
		result = append(result, SafeUser{ID: u.ID, Username: u.Username, Enable: u.Enable})
	}
	Success(c, result)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// Create 创建用户
// @Summary 创建用户
// @Tags Users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "用户信息"
// @Success 201 {object} Response
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请输入用户名和密码（密码至少6位）")
		return
	}

	if err := h.userService.CreateUser(req.Username, req.Password); err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessMsg(c, "用户已创建", nil)
}

// SetPasswordRequest 设置密码请求（管理员操作）
type SetPasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ChangePassword 修改用户密码
// @Summary 修改用户密码
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body ChangePasswordRequest true "新密码"
// @Success 200 {object} Response
// @Router /api/v1/users/{id}/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的用户ID")
		return
	}

	var req SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "密码至少6位")
		return
	}

	if err := h.userService.UpdatePassword(id, req.NewPassword); err != nil {
		ErrorFromErr(c, err)
		return
	}

	SuccessMsg(c, "密码已修改", nil)
}

// Delete 删除用户
// @Summary 删除用户
// @Tags Users
// @Produce json
// @Param id path int true "用户ID"
// @Success 204
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.userService.DeleteUser(id); err != nil {
		ErrorFromErr(c, err)
		return
	}

	NoContent(c)
}
