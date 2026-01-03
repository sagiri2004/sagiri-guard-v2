package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"demo/network/go_server/server"
	"fmt"
	"time"
)

type CommandService struct {
	Repo *repositories.CommandRepository
}

func NewCommandService(repo *repositories.CommandRepository) *CommandService {
	return &CommandService{Repo: repo}
}

func (s *CommandService) CreateCommand(deviceID string, cmdType int, payload string) (*models.Command, error) {
	cmd := &models.Command{
		DeviceID:    deviceID,
		CommandType: cmdType,
		Payload:     payload,
		Status:      models.StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := s.Repo.Create(cmd)
	return cmd, err
}

func (s *CommandService) ProcessPendingCommands(deviceID string) {
	fmt.Printf("[Service] Processing Queue for %s\n", deviceID)
	cmds, err := s.Repo.GetPendingByDevice(deviceID)
	if err != nil {
		fmt.Printf("[Service] Error fetching pending commands: %v\n", err)
		return
	}

	for _, cmd := range cmds {
		fmt.Printf("[Service] Sending Command %d (Type %d)\n", cmd.ID, cmd.CommandType)
		success := server.SendToDevice(deviceID, cmd.CommandType, cmd.Payload)
		if success {
			fmt.Printf("[Service] Command %d sent successfully.\n", cmd.ID)
			cmd.Status = models.StatusSent
			cmd.UpdatedAt = time.Now()
			s.Repo.Update(&cmd)
		} else {
			fmt.Printf("[Service] Failed to send Command %d.\n", cmd.ID)
		}
	}
}

// TrySendImmediately attempts to send and updates status if successful
func (s *CommandService) TrySendImmediately(cmd *models.Command) bool {
	success := server.SendToDevice(cmd.DeviceID, cmd.CommandType, cmd.Payload)
	if success {
		cmd.Status = models.StatusSent
		cmd.UpdatedAt = time.Now()
		s.Repo.Update(cmd)
		return true
	}
	return false
}
