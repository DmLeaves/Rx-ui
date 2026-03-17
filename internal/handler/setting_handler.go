package handler

import (
	"rxui/internal/model"
	"rxui/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 设置处理器
type SettingHandler struct {
	settingService *service.SettingService
}

// NewSettingHandler 创建设置处理器
func NewSettingHandler(settingService *service.SettingService) *SettingHandler {
	return &SettingHandler{settingService: settingService}
}

// RegisterRoutes 注册路由
func (h *SettingHandler) RegisterRoutes(r *gin.RouterGroup) {
	settings := r.Group("/settings")
	{
		settings.GET("", h.GetAll)
		settings.PUT("", h.Update)
		settings.POST("/reset", h.Reset)
	}
}

// AllSettings 所有设置
type AllSettings struct {
	WebListen    string   `json:"webListen"`
	WebPort      int      `json:"webPort"`
	WebBasePath  string   `json:"webBasePath"`
	WebCertFile  string   `json:"webCertFile"`
	WebKeyFile   string   `json:"webKeyFile"`
	TimeLocation string   `json:"timeLocation"`
	FrontendMode string   `json:"frontendMode"`
	CDNProviders []string `json:"cdnProviders"`
}

// GetAll 获取所有设置
// @Summary 获取所有设置
// @Tags Settings
// @Produce json
// @Success 200 {object} Response{data=AllSettings}
// @Router /api/v1/settings [get]
func (h *SettingHandler) GetAll(c *gin.Context) {
	webPort, _ := h.settingService.GetWebPort()
	webListen, _ := h.settingService.GetWebListen()
	webBasePath, _ := h.settingService.GetWebBasePath()
	certFile, _ := h.settingService.GetCertFile()
	keyFile, _ := h.settingService.GetKeyFile()
	timeLocation, _ := h.settingService.GetString(model.SettingKeyTimeLocation)
	frontendMode, _ := h.settingService.GetFrontendMode()
	cdnProviders, _ := h.settingService.GetCDNProviders()

	settings := AllSettings{
		WebListen:    webListen,
		WebPort:      webPort,
		WebBasePath:  webBasePath,
		WebCertFile:  certFile,
		WebKeyFile:   keyFile,
		TimeLocation: timeLocation,
		FrontendMode: frontendMode,
		CDNProviders: cdnProviders,
	}

	Success(c, settings)
}

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
	WebListen    *string  `json:"webListen"`
	WebPort      *int     `json:"webPort"`
	WebBasePath  *string  `json:"webBasePath"`
	WebCertFile  *string  `json:"webCertFile"`
	WebKeyFile   *string  `json:"webKeyFile"`
	TimeLocation *string  `json:"timeLocation"`
	FrontendMode *string  `json:"frontendMode"`
	CDNProviders []string `json:"cdnProviders"`
}

// Update 更新设置
// @Summary 更新设置
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body UpdateSettingsRequest true "设置信息"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/v1/settings [put]
func (h *SettingHandler) Update(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}

	// 更新各项设置
	if req.WebListen != nil {
		h.settingService.SetString(model.SettingKeyWebListen, *req.WebListen)
	}
	if req.WebPort != nil {
		h.settingService.SetWebPort(*req.WebPort)
	}
	if req.WebBasePath != nil {
		h.settingService.SetString(model.SettingKeyWebBasePath, *req.WebBasePath)
	}
	if req.WebCertFile != nil {
		h.settingService.SetString(model.SettingKeyWebCertFile, *req.WebCertFile)
	}
	if req.WebKeyFile != nil {
		h.settingService.SetString(model.SettingKeyWebKeyFile, *req.WebKeyFile)
	}
	if req.TimeLocation != nil {
		h.settingService.SetString(model.SettingKeyTimeLocation, *req.TimeLocation)
	}
	if req.FrontendMode != nil {
		h.settingService.SetFrontendMode(*req.FrontendMode)
	}
	if req.CDNProviders != nil {
		h.settingService.SetCDNProviders(req.CDNProviders)
	}

	SuccessMsg(c, "设置已更新", nil)
}

// Reset 重置所有设置
// @Summary 重置所有设置
// @Tags Settings
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/settings/reset [post]
func (h *SettingHandler) Reset(c *gin.Context) {
	if err := h.settingService.Reset(); err != nil {
		ServerError(c, "重置设置失败")
		return
	}
	SuccessMsg(c, "设置已重置为默认值", nil)
}
