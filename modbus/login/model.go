package login

import (
	"log"
	"modbus/gorm"
)

// User 用户表模型
type User struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"type:varchar(64);uniqueIndex;not null"`
	PasswordHash string `gorm:"type:varchar(256);not null"`
}

// TableName 设置表名
func (User) TableName() string {
	return "users"
}

// InitDB 自动迁移用户表
func InitDB() {
	err := gorm.DB.AutoMigrate(&User{})
	if err != nil {
		log.Printf("⚠️ [Login] 数据库表迁移失败: %v", err)
	} else {
		log.Print("✅ [Login] 数据库表迁移成功")
	}
}
