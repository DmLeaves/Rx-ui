package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"Rx-ui/internal/config"
	"Rx-ui/internal/model"
	"Rx-ui/internal/repository"
)

var (
	ErrXrayNotRunning = errors.New("xray is not running")
	ErrXrayStartFailed = errors.New("failed to start xray")
)

// XrayService Xray进程管理服务
type XrayService struct {
	config      *config.XrayConfig
	inboundRepo repository.InboundRepository
	clientRepo  repository.ClientRepository
	certRepo    repository.CertificateRepository

	process *os.Process
	mu      sync.RWMutex
}

// NewXrayService 创建Xray服务
func NewXrayService(
	cfg *config.XrayConfig,
	inboundRepo repository.InboundRepository,
	clientRepo repository.ClientRepository,
	certRepo repository.CertificateRepository,
) *XrayService {
	return &XrayService{
		config:      cfg,
		inboundRepo: inboundRepo,
		clientRepo:  clientRepo,
		certRepo:    certRepo,
	}
}

// IsRunning 检查Xray是否正在运行
func (s *XrayService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.process != nil
}

// Start 启动Xray
func (s *XrayService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.process != nil {
		return nil // 已经在运行
	}

	// 生成配置文件
	if err := s.generateConfig(); err != nil {
		return err
	}

	// 启动进程
	cmd := exec.Command(s.config.BinPath, "-config", s.config.ConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return ErrXrayStartFailed
	}

	s.process = cmd.Process
	return nil
}

// Stop 停止Xray
func (s *XrayService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.process == nil {
		return nil
	}

	if err := s.process.Kill(); err != nil {
		return err
	}

	s.process = nil
	return nil
}

// Restart 重启Xray
func (s *XrayService) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}

	// 延迟重启
	time.Sleep(time.Duration(s.config.RestartDelay) * time.Second)

	return s.Start()
}

// generateConfig 生成Xray配置文件
func (s *XrayService) generateConfig() error {
	config, err := s.buildConfig()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.config.ConfigPath, data, 0644)
}

// buildConfig 构建Xray配置
func (s *XrayService) buildConfig() (map[string]interface{}, error) {
	inbounds, err := s.inboundRepo.FindEnabled()
	if err != nil {
		return nil, err
	}

	inboundConfigs := make([]map[string]interface{}, 0)
	for _, inbound := range inbounds {
		cfg, err := s.buildInboundConfig(inbound)
		if err != nil {
			continue // 跳过配置错误的入站规则
		}
		inboundConfigs = append(inboundConfigs, cfg)
	}

	// 基础配置
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds":  inboundConfigs,
		"outbounds": []map[string]interface{}{
			{
				"protocol": "freedom",
				"tag":      "direct",
			},
			{
				"protocol": "blackhole",
				"tag":      "blocked",
			},
		},
		"routing": map[string]interface{}{
			"rules": []map[string]interface{}{
				{
					"type":        "field",
					"outboundTag": "blocked",
					"ip":          []string{"geoip:private"},
				},
			},
		},
		"stats": map[string]interface{}{},
		"api": map[string]interface{}{
			"tag": "api",
			"services": []string{
				"HandlerService",
				"LoggerService",
				"StatsService",
			},
		},
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{
				"0": map[string]interface{}{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]interface{}{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
	}

	return config, nil
}

// buildInboundConfig 构建单个入站配置
func (s *XrayService) buildInboundConfig(inbound *model.Inbound) (map[string]interface{}, error) {
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return nil, err
	}

	var streamSettings map[string]interface{}
	if inbound.StreamSettings != "" {
		if err := json.Unmarshal([]byte(inbound.StreamSettings), &streamSettings); err != nil {
			return nil, err
		}
	}

	// 获取该入站规则下的所有客户端
	clients, _ := s.clientRepo.FindByInboundID(inbound.ID)
	if len(clients) > 0 {
		// 将客户端添加到 settings
		clientConfigs := make([]map[string]interface{}, 0)
		for _, client := range clients {
			if !client.Enable || client.IsExpired() || client.IsTrafficExceeded() {
				continue
			}
			clientCfg := map[string]interface{}{
				"email": client.Email,
			}
			if client.UUID != "" {
				clientCfg["id"] = client.UUID
			}
			if client.Password != "" {
				clientCfg["password"] = client.Password
			}
			if client.Flow != "" {
				clientCfg["flow"] = client.Flow
			}
			clientConfigs = append(clientConfigs, clientCfg)
		}
		if len(clientConfigs) > 0 {
			settings["clients"] = clientConfigs
		}
	}

	config := map[string]interface{}{
		"listen":   inbound.Listen,
		"port":     inbound.Port,
		"protocol": string(inbound.Protocol),
		"settings": settings,
		"tag":      inbound.Tag,
	}

	if streamSettings != nil {
		config["streamSettings"] = streamSettings
	}

	if inbound.Sniffing != "" {
		var sniffing map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.Sniffing), &sniffing); err == nil {
			config["sniffing"] = sniffing
		}
	}

	return config, nil
}

// GetTrafficStats 获取流量统计
func (s *XrayService) GetTrafficStats(ctx context.Context) (map[string]*TrafficStats, error) {
	// TODO: 通过 Xray API 获取流量统计
	return nil, nil
}

// TrafficStats 流量统计
type TrafficStats struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}
