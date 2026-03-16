package handler

import (
	"strconv"

	"Rx-ui/internal/model"
	"Rx-ui/internal/service"

	"github.com/gin-gonic/gin"
)

// InboundHandler 入站规则处理器
type InboundHandler struct {
	inboundService *service.InboundService
}

// NewInboundHandler 创建入站规则处理器
func NewInboundHandler(inboundService *service.InboundService) *InboundHandler {
	return &InboundHandler{inboundService: inboundService}
}

// RegisterRoutes 注册路由
func (h *InboundHandler) RegisterRoutes(r *gin.RouterGroup) {
	inbounds := r.Group("/inbounds")
	{
		inbounds.GET("", h.List)
		inbounds.GET("/:id", h.Get)
		inbounds.POST("", h.Create)
		inbounds.PUT("/:id", h.Update)
		inbounds.DELETE("/:id", h.Delete)
		inbounds.POST("/:id/reset-traffic", h.ResetTraffic)

		// 客户端管理（方向2：多凭证）
		inbounds.GET("/:id/clients", h.ListClients)
		inbounds.POST("/:id/clients", h.AddClient)
		inbounds.PUT("/:id/clients/:clientId", h.UpdateClient)
		inbounds.DELETE("/:id/clients/:clientId", h.DeleteClient)
	}
}

// List 获取所有入站规则
// @Summary 获取入站规则列表
// @Tags Inbounds
// @Produce json
// @Success 200 {object} Response{data=[]model.Inbound}
// @Router /api/v1/inbounds [get]
func (h *InboundHandler) List(c *gin.Context) {
	inbounds, err := h.inboundService.GetAll()
	if err != nil {
		ErrorFromErr(c, err)
		return
	}
	Success(c, inbounds)
}

// Get 获取单个入站规则
// @Summary 获取入站规则详情
// @Tags Inbounds
// @Produce json
// @Param id path int true "入站规则ID"
// @Success 200 {object} Response{data=model.Inbound}
// @Failure 404 {object} Response
// @Router /api/v1/inbounds/{id} [get]
func (h *InboundHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	inbound, err := h.inboundService.GetByID(id)
	if err != nil {
		NotFound(c, "入站规则不存在")
		return
	}
	Success(c, inbound)
}

// CreateInboundRequest 创建入站规则请求
type CreateInboundRequest struct {
	Remark         string         `json:"remark"`
	Enable         bool           `json:"enable"`
	Listen         string         `json:"listen"`
	Port           int            `json:"port" binding:"required"`
	Protocol       model.Protocol `json:"protocol" binding:"required"`
	Settings       string         `json:"settings"`
	StreamSettings string         `json:"streamSettings"`
	Sniffing       string         `json:"sniffing"`
	Tag            string         `json:"tag"`
	Total          int64          `json:"total"`
	ExpiryTime     int64          `json:"expiryTime"`
	CertificateID  *int           `json:"certificateId"`
}

// Create 创建入站规则
// @Summary 创建入站规则
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param request body CreateInboundRequest true "入站规则信息"
// @Success 201 {object} Response{data=model.Inbound}
// @Failure 400 {object} Response
// @Router /api/v1/inbounds [post]
func (h *InboundHandler) Create(c *gin.Context) {
	var req CreateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	inbound := &model.Inbound{
		Remark:         req.Remark,
		Enable:         req.Enable,
		Listen:         req.Listen,
		Port:           req.Port,
		Protocol:       req.Protocol,
		Settings:       req.Settings,
		StreamSettings: req.StreamSettings,
		Sniffing:       req.Sniffing,
		Tag:            req.Tag,
		Total:          req.Total,
		ExpiryTime:     req.ExpiryTime,
		CertificateID:  req.CertificateID,
	}

	if err := h.inboundService.Create(inbound); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, inbound)
}

// Update 更新入站规则
// @Summary 更新入站规则
// @Tags Inbounds
// @Accept json
// @Produce json
// @Param id path int true "入站规则ID"
// @Param request body CreateInboundRequest true "入站规则信息"
// @Success 200 {object} Response{data=model.Inbound}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /api/v1/inbounds/{id} [put]
func (h *InboundHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	var req CreateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	inbound := &model.Inbound{
		ID:             id,
		Remark:         req.Remark,
		Enable:         req.Enable,
		Listen:         req.Listen,
		Port:           req.Port,
		Protocol:       req.Protocol,
		Settings:       req.Settings,
		StreamSettings: req.StreamSettings,
		Sniffing:       req.Sniffing,
		Tag:            req.Tag,
		Total:          req.Total,
		ExpiryTime:     req.ExpiryTime,
		CertificateID:  req.CertificateID,
	}

	if err := h.inboundService.Update(inbound); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, inbound)
}

// Delete 删除入站规则
// @Summary 删除入站规则
// @Tags Inbounds
// @Produce json
// @Param id path int true "入站规则ID"
// @Success 204
// @Failure 404 {object} Response
// @Router /api/v1/inbounds/{id} [delete]
func (h *InboundHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	if err := h.inboundService.Delete(id); err != nil {
		NotFound(c, "入站规则不存在")
		return
	}

	NoContent(c)
}

// ResetTraffic 重置流量统计
// @Summary 重置入站规则流量
// @Tags Inbounds
// @Produce json
// @Param id path int true "入站规则ID"
// @Success 200 {object} Response
// @Router /api/v1/inbounds/{id}/reset-traffic [post]
func (h *InboundHandler) ResetTraffic(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	if err := h.inboundService.ResetTraffic(id); err != nil {
		ErrorFromErr(c, err)
		return
	}

	SuccessMsg(c, "流量已重置", nil)
}

// ===== 客户端管理（方向2：多凭证）=====

// ListClients 获取入站规则下的所有客户端
// @Summary 获取客户端列表
// @Tags Clients
// @Produce json
// @Param id path int true "入站规则ID"
// @Success 200 {object} Response{data=[]model.Client}
// @Router /api/v1/inbounds/{id}/clients [get]
func (h *InboundHandler) ListClients(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	clients, err := h.inboundService.GetClientsForInbound(id)
	if err != nil {
		ErrorFromErr(c, err)
		return
	}

	Success(c, clients)
}

// CreateClientRequest 创建客户端请求
type CreateClientRequest struct {
	Email         string `json:"email" binding:"required"`
	UUID          string `json:"uuid"`
	Password      string `json:"password"`
	Flow          string `json:"flow"`
	Enable        bool   `json:"enable"`
	Total         int64  `json:"total"`
	ExpiryTime    int64  `json:"expiryTime"`
	CertificateID *int   `json:"certificateId"`
}

// AddClient 添加客户端
// @Summary 添加客户端
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path int true "入站规则ID"
// @Param request body CreateClientRequest true "客户端信息"
// @Success 201 {object} Response{data=model.Client}
// @Router /api/v1/inbounds/{id}/clients [post]
func (h *InboundHandler) AddClient(c *gin.Context) {
	inboundID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	client := &model.Client{
		InboundID:     inboundID,
		Email:         req.Email,
		UUID:          req.UUID,
		Password:      req.Password,
		Flow:          req.Flow,
		Enable:        req.Enable,
		Total:         req.Total,
		ExpiryTime:    req.ExpiryTime,
		CertificateID: req.CertificateID,
	}

	if err := h.inboundService.AddClient(client); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, client)
}

// UpdateClient 更新客户端
// @Summary 更新客户端
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path int true "入站规则ID"
// @Param clientId path int true "客户端ID"
// @Param request body CreateClientRequest true "客户端信息"
// @Success 200 {object} Response{data=model.Client}
// @Router /api/v1/inbounds/{id}/clients/{clientId} [put]
func (h *InboundHandler) UpdateClient(c *gin.Context) {
	clientID, err := strconv.Atoi(c.Param("clientId"))
	if err != nil {
		BadRequest(c, "无效的客户端ID")
		return
	}

	inboundID, _ := strconv.Atoi(c.Param("id"))

	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	client := &model.Client{
		ID:            clientID,
		InboundID:     inboundID,
		Email:         req.Email,
		UUID:          req.UUID,
		Password:      req.Password,
		Flow:          req.Flow,
		Enable:        req.Enable,
		Total:         req.Total,
		ExpiryTime:    req.ExpiryTime,
		CertificateID: req.CertificateID,
	}

	if err := h.inboundService.UpdateClient(client); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, client)
}

// DeleteClient 删除客户端
// @Summary 删除客户端
// @Tags Clients
// @Produce json
// @Param id path int true "入站规则ID"
// @Param clientId path int true "客户端ID"
// @Success 204
// @Router /api/v1/inbounds/{id}/clients/{clientId} [delete]
func (h *InboundHandler) DeleteClient(c *gin.Context) {
	clientID, err := strconv.Atoi(c.Param("clientId"))
	if err != nil {
		BadRequest(c, "无效的客户端ID")
		return
	}

	if err := h.inboundService.DeleteClient(clientID); err != nil {
		ErrorFromErr(c, err)
		return
	}

	NoContent(c)
}
