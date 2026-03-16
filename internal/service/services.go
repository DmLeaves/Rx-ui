package service

import (
	"Rx-ui/internal/config"
	"Rx-ui/internal/repository"
)

// Services 聚合所有服务（依赖注入容器）
type Services struct {
	User        *UserService
	Inbound     *InboundService
	Xray        *XrayService
	Setting     *SettingService
	Certificate *CertificateService
}

// NewServices 创建所有服务
func NewServices(cfg *config.Config, repos *repository.Repositories) *Services {
	// 创建 Xray 服务（被其他服务依赖）
	xrayService := NewXrayService(
		&cfg.Xray,
		repos.Inbound,
		repos.Client,
		repos.Certificate,
	)

	return &Services{
		User:        NewUserService(repos.User),
		Inbound:     NewInboundService(repos.Inbound, repos.Client, xrayService),
		Xray:        xrayService,
		Setting:     NewSettingService(repos.Setting),
		Certificate: NewCertificateService(repos.Certificate),
	}
}
