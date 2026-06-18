package model

import "time"

// ChainedProxy 连锁（上游）代理
// 每条记录对应一个上游代理服务器，可被客户端选择以实现「代理链」转发。
// 在 Xray 中会生成一个 outbound（tag = proxy-<id>），
// 被分配该代理的客户端流量经路由规则导向此 outbound。
type ChainedProxy struct {
	ID       int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Remark   string `json:"remark" gorm:"size:128"`              // 备注名称
	Protocol string `json:"protocol" gorm:"size:16;default:socks"` // 上游协议：socks | http
	Host     string `json:"host" gorm:"size:256"`                // 主机地址
	Port     int    `json:"port"`                                // 端口
	Username string `json:"username" gorm:"size:256"`            // 认证用户名（可选）
	Password string `json:"password" gorm:"size:256"`            // 认证密码（可选）
	Enable   bool   `json:"enable" gorm:"default:true"`          // 是否启用

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ChainedProxy) TableName() string {
	return "chained_proxies"
}
