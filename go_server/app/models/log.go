package models

import (
	"time"
)

type Log struct {
	ID        uint   `gorm:"primaryKey"`
	DeviceID  string `gorm:"index"`
	Content   string `gorm:"type:text"`
	CreatedAt time.Time
}
