package models

import (
	"time"
)

type CommandStatus string

const (
	StatusPending   CommandStatus = "PENDING"
	StatusSent      CommandStatus = "SENT"
	StatusCompleted CommandStatus = "COMPLETED"
	StatusFailed    CommandStatus = "FAILED"
)

type Command struct {
	ID          uint          `gorm:"primaryKey"`
	DeviceID    string        `gorm:"index"`
	CommandType int           `gorm:"not null"` // e.g. 0xD3 (GET_LOGS)
	Payload     string        `gorm:"type:text"`
	Status      CommandStatus `gorm:"default:'PENDING'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
