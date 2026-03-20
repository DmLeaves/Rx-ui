package model

import "time"

// ControlAuditLog 记录 AI 控制 API 的调用审计
// 不存敏感签名，仅记录调用方、动作、结果与摘要。
type ControlAuditLog struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientID   string    `json:"clientId" gorm:"size:128;index"`
	Path       string    `json:"path" gorm:"size:64"`
	Action     string    `json:"action" gorm:"size:128;index"`
	QueryOnly  bool      `json:"queryOnly"`
	Success    bool      `json:"success" gorm:"index"`
	ErrorCode  string    `json:"errorCode" gorm:"size:64"`
	ErrorMsg   string    `json:"errorMsg" gorm:"size:512"`
	RequestID  string    `json:"requestId" gorm:"size:64;index"`
	BodySHA256 string    `json:"bodySha256" gorm:"size:64"`
	DurationMs int64     `json:"durationMs"`
	CreatedAt  time.Time `json:"createdAt" gorm:"index"`
}

func (ControlAuditLog) TableName() string {
	return "control_audit_logs"
}
