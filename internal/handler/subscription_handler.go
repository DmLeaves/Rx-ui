package handler

import (
	"encoding/base64"
	"fmt"
	"strings"

	"Rx-ui/internal/service"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandler 订阅处理器
type SubscriptionHandler struct {
	inboundService *service.InboundService
}

// NewSubscriptionHandler 创建订阅处理器
func NewSubscriptionHandler(inboundService *service.InboundService) *SubscriptionHandler {
	return &SubscriptionHandler{inboundService: inboundService}
}

// RegisterRoutes 注册路由
func (h *SubscriptionHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/sub", h.GetSubscription)
	r.GET("/sub/links", h.GetLinks)
}

// GetSubscription 获取订阅内容（Base64 编码）
// @Summary 获取订阅
// @Tags Subscription
// @Produce text/plain
// @Param host query string false "自定义服务器地址"
// @Success 200 {string} string
// @Router /api/v1/sub [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	host := c.Query("host")
	if host == "" {
		host = c.Request.Host
		// 去掉端口
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
	}

	links, err := h.generateLinks(host)
	if err != nil {
		ServerError(c, "生成订阅失败")
		return
	}

	content := strings.Join(links, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Subscription-Userinfo", "upload=0; download=0; total=0; expire=0")
	c.String(200, encoded)
}

// GetLinks 获取订阅链接列表（JSON 格式）
// @Summary 获取订阅链接列表
// @Tags Subscription
// @Produce json
// @Param host query string false "自定义服务器地址"
// @Success 200 {object} Response
// @Router /api/v1/sub/links [get]
func (h *SubscriptionHandler) GetLinks(c *gin.Context) {
	host := c.Query("host")
	if host == "" {
		host = c.Request.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
	}

	links, err := h.generateLinks(host)
	if err != nil {
		ServerError(c, "生成链接失败")
		return
	}

	sub := base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))

	Success(c, gin.H{
		"links": links,
		"sub":   sub,
	})
}

// generateLinks 根据所有启用的入站规则生成链接
func (h *SubscriptionHandler) generateLinks(host string) ([]string, error) {
	inbounds, err := h.inboundService.GetAll()
	if err != nil {
		return nil, err
	}

	var links []string
	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		// 链接生成在 service 层（或直接简单处理）
		// 这里只生成格式标记，实际 link 生成在前端
		link := fmt.Sprintf("# %s (%s:%d)", inbound.Remark, inbound.Protocol, inbound.Port)
		links = append(links, link)
	}

	return links, nil
}
