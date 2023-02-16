package config

import "time"

// Validation tags described here: https://pkg.go.dev/github.com/go-playground/validator/v10
type Config struct {
	Contract struct {
		Address                string        `env:"CLONE_FACTORY_ADDRESS" flag:"contract-address" validate:"required_if=Disable false,omitempty,eth_addr"`
		IsBuyer                bool          `env:"IS_BUYER" flag:"is-buyer"`
		HashrateDiffThreshold  float64       `env:"HASHRATE_DIFF_THRESHOLD" flag:"hashrate-diff-threshold"`
		ValidationBufferPeriod time.Duration `env:"VALIDATION_BUFFER_PERIOD" flag:"validation-buffer-period" validate:"duration"`
		Mnemonic               string        `env:"CONTRACT_MNEMONIC" flag:"contract-mnemonic" validate:"required_without=WalletPrivateKey|required_if=Disable false"`
		AccountIndex           int           `env:"ACCOUNT_INDEX" flag:"account-index"`
		WalletPrivateKey       string        `env:"WALLET_PRIVATE_KEY" flag:"wallet-private-key" validate:"required_without=Mnemonic|required_if=Disable false"`
		WalletAddress          string        `env:"WALLET_ADDRESS" flag:"wallet-address" validate:"required_without=Mnemonic|required_if=Disable false"`
		ClaimFunds             bool
		LumerinTokenAddress    string
		ValidatorAddress       string
		ProxyAddress           string
		Disable                bool          `env:"CONTRACT_DISABLE" flag:"contract-disable"`
		CycleDuration          time.Duration `env:"CONTRACT_CYCLE_DURATION" flag:"contract-cycle-duration"`
	}
	Environment string `env:"ENVIRONMENT" flag:"environment"`
	EthNode     struct {
		Address  string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address" validate:"required,url"`
		LegacyTx bool   `env:"ETH_NODE_LEGACY_TX" flag:"eth-node-legacy-tx" desc:"use it to disable EIP-1559 transactions"`
	}
	Miner struct {
		VettingDuration time.Duration `env:"MINER_VETTING_DURATION" flag:"miner-vetting-duration" validate:"duration"`
		SubmitErrLimit  int           `env:"MINER_SUBMIT_ERR_LIMIT" flag:"miner-submit-err-limit" desc:"amount of consecutive submit errors to consider miner faulty and exclude it from contracts, zero means disable faulty miners tracking"`
	}
	Log struct {
		Syslog    bool   `env:"LOG_SYSLOG" flag:"log-syslog"`
		LogToFile bool   `env:"LOG_TO_FILE" flag:"log-to-file"`
		Color     bool   `env:"LOG_COLOR" flag:"log-color"`
		Level     string `env:"LOG_LEVEL" flag:"log-level" validate:"oneof=debug info warn error dpanic panic fatal"`
	}
	Proxy struct {
		Address              string `env:"PROXY_ADDRESS" flag:"proxy-address" validate:"required,hostname_port"`
		LogStratum           bool   `env:"PROXY_LOG_STRATUM" flag:"proxy-log-stratum"`
		ConnectionBufferSize int    `env:"STRATUM_SOCKET_BUFFER_SIZE" flag:"stratum-socket-buffer" validate:"required,numeric"`
	}
	Pool struct {
		Address     string        `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,uri"`
		MinDuration time.Duration `env:"POOL_MIN_DURATION" flag:"pool-min-duration" validate:"duration"`
		MaxDuration time.Duration `env:"POOL_MAX_DURATION" flag:"pool-max-duration" validate:"duration"`
		ConnTimeout time.Duration `env:"POOL_CONN_TIMEOUT" flag:"pool-conn-timeout" validate:"duration"`
	}
	Web struct {
		Address   string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port"`
		PublicUrl string `env:"WEB_PUBLIC_URL" flag:"web-public-url" desc:"public url of the proxyrouter, falls back to web-address if empty" validate:"omitempty,url"`
	}
}
