package model

import "time"

// Protocol 支持的协议类型
type Protocol string

const (
	ProtocolVMess       Protocol = "vmess"
	ProtocolVLESS       Protocol = "vless"
	ProtocolTrojan      Protocol = "trojan"
	ProtocolShadowsocks Protocol = "shadowsocks"
	ProtocolDokodemo    Protocol = "dokodemo-door"
	ProtocolHTTP        Protocol = "http"
	ProtocolSOCKS       Protocol = "socks"
)

// Inbound 入站规则（核心模型）
type Inbound struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int       `json:"userId" gorm:"index"`          // 关联管理员
	Remark     string    `json:"remark" gorm:"size:256"`       // 备注名称
	Enable     bool      `json:"enable" gorm:"default:true"`   // 是否启用
	Listen     string    `json:"listen" gorm:"size:64"`        // 监听地址
	Port       int       `json:"port" gorm:"uniqueIndex"`      // 监听端口
	Protocol   Protocol  `json:"protocol" gorm:"size:32"`      // 协议类型
	Tag        string    `json:"tag" gorm:"uniqueIndex;size:64"` // Xray tag

	// 协议配置（JSON字符串）
	Settings       string `json:"settings" gorm:"type:text"`       // 协议设置
	StreamSettings string `json:"streamSettings" gorm:"type:text"` // 传输设置
	Sniffing       string `json:"sniffing" gorm:"type:text"`       // 嗅探设置

	// 流量统计
	Up    int64 `json:"up"`    // 上行流量（字节）
	Down  int64 `json:"down"`  // 下行流量（字节）
	Total int64 `json:"total"` // 流量限制（0=不限制）

	// 有效期
	ExpiryTime int64 `json:"expiryTime"` // 到期时间戳（0=永久）

	// 证书关联（方向2扩展）
	CertificateID *int `json:"certificateId" gorm:"index"` // 关联证书（可选）

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Inbound) TableName() string {
	return "inbounds"
}

// IsExpired 检查是否已过期
func (i *Inbound) IsExpired() bool {
	if i.ExpiryTime == 0 {
		return false
	}
	return time.Now().Unix() > i.ExpiryTime
}

// IsTrafficExceeded 检查流量是否超限
func (i *Inbound) IsTrafficExceeded() bool {
	if i.Total == 0 {
		return false
	}
	return (i.Up + i.Down) >= i.Total
}

// Client 入站规则下的客户端（用户凭证）
// 支持同一个 Inbound 下多个客户端，各自独立统计
type Client struct {
	ID         int      `json:"id" gorm:"primaryKey;autoIncrement"`
	InboundID  int      `json:"inboundId" gorm:"index"`           // 关联入站规则
	Email      string   `json:"email" gorm:"size:128;index"`      // 客户端标识（用于流量统计）
	UUID       string   `json:"uuid" gorm:"size:64"`              // VMess/VLESS UUID
	Password   string   `json:"password" gorm:"size:128"`         // Trojan/SS 密码
	Flow       string   `json:"flow" gorm:"size:32"`              // VLESS flow
	Enable     bool     `json:"enable" gorm:"default:true"`       // 是否启用

	// 独立流量统计
	Up    int64 `json:"up"`
	Down  int64 `json:"down"`
	Total int64 `json:"total"` // 流量限制（0=不限制）

	// 独立有效期
	ExpiryTime int64 `json:"expiryTime"` // 到期时间戳（0=跟随Inbound）

	// 证书关联（方向2扩展：同端口不同凭证可用不同证书）
	CertificateID *int `json:"certificateId" gorm:"index"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Client) TableName() string {
	return "clients"
}

// IsExpired 检查客户端是否已过期
func (c *Client) IsExpired() bool {
	if c.ExpiryTime == 0 {
		return false
	}
	return time.Now().Unix() > c.ExpiryTime
}

// IsTrafficExceeded 检查客户端流量是否超限
func (c *Client) IsTrafficExceeded() bool {
	if c.Total == 0 {
		return false
	}
	return (c.Up + c.Down) >= c.Total
}
