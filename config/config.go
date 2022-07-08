package config

import (
	"github.com/jinzhu/configor"
	"github.com/joho/godotenv"
)

type Config struct {
	WebAddress      string `env:"WEB_ADDRESS"`
	PoolAddress     string `env:"POOL_ADDRESS"`
	PoolUser        string `env:"POOL_USER"`
	PoolPassword    string `env:"POOL_PASSWORD"`
	ContractAddress string `env:"CONTRACT_ADDRESS"`
	EthNodeAddress  string `env:"ETH_NODE_ADDRESS"`
	LogSyslog       string `env:"LOG_SYSLOG"`
	LogFilePath     string `env:"LOG_FILE_PATH"`
}

func NewConfig() (*Config, error) {
	var config Config
	if err := godotenv.Load(".env"); err != nil {
		return nil, err
	}

	if err := configor.Load(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
