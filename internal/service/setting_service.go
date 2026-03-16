package service

import (
	"encoding/json"
	"strconv"
	"time"

	"Rx-ui/internal/model"
	"Rx-ui/internal/repository"
)

// SettingService 系统设置服务
type SettingService struct {
	repo repository.SettingRepository
}

// NewSettingService 创建设置服务
func NewSettingService(repo repository.SettingRepository) *SettingService {
	return &SettingService{repo: repo}
}

// GetString 获取字符串设置
func (s *SettingService) GetString(key model.SettingKey) (string, error) {
	value, err := s.repo.Get(key)
	if err != nil {
		// 返回默认值
		if defaultVal, ok := model.DefaultSettings[key]; ok {
			return defaultVal, nil
		}
		return "", err
	}
	return value, nil
}

// SetString 设置字符串值
func (s *SettingService) SetString(key model.SettingKey, value string) error {
	return s.repo.Set(key, value)
}

// GetInt 获取整数设置
func (s *SettingService) GetInt(key model.SettingKey) (int, error) {
	str, err := s.GetString(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}

// SetInt 设置整数值
func (s *SettingService) SetInt(key model.SettingKey, value int) error {
	return s.SetString(key, strconv.Itoa(value))
}

// GetBool 获取布尔设置
func (s *SettingService) GetBool(key model.SettingKey) (bool, error) {
	str, err := s.GetString(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(str)
}

// SetBool 设置布尔值
func (s *SettingService) SetBool(key model.SettingKey, value bool) error {
	return s.SetString(key, strconv.FormatBool(value))
}

// GetJSON 获取JSON设置
func (s *SettingService) GetJSON(key model.SettingKey, dest interface{}) error {
	str, err := s.GetString(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), dest)
}

// SetJSON 设置JSON值
func (s *SettingService) SetJSON(key model.SettingKey, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.SetString(key, string(data))
}

// GetAll 获取所有设置
func (s *SettingService) GetAll() (map[model.SettingKey]string, error) {
	return s.repo.GetAll()
}

// Reset 重置所有设置为默认值
func (s *SettingService) Reset() error {
	return s.repo.ResetAll()
}

// ===== 便捷方法 =====

// GetWebPort 获取Web端口
func (s *SettingService) GetWebPort() (int, error) {
	return s.GetInt(model.SettingKeyWebPort)
}

// SetWebPort 设置Web端口
func (s *SettingService) SetWebPort(port int) error {
	return s.SetInt(model.SettingKeyWebPort, port)
}

// GetWebListen 获取监听地址
func (s *SettingService) GetWebListen() (string, error) {
	return s.GetString(model.SettingKeyWebListen)
}

// GetWebBasePath 获取基础路径
func (s *SettingService) GetWebBasePath() (string, error) {
	basePath, err := s.GetString(model.SettingKeyWebBasePath)
	if err != nil {
		return "/", err
	}
	// 确保路径格式正确
	if basePath == "" {
		return "/", nil
	}
	if basePath[0] != '/' {
		basePath = "/" + basePath
	}
	if basePath[len(basePath)-1] != '/' {
		basePath = basePath + "/"
	}
	return basePath, nil
}

// GetTimeLocation 获取时区
func (s *SettingService) GetTimeLocation() (*time.Location, error) {
	loc, err := s.GetString(model.SettingKeyTimeLocation)
	if err != nil {
		return time.Local, err
	}
	return time.LoadLocation(loc)
}

// GetSecret 获取Session密钥
func (s *SettingService) GetSecret() ([]byte, error) {
	secret, err := s.GetString(model.SettingKeySecret)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}

// GetCertFile 获取证书文件路径
func (s *SettingService) GetCertFile() (string, error) {
	return s.GetString(model.SettingKeyWebCertFile)
}

// GetKeyFile 获取私钥文件路径
func (s *SettingService) GetKeyFile() (string, error) {
	return s.GetString(model.SettingKeyWebKeyFile)
}

// ===== 前端CDN配置（方向1）=====

// GetFrontendMode 获取前端资源模式
func (s *SettingService) GetFrontendMode() (string, error) {
	return s.GetString(model.SettingKeyFrontendMode)
}

// SetFrontendMode 设置前端资源模式
func (s *SettingService) SetFrontendMode(mode string) error {
	return s.SetString(model.SettingKeyFrontendMode, mode)
}

// GetCDNProviders 获取CDN提供商列表
func (s *SettingService) GetCDNProviders() ([]string, error) {
	var providers []string
	err := s.GetJSON(model.SettingKeyCDNProviders, &providers)
	return providers, err
}

// SetCDNProviders 设置CDN提供商列表
func (s *SettingService) SetCDNProviders(providers []string) error {
	return s.SetJSON(model.SettingKeyCDNProviders, providers)
}
