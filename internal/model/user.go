package model

import "time"

// User 系统管理员账号
type User struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:64"`
	Password  string    `json:"-" gorm:"size:256"` // json忽略密码
	Enable    bool      `json:"enable" gorm:"default:true"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (User) TableName() string {
	return "users"
}
