package service

import (
	"errors"

	"rxui/internal/model"
	"rxui/internal/repository"
)

var (
	ErrInboundNotFound  = errors.New("inbound not found")
	ErrPortExists       = errors.New("port already in use")
	ErrTagExists        = errors.New("tag already exists")
	ErrInboundDisabled  = errors.New("inbound is disabled")
	ErrInboundExpired   = errors.New("inbound has expired")
	ErrTrafficExceeded  = errors.New("traffic limit exceeded")
)

// InboundService 入站规则服务
type InboundService struct {
	inboundRepo repository.InboundRepository
	clientRepo  repository.ClientRepository
	xrayService *XrayService
}

// NewInboundService 创建入站规则服务
func NewInboundService(
	inboundRepo repository.InboundRepository,
	clientRepo repository.ClientRepository,
	xrayService *XrayService,
) *InboundService {
	return &InboundService{
		inboundRepo: inboundRepo,
		clientRepo:  clientRepo,
		xrayService: xrayService,
	}
}

// GetAll 获取所有入站规则
func (s *InboundService) GetAll() ([]*model.Inbound, error) {
	return s.inboundRepo.FindAll()
}

// GetByID 根据ID获取入站规则
func (s *InboundService) GetByID(id int) (*model.Inbound, error) {
	inbound, err := s.inboundRepo.FindByID(id)
	if err != nil {
		return nil, ErrInboundNotFound
	}
	return inbound, nil
}

// GetByUserID 获取用户的所有入站规则
func (s *InboundService) GetByUserID(userID int) ([]*model.Inbound, error) {
	return s.inboundRepo.FindByUserID(userID)
}

// GetEnabled 获取所有启用的入站规则
func (s *InboundService) GetEnabled() ([]*model.Inbound, error) {
	return s.inboundRepo.FindEnabled()
}

// Create 创建入站规则
func (s *InboundService) Create(inbound *model.Inbound) error {
	// 检查端口是否已被使用
	existing, _ := s.inboundRepo.FindByPort(inbound.Port)
	if existing != nil {
		return ErrPortExists
	}

	// 检查 tag 是否已存在
	if inbound.Tag != "" {
		existing, _ = s.inboundRepo.FindByTag(inbound.Tag)
		if existing != nil {
			return ErrTagExists
		}
	}

	if err := s.inboundRepo.Create(inbound); err != nil {
		return err
	}

	// 重启 Xray 使配置生效
	return s.xrayService.Restart()
}

// Update 更新入站规则
func (s *InboundService) Update(inbound *model.Inbound) error {
	existing, err := s.inboundRepo.FindByID(inbound.ID)
	if err != nil {
		return ErrInboundNotFound
	}

	// 检查端口冲突（如果端口改变了）
	if inbound.Port != existing.Port {
		conflict, _ := s.inboundRepo.FindByPort(inbound.Port)
		if conflict != nil {
			return ErrPortExists
		}
	}

	if err := s.inboundRepo.Update(inbound); err != nil {
		return err
	}

	return s.xrayService.Restart()
}

// Delete 删除入站规则
func (s *InboundService) Delete(id int) error {
	if err := s.inboundRepo.Delete(id); err != nil {
		return err
	}

	return s.xrayService.Restart()
}

// UpdateTraffic 更新流量统计
func (s *InboundService) UpdateTraffic(id int, up, down int64) error {
	return s.inboundRepo.UpdateTraffic(id, up, down)
}

// ResetTraffic 重置流量统计
func (s *InboundService) ResetTraffic(id int) error {
	return s.inboundRepo.ResetTraffic(id)
}

// CheckStatus 检查入站规则状态
func (s *InboundService) CheckStatus(inbound *model.Inbound) error {
	if !inbound.Enable {
		return ErrInboundDisabled
	}
	if inbound.IsExpired() {
		return ErrInboundExpired
	}
	if inbound.IsTrafficExceeded() {
		return ErrTrafficExceeded
	}
	return nil
}

// GetClientsForInbound 获取入站规则下的所有客户端
func (s *InboundService) GetClientsForInbound(inboundID int) ([]*model.Client, error) {
	return s.clientRepo.FindByInboundID(inboundID)
}

// AddClient 添加客户端到入站规则
func (s *InboundService) AddClient(client *model.Client) error {
	// 确保入站规则存在
	_, err := s.inboundRepo.FindByID(client.InboundID)
	if err != nil {
		return ErrInboundNotFound
	}

	if err := s.clientRepo.Create(client); err != nil {
		return err
	}

	return s.xrayService.Restart()
}

// UpdateClient 更新客户端
func (s *InboundService) UpdateClient(client *model.Client) error {
	if err := s.clientRepo.Update(client); err != nil {
		return err
	}

	return s.xrayService.Restart()
}

// DeleteClient 删除客户端
func (s *InboundService) DeleteClient(id int) error {
	if err := s.clientRepo.Delete(id); err != nil {
		return err
	}

	return s.xrayService.Restart()
}

// UpdateClientTraffic 更新客户端流量
func (s *InboundService) UpdateClientTraffic(id int, up, down int64) error {
	return s.clientRepo.UpdateTraffic(id, up, down)
}
