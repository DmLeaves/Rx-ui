package handler

import (
	"strconv"
	"time"

	"Rx-ui/internal/model"
	"Rx-ui/internal/service"

	"github.com/gin-gonic/gin"
)

// CertificateHandler 证书处理器（方向2扩展）
type CertificateHandler struct {
	certService *service.CertificateService
}

// NewCertificateHandler 创建证书处理器
func NewCertificateHandler(certService *service.CertificateService) *CertificateHandler {
	return &CertificateHandler{certService: certService}
}

// RegisterRoutes 注册路由
func (h *CertificateHandler) RegisterRoutes(r *gin.RouterGroup) {
	certs := r.Group("/certificates")
	{
		certs.GET("", h.List)
		certs.GET("/:id", h.Get)
		certs.POST("", h.Create)
		certs.PUT("/:id", h.Update)
		certs.DELETE("/:id", h.Delete)
		certs.GET("/expiring", h.GetExpiring)
	}
}

// List 获取所有证书
// @Summary 获取证书列表
// @Tags Certificates
// @Produce json
// @Success 200 {object} Response{data=[]model.Certificate}
// @Router /api/v1/certificates [get]
func (h *CertificateHandler) List(c *gin.Context) {
	certs, err := h.certService.GetAll()
	if err != nil {
		ErrorFromErr(c, err)
		return
	}
	Success(c, certs)
}

// Get 获取单个证书
// @Summary 获取证书详情
// @Tags Certificates
// @Produce json
// @Param id path int true "证书ID"
// @Success 200 {object} Response{data=model.Certificate}
// @Failure 404 {object} Response
// @Router /api/v1/certificates/{id} [get]
func (h *CertificateHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	cert, err := h.certService.GetByID(id)
	if err != nil {
		NotFound(c, "证书不存在")
		return
	}
	Success(c, cert)
}

// CreateCertificateRequest 创建证书请求
type CreateCertificateRequest struct {
	Domain      string    `json:"domain" binding:"required"`
	CertFile    string    `json:"certFile"`
	KeyFile     string    `json:"keyFile"`
	CertContent string    `json:"certContent"`
	KeyContent  string    `json:"keyContent"`
	Remark      string    `json:"remark"`
	AutoRenew   bool      `json:"autoRenew"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// Create 创建证书
// @Summary 创建证书
// @Tags Certificates
// @Accept json
// @Produce json
// @Param request body CreateCertificateRequest true "证书信息"
// @Success 201 {object} Response{data=model.Certificate}
// @Failure 400 {object} Response
// @Router /api/v1/certificates [post]
func (h *CertificateHandler) Create(c *gin.Context) {
	var req CreateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	cert := &model.Certificate{
		Domain:      req.Domain,
		CertFile:    req.CertFile,
		KeyFile:     req.KeyFile,
		CertContent: req.CertContent,
		KeyContent:  req.KeyContent,
		Remark:      req.Remark,
		AutoRenew:   req.AutoRenew,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := h.certService.Create(cert); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, cert)
}

// Update 更新证书
// @Summary 更新证书
// @Tags Certificates
// @Accept json
// @Produce json
// @Param id path int true "证书ID"
// @Param request body CreateCertificateRequest true "证书信息"
// @Success 200 {object} Response{data=model.Certificate}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /api/v1/certificates/{id} [put]
func (h *CertificateHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	var req CreateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	cert := &model.Certificate{
		ID:          id,
		Domain:      req.Domain,
		CertFile:    req.CertFile,
		KeyFile:     req.KeyFile,
		CertContent: req.CertContent,
		KeyContent:  req.KeyContent,
		Remark:      req.Remark,
		AutoRenew:   req.AutoRenew,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := h.certService.Update(cert); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, cert)
}

// Delete 删除证书
// @Summary 删除证书
// @Tags Certificates
// @Produce json
// @Param id path int true "证书ID"
// @Success 204
// @Failure 404 {object} Response
// @Router /api/v1/certificates/{id} [delete]
func (h *CertificateHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "无效的ID")
		return
	}

	if err := h.certService.Delete(id); err != nil {
		NotFound(c, "证书不存在")
		return
	}

	NoContent(c)
}

// GetExpiring 获取即将过期的证书
// @Summary 获取即将过期的证书
// @Tags Certificates
// @Produce json
// @Param days query int false "天数（默认30）"
// @Success 200 {object} Response{data=[]model.Certificate}
// @Router /api/v1/certificates/expiring [get]
func (h *CertificateHandler) GetExpiring(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	certs, err := h.certService.GetExpiring(days)
	if err != nil {
		ErrorFromErr(c, err)
		return
	}

	Success(c, certs)
}
