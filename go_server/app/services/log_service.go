package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
)

type LogService struct {
	Repo *repositories.LogRepository
}

func NewLogService(repo *repositories.LogRepository) *LogService {
	return &LogService{Repo: repo}
}

func (s *LogService) StoreLog(deviceID, content string) error {
	log := &models.Log{
		DeviceID: deviceID,
		Content:  content,
	}
	return s.Repo.Create(log)
}

func (s *LogService) GetRecentLogs(deviceID string, limit int) ([]models.Log, error) {
	return s.Repo.GetRecent(deviceID, limit)
}
