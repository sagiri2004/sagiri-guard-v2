package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string   `gorm:"size:255;uniqueIndex;not null"`
	Password string   `gorm:"not null"`
	Devices  []Device `gorm:"foreignKey:UserID"`
}
