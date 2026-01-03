package repositories

import (
	"demo/network/go_server/app/models"

	"gorm.io/gorm"
)

type LogRepository struct {
	DB *gorm.DB
}

func NewLogRepository(db *gorm.DB) *LogRepository {
	return &LogRepository{DB: db}
}

func (r *LogRepository) Create(log *models.Log) error {
	return r.DB.Create(log).Error
}

func (r *LogRepository) GetRecent(deviceID string, limit int) ([]models.Log, error) {
	var logs []models.Log
	query := r.DB.Order("created_at desc").Limit(limit)
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	result := query.Find(&logs)
	return logs, result.Error
}
