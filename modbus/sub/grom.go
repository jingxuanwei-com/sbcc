package sub

import (
	"fmt"
	"modbus/gorm"
)

type Sub struct {
	// ID 必须大写开头，GORM 才能识别并迁移
	ID    int64  `gorm:"primaryKey;autoIncrement:false"`
	Token string `gorm:"type:char(32);index;unique"`
}

// TableName 设置表名
func (Sub) TableName() string {
	return "sub"
}

// 初始化数据库
func InitDB() {
	// 自动创建不存在的表
	err := gorm.DB.AutoMigrate(&Sub{})
	if err != nil {
		fmt.Println("✅ [Sub] gorm 数据库表迁移失败:", err)
	} else {
		fmt.Println("✅ [Sub] gorm 数据库表迁移成功")
	}
}
