package repositories

import (
	"demo/network/go_server/app/models"

	"gorm.io/gorm"
)

type FileHistoryRepository struct {
	db *gorm.DB
}

func NewFileHistoryRepository(db *gorm.DB) *FileHistoryRepository {
	return &FileHistoryRepository{db: db}
}

func (r *FileHistoryRepository) BulkCreate(histories []models.DeviceFileHistory) error {
	return r.db.Create(&histories).Error
}
