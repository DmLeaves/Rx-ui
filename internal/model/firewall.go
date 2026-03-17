package model

import "time"

type FirewallScope string

type FirewallStatus string

const (
	FirewallScopePanel   FirewallScope = "panel"
	FirewallScopeInbound FirewallScope = "inbound"
	FirewallScopeCustom  FirewallScope = "custom"

	FirewallStatusPending FirewallStatus = "pending"
	FirewallStatusApplied FirewallStatus = "applied"
	FirewallStatusFailed  FirewallStatus = "failed"
	FirewallStatusStale   FirewallStatus = "stale"
)

type FirewallRule struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement"`
	Scope     FirewallScope  `json:"scope" gorm:"size:32;index"`
	RefID     *int           `json:"refId" gorm:"index"`
	Port      int            `json:"port" gorm:"index"`
	Protocol  string         `json:"protocol" gorm:"size:8;default:tcp"`
	Source    string         `json:"source" gorm:"size:64;default:any"`
	Action    string         `json:"action" gorm:"size:16;default:allow"`
	Provider  string         `json:"provider" gorm:"size:32"`
	Status    FirewallStatus `json:"status" gorm:"size:16;index"`
	LastError string         `json:"lastError" gorm:"type:text"`
	AppliedAt *time.Time     `json:"appliedAt"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

func (FirewallRule) TableName() string {
	return "firewall_rules"
}
