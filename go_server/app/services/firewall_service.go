package services

import (
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"encoding/json"
	"fmt"
	"time"
)

type FirewallService struct {
	DeviceRepo   *repositories.DeviceRepository
	FirewallRepo *repositories.FirewallRepository
	CommandSvc   *CommandService
}

func NewFirewallService(dRepo *repositories.DeviceRepository, fRepo *repositories.FirewallRepository, cSvc *CommandService) *FirewallService {
	return &FirewallService{
		DeviceRepo:   dRepo,
		FirewallRepo: fRepo,
		CommandSvc:   cSvc,
	}
}

func (s *FirewallService) UpdateConfig(deviceID string, enable bool, categories []int) error {
	// Upsert Device Config
	config, err := s.DeviceRepo.GetByDeviceID(deviceID)
	if err != nil || config == nil {
		config = &models.DeviceConfig{DeviceID: deviceID}
	}

	config.FirewallEnabled = enable
	catsBytes, _ := json.Marshal(categories)
	config.SelectedCategories = string(catsBytes)
	config.UpdatedAt = time.Now()

	if err := s.DeviceRepo.Upsert(config); err != nil {
		return err
	}

	// Notify Client
	cmd, _ := s.CommandSvc.CreateCommand(deviceID, 0xE3, `{"command": "FIREWALL_UPDATE"}`)
	success := s.CommandSvc.TrySendImmediately(cmd)
	if success {
		fmt.Printf("[Service] Firewall Update Sent to %s\n", deviceID)
	} else {
		fmt.Printf("[Service] Firewall Update Queued for %s\n", deviceID)
	}

	return nil
}

type FirewallConfigResp struct {
	Enabled bool     `json:"enabled"`
	Domains []string `json:"domains"`
}

func (s *FirewallService) GetConfig(deviceID string) (*FirewallConfigResp, error) {
	config, err := s.DeviceRepo.GetByDeviceID(deviceID)
	if err != nil {
		return nil, err
	}

	resp := &FirewallConfigResp{
		Enabled: false,
		Domains: []string{},
	}

	resp.Enabled = config.FirewallEnabled
	if config.FirewallEnabled && config.SelectedCategories != "" {
		var catIDs []int
		json.Unmarshal([]byte(config.SelectedCategories), &catIDs)
		if len(catIDs) > 0 {
			domains, _ := s.FirewallRepo.GetDomainsByCategories(catIDs)
			resp.Domains = domains
		}
	}

	return resp, nil
}
