package config

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig structure for config.yml
type AppConfig struct {
	Client struct {
		ServerHost string `yaml:"server_host"`
		ServerPort int    `yaml:"server_port"`
		APIPort    int    `yaml:"api_port"`
	} `yaml:"client"`
	Admin struct {
		ServerHost string `yaml:"server_host"`
		ServerPort int    `yaml:"server_port"`
		APIPort    int    `yaml:"api_port"`
	} `yaml:"admin"`
}

// DeviceConfig structure for device.json (persisted device info)
type DeviceConfig struct {
	DeviceID string `json:"device_id"`
}

var GlobalAppConfig AppConfig

func LoadAppConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &GlobalAppConfig)
}

func LoadDeviceConfig() (*DeviceConfig, error) {
	file, err := os.Open("device.json")
	if err != nil {
		return &DeviceConfig{}, err
	}
	defer file.Close()

	var config DeviceConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return &config, err
}

func SaveDeviceConfig(config *DeviceConfig) error {
	file, err := os.Create("device.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}
