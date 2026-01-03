package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"fmt"
)

type AdminService struct {
	CmdRepo    *repositories.CommandRepository
	CommandSvc *CommandService
}

func NewAdminService(cmdRepo *repositories.CommandRepository, cmdSvc *CommandService) *AdminService {
	return &AdminService{
		CmdRepo:    cmdRepo,
		CommandSvc: cmdSvc,
	}
}

func (s *AdminService) QueueGetLogs(deviceID string, lineCount int) (*models.Command, error) {
	payload := fmt.Sprintf(`{"command":"GET_LOGS", "line_count": %d}`, lineCount)
	cmd, err := s.CommandSvc.CreateCommand(deviceID, 0xD3, payload)
	if err != nil {
		return nil, err
	}

	// Async attempt
	go s.CommandSvc.TrySendImmediately(cmd)

	return cmd, nil
}

func (s *AdminService) GetCommandHistory(deviceID string, page, size int) ([]models.Command, error) {
	offset := (page - 1) * size
	return s.CmdRepo.GetHistory(deviceID, size, offset)
}
