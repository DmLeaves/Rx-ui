package handler

import (
	"Rx-ui/internal/service"

	"github.com/gin-gonic/gin"
)

// Router API 路由管理
type Router struct {
	services *service.Services
}

// NewRouter 创建路由管理器
func NewRouter(services *service.Services) *Router {
	return &Router{services: services}
}

// SetupRoutes 设置所有 API 路由
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// API v1
	v1 := engine.Group("/api/v1")
	{
		// 认证
		authHandler := NewAuthHandler(r.services.User)
		authHandler.RegisterRoutes(v1)

		// 用户管理
		userHandler := NewUserHandler(r.services.User)
		userHandler.RegisterRoutes(v1)

		// 入站规则
		inboundHandler := NewInboundHandler(r.services.Inbound)
		inboundHandler.RegisterRoutes(v1)

		// 订阅
		subHandler := NewSubscriptionHandler(r.services.Inbound)
		subHandler.RegisterRoutes(v1)

		// 系统信息
		systemHandler := NewSystemHandler(r.services.Xray)
		systemHandler.RegisterRoutes(v1)

		// 设置
		settingHandler := NewSettingHandler(r.services.Setting)
		settingHandler.RegisterRoutes(v1)

		// 证书管理（方向2扩展）
		certHandler := NewCertificateHandler(r.services.Certificate)
		certHandler.RegisterRoutes(v1)
	}
}

// API 路由汇总（方便查阅）
//
// Auth:
//   POST   /api/v1/auth/login          登录
//   POST   /api/v1/auth/logout         登出
//   GET    /api/v1/auth/me             获取当前用户
//   PUT    /api/v1/auth/password       修改密码
//
// Inbounds:
//   GET    /api/v1/inbounds            获取入站规则列表
//   GET    /api/v1/inbounds/:id        获取入站规则详情
//   POST   /api/v1/inbounds            创建入站规则
//   PUT    /api/v1/inbounds/:id        更新入站规则
//   DELETE /api/v1/inbounds/:id        删除入站规则
//   POST   /api/v1/inbounds/:id/reset-traffic  重置流量
//
// Clients (Direction 2: Multi-credential):
//   GET    /api/v1/inbounds/:id/clients            获取客户端列表
//   POST   /api/v1/inbounds/:id/clients            添加客户端
//   PUT    /api/v1/inbounds/:id/clients/:clientId  更新客户端
//   DELETE /api/v1/inbounds/:id/clients/:clientId  删除客户端
//
// System:
//   GET    /api/v1/system/status       获取系统状态
//   POST   /api/v1/system/xray/restart 重启 Xray
//   GET    /api/v1/system/xray/version 获取 Xray 版本
//
// Settings:
//   GET    /api/v1/settings            获取所有设置
//   PUT    /api/v1/settings            更新设置
//   POST   /api/v1/settings/reset      重置设置
//
// Certificates (Direction 2):
//   GET    /api/v1/certificates        获取证书列表
//   GET    /api/v1/certificates/:id    获取证书详情
//   POST   /api/v1/certificates        上传证书
//   PUT    /api/v1/certificates/:id    更新证书
//   DELETE /api/v1/certificates/:id    删除证书
