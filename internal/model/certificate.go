package model

import "time"

// Certificate TLS证书管理（方向2扩展）
type Certificate struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Domain      string    `json:"domain" gorm:"size:256;index"`     // 域名
	CertFile    string    `json:"certFile" gorm:"size:512"`         // 证书文件路径
	KeyFile     string    `json:"keyFile" gorm:"size:512"`          // 私钥文件路径
	CertContent string    `json:"-" gorm:"type:text"`               // 证书内容（可选，直接存储）
	KeyContent  string    `json:"-" gorm:"type:text"`               // 私钥内容（可选，直接存储）
	Issuer      string    `json:"issuer" gorm:"size:128"`           // 签发者
	ExpiresAt   time.Time `json:"expiresAt"`                        // 过期时间
	AutoRenew   bool      `json:"autoRenew" gorm:"default:false"`   // 是否自动续期
	Remark      string    `json:"remark" gorm:"size:256"`           // 备注

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Certificate) TableName() string {
	return "certificates"
}

// IsExpired 检查证书是否已过期
func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// DaysUntilExpiry 距离过期还有多少天
func (c *Certificate) DaysUntilExpiry() int {
	duration := time.Until(c.ExpiresAt)
	return int(duration.Hours() / 24)
}
