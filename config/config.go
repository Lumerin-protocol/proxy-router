package config

import "time"

// Validation tags described here: https://github.com/go-playground/validator
type Config struct {
	Contract struct {
		Address             string `env:"CONTRACT_ADDRESS" flag:"contract-address" validate:"required,eth_addr"`
		IsBuyer             bool   `env:"IS_BUYER" flag:"is-buyer"`
		Mnemonic            string `env:"CONTRACT_MNEMONIC"`
		AccountIndex        int    `env:"ACCOUNT_INDEX"`
		EthNodeAddr         string
		ClaimFunds          bool
		TimeThreshold       int
		LumerinTokenAddress string
		ValidatorAddress    string
		ProxyAddress        string
		WalletAddress       string `env:"SELLER_ADDRESS"`
	}
	Environment string `env:"ENVIRONMENT" flag:"environment"`
	EthNode     struct {
		Address string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address" validate:"required,url"`
	}
	Miner struct {
		VettingDuration time.Duration `env:"MINER_VETTING_DURATION" validate:"duration"`
	}
	Log struct {
		Syslog bool `env:"LOG_SYSLOG" flag:"log-syslog"`
	}
	Proxy struct {
		Address    string `env:"PROXY_ADDRESS" flag:"proxy-address" validate:"required,hostname_port"`
		LogStratum bool   `env:"PROXY_LOG_STRATUM"`
	}
	Pool struct {
		Address     string        `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,uri"`
		MinDuration time.Duration `env:"POOL_MIN_DURATION" validate:"duration"`
		MaxDuration time.Duration `env:"POOL_MAX_DURATION" validate:"duration"`
	}
	Web struct {
		Address string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port"`
	}
}
