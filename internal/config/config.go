package config

import "time"

// Validation tags described here: https://pkg.go.dev/github.com/go-playground/validator/v10
type Config struct {
	Blockchain struct {
		EthNodeAddress string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address" validate:"required,url"`
		EthLegacyTx    bool   `env:"ETH_NODE_LEGACY_TX" flag:"eth-node-legacy-tx" desc:"use it to disable EIP-1559 transactions"`
	}
	Environment string `env:"ENVIRONMENT" flag:"environment"`
	Hashrate    struct {
		HashrateDiffThreshold  float64       `env:"HASHRATE_DIFF_THRESHOLD" flag:"hashrate-diff-threshold"`
		ValidationBufferPeriod time.Duration `env:"VALIDATION_BUFFER_PERIOD" flag:"validation-buffer-period" validate:"duration"`
	}
	Marketplace struct {
		CloneFactoryAddress string `env:"CLONE_FACTORY_ADDRESS" flag:"contract-address" validate:"required_if=Disable false,omitempty,eth_addr"`
		LumerinTokenAddress string `env:"LUMERIN_TOKEN_ADDRESS" flag:"lumerin-token-address" validate:"required_if=Disable false,omitempty,eth_addr"`
		Disable             bool   `env:"CONTRACT_DISABLE" flag:"contract-disable"`
		IsBuyer             bool   `env:"IS_BUYER" flag:"is-buyer"`
		Mnemonic            string `env:"CONTRACT_MNEMONIC" flag:"contract-mnemonic" validate:"required_without=WalletPrivateKey|required_if=Disable false"`
		WalletPrivateKey    string `env:"WALLET_PRIVATE_KEY" flag:"wallet-private-key" validate:"required_without=Mnemonic|required_if=Disable false"`
	}
	Miner struct {
		VettingDuration time.Duration `env:"MINER_VETTING_DURATION" flag:"miner-vetting-duration" validate:"duration"`
		SubmitErrLimit  int           `env:"MINER_SUBMIT_ERR_LIMIT" flag:"miner-submit-err-limit" desc:"amount of consecutive submit errors to consider miner faulty and exclude it from contracts, zero means disable faulty miners tracking"`
	}
	Log struct {
		LogToFile bool   `env:"LOG_TO_FILE" flag:"log-to-file"`
		Color     bool   `env:"LOG_COLOR" flag:"log-color"`
		Level     string `env:"LOG_LEVEL" flag:"log-level" validate:"oneof=debug info warn error dpanic panic fatal"`
	}
	Pool struct {
		Address     string        `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,uri"`
		ConnTimeout time.Duration `env:"POOL_CONN_TIMEOUT" flag:"pool-conn-timeout" validate:"duration"`
	}
	Proxy struct {
		Address     string `env:"PROXY_ADDRESS" flag:"proxy-address" validate:"required,hostname_port"`
		ProtocolLog bool   `env:"PROXY_PROTOCOL_LOG" flag:"proxy-protocol-log" desc:"verbose dump of protocol activities to separate files per connection"`
	}
	Web struct {
		Address   string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port"`
		PublicUrl string `env:"WEB_PUBLIC_URL" flag:"web-public-url" desc:"public url of the proxyrouter, falls back to web-address if empty" validate:"omitempty,url"`
	}
}
