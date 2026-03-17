package repository

import (
	"rxui/internal/model"
)

// UserRepository 用户数据访问接口
type UserRepository interface {
	GetAll() ([]*model.User, error)
	FindByID(id int) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Create(user *model.User) error
	Update(user *model.User) error
	Delete(id int) error
	GetFirst() (*model.User, error)
}

// InboundRepository 入站规则数据访问接口
type InboundRepository interface {
	FindAll() ([]*model.Inbound, error)
	FindByID(id int) (*model.Inbound, error)
	FindByUserID(userID int) ([]*model.Inbound, error)
	FindByPort(port int) (*model.Inbound, error)
	FindByTag(tag string) (*model.Inbound, error)
	FindEnabled() ([]*model.Inbound, error)
	Create(inbound *model.Inbound) error
	Update(inbound *model.Inbound) error
	Delete(id int) error
	UpdateTraffic(id int, up, down int64) error
	ResetTraffic(id int) error
}

// ClientRepository 客户端（用户凭证）数据访问接口
type ClientRepository interface {
	FindAll() ([]*model.Client, error)
	FindByID(id int) (*model.Client, error)
	FindByInboundID(inboundID int) ([]*model.Client, error)
	FindByEmail(email string) (*model.Client, error)
	FindByUUID(uuid string) (*model.Client, error)
	Create(client *model.Client) error
	Update(client *model.Client) error
	Delete(id int) error
	UpdateTraffic(id int, up, down int64) error
	ResetTraffic(id int) error
}

// CertificateRepository 证书数据访问接口
type CertificateRepository interface {
	FindAll() ([]*model.Certificate, error)
	FindByID(id int) (*model.Certificate, error)
	FindByDomain(domain string) (*model.Certificate, error)
	FindExpiring(days int) ([]*model.Certificate, error) // 查找即将过期的证书
	Create(cert *model.Certificate) error
	Update(cert *model.Certificate) error
	Delete(id int) error
}

// SettingRepository 设置数据访问接口
type SettingRepository interface {
	Get(key model.SettingKey) (string, error)
	Set(key model.SettingKey, value string) error
	GetAll() (map[model.SettingKey]string, error)
	SetAll(settings map[model.SettingKey]string) error
	Delete(key model.SettingKey) error
	ResetAll() error
}

// TrafficUpdate 流量更新
type TrafficUpdate struct {
	Tag  string
	Up   int64
	Down int64
}

// Repositories 聚合所有 Repository（依赖注入容器）
type Repositories struct {
	User        UserRepository
	Inbound     InboundRepository
	Client      ClientRepository
	Certificate CertificateRepository
	Setting     SettingRepository
}
