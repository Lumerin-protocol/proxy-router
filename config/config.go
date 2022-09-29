package config

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
		VettingPeriodSeconds int `env:"MINER_VETTING_PERIOD_SECONDS" validate:"required,gte=0"`
	}
	Log struct {
		Syslog bool `env:"LOG_SYSLOG" flag:"log-syslog"`
	}
	Proxy struct {
		Address    string `env:"PROXY_ADDRESS" flag:"proxy-address" validate:"required,hostname_port"`
		LogStratum bool   `env:"PROXY_LOG_STRATUM"`
	}
	Pool struct {
		Address string `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,uri"`
	}
	Web struct {
		Address string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port"`
	}
}
