package config

import "time"

// Validation tags described here: https://pkg.go.dev/github.com/go-playground/validator/v10
type Config struct {
	Contract struct {
		Address                string        `env:"CLONE_FACTORY_ADDRESS" flag:"contract-address" validate:"required_if=Disable false,omitempty,eth_addr"`
		IsBuyer                bool          `env:"IS_BUYER" flag:"is-buyer"`
		HashrateDiffThreshold  float64       `env:"HASHRATE_DIFF_THRESHOLD"`
		ValidationBufferPeriod time.Duration `env:"VALIDATION_BUFFER_PERIOD" validate:"duration"`
		Mnemonic               string        `env:"CONTRACT_MNEMONIC" validate:"required_without=WalletPrivateKey|required_if=Disable false"`
		AccountIndex           int           `env:"ACCOUNT_INDEX"`
		WalletPrivateKey       string        `env:"WALLET_PRIVATE_KEY" validate:"required_without=Mnemonic|required_if=Disable false"`
		WalletAddress          string        `env:"WALLET_ADDRESS" validate:"required_without=Mnemonic|required_if=Disable false"`
		ClaimFunds             bool
		LumerinTokenAddress    string
		ValidatorAddress       string
		ProxyAddress           string
		Disable                bool `env:"CONTRACT_DISABLE" flag:"contract-disable"`
	}
	Environment string `env:"ENVIRONMENT" flag:"environment"`
	EthNode     struct {
		Address string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address" validate:"required,url"`
	}
	Miner struct {
		VettingDuration time.Duration `env:"MINER_VETTING_DURATION" flag:"miner-vetting-duration" validate:"duration"`
	}
	Log struct {
		Syslog    bool   `env:"LOG_SYSLOG" flag:"log-syslog"`
		LogToFile bool   `env:"LOG_TO_FILE" flag:"log-to-file"`
		Color     bool   `env:"LOG_COLOR" flag:"log-color"`
		Level     string `env:"LOG_LEVEL" flag:"log-level" validate:"oneof=debug info warn error dpanic panic fatal"`
	}
	Proxy struct {
		Address              string `env:"PROXY_ADDRESS" flag:"proxy-address" validate:"required,hostname_port"`
		LogStratum           bool   `env:"PROXY_LOG_STRATUM"`
		ConnectionBufferSize int    `env:"STRATUM_SOCKET_BUFFER_SIZE" flag:"stratum-socket-buffer" validate:"required,numeric"`
	}
	Pool struct {
		Address     string        `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,uri"`
		MinDuration time.Duration `env:"POOL_MIN_DURATION" validate:"duration"`
		MaxDuration time.Duration `env:"POOL_MAX_DURATION" validate:"duration"`
		ConnTimeout time.Duration `env:"POOL_CONN_TIMEOUT" validate:"duration"`
	}
	Web struct {
		Address   string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port"`
		PublicUrl string `env:"WEB_PUBLIC_URL" flag:"web-public-url" desc:"public url of the proxyrouter, falls back to web-address if empty" validate:"omitempty,url"`
	}
}
