package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port    int    `yaml:"port"`
		APIPort int    `yaml:"api_port"`
		DBDSN   string `yaml:"db_dsn"`
	} `yaml:"server"`
	Backup struct {
		StoragePath string `yaml:"storage_path"`
	} `yaml:"backup"`
}

var AppConfig Config

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &AppConfig)
}
