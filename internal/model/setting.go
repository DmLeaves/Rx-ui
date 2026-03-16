package model

import "time"

// Setting 系统设置（KV存储）
type Setting struct {
	ID    int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Key   string `json:"key" gorm:"uniqueIndex;size:64"`
	Value string `json:"value" gorm:"type:text"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func (Setting) TableName() string {
	return "settings"
}

// SettingKey 预定义的设置项
type SettingKey string

const (
	SettingKeyWebListen     SettingKey = "webListen"
	SettingKeyWebPort       SettingKey = "webPort"
	SettingKeyWebBasePath   SettingKey = "webBasePath"
	SettingKeyWebCertFile   SettingKey = "webCertFile"
	SettingKeyWebKeyFile    SettingKey = "webKeyFile"
	SettingKeySecret        SettingKey = "secret"
	SettingKeyTimeLocation  SettingKey = "timeLocation"
	SettingKeyXrayConfig    SettingKey = "xrayTemplateConfig"

	// 前端CDN设置（方向1扩展）
	SettingKeyFrontendMode  SettingKey = "frontendMode"  // embedded | cdn
	SettingKeyCDNProviders  SettingKey = "cdnProviders"  // JSON数组
)

// DefaultSettings 默认设置值
var DefaultSettings = map[SettingKey]string{
	SettingKeyWebListen:    "",
	SettingKeyWebPort:      "54321",
	SettingKeyWebBasePath:  "/",
	SettingKeyTimeLocation: "Asia/Shanghai",
	SettingKeyFrontendMode: "embedded",
	SettingKeyCDNProviders: "[]",
}
