package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	configFileName = ".radconfig.json"
)

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	var cfg Config
	path, err := configFilePath()
	if err != nil {
		return cfg, fmt.Errorf("error calling Config method configFilePath: %w", err)
	}

	configData, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("error reading file: %w", err)
	}

	err = json.Unmarshal(configData, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("error unmarshalling data: %w", err)
	}

	return cfg, nil
}

func (c *Config) SetUser(name string) error {
	c.CurrentUserName = name
	if err := write(*c); err != nil {
		return fmt.Errorf("error calling Config method write: %w", err)
	}
	return nil
}

func configFilePath() (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path + "/" + configFileName, nil
}

func write(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("unable to marshal config with err: %w", err)
	}
	path, err := configFilePath()
	if err != nil {
		return fmt.Errorf("error calling Config method configFilePath: %w", err)
	}
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return err
	}

	return nil
}
