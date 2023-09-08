package config

import (
	"runtime"
	"strings"
	"time"
)

// Validation tags described here: https://pkg.go.dev/github.com/go-playground/validator/v10
type Config struct {
	Blockchain struct {
		EthNodeAddress string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address" validate:"required,url"`
		EthLegacyTx    bool   `env:"ETH_NODE_LEGACY_TX" flag:"eth-node-legacy-tx" desc:"use it to disable EIP-1559 transactions"`
	}
	Environment string `env:"ENVIRONMENT" flag:"environment"`
	Hashrate    struct {
		CycleDuration time.Duration `env:"HASHRATE_CYCLE_DURATION" flag:"hashrate-cycle-duration" validate:"duration"`
		// DiffThreshold          float64       `env:"HASHRATE_DIFF_THRESHOLD" flag:"hashrate-diff-threshold"`
		ValidationBufferPeriod time.Duration `env:"VALIDATION_BUFFER_PERIOD" flag:"validation-buffer-period" validate:"duration"`
	}
	Marketplace struct {
		CloneFactoryAddress string `env:"CLONE_FACTORY_ADDRESS" flag:"contract-address" validate:"required_if=Disable false,omitempty,eth_addr"`
		// LumerinTokenAddress string `env:"LUMERIN_TOKEN_ADDRESS" flag:"lumerin-token-address" validate:"required_if=Disable false,omitempty,eth_addr"`
		Disable          bool   `env:"CONTRACT_DISABLE" flag:"contract-disable"`
		IsBuyer          bool   `env:"IS_BUYER" flag:"is-buyer"`
		Mnemonic         string `env:"CONTRACT_MNEMONIC" flag:"contract-mnemonic" validate:"required_without=WalletPrivateKey|required_if=Disable false"`
		WalletPrivateKey string `env:"WALLET_PRIVATE_KEY" flag:"wallet-private-key" validate:"required_without=Mnemonic|required_if=Disable false"`
	}
	Miner struct {
		VettingDuration time.Duration `env:"MINER_VETTING_DURATION" flag:"miner-vetting-duration" validate:"duration"`
		ShareTimeout    time.Duration `env:"MINER_SHARE_TIMEOUT" flag:"miner-share-timeout" validate:"duration"`
		// SubmitErrLimit  int           `env:"MINER_SUBMIT_ERR_LIMIT" flag:"miner-submit-err-limit" desc:"amount of consecutive submit errors to consider miner faulty and exclude it from contracts, zero means disable faulty miners tracking"`
	}
	Log struct {
		LogToFile       bool   `env:"LOG_TO_FILE" flag:"log-to-file"`
		Color           bool   `env:"LOG_COLOR" flag:"log-color"`
		LevelConnection string `env:"LOG_LEVEL_CONNECTION" flag:"log-level-connection" validate:"oneof=debug info warn error dpanic panic fatal"`
		LevelProxy      string `env:"LOG_LEVEL_PROXY" flag:"log-level-proxy" validate:"oneof=debug info warn error dpanic panic fatal"`
		LevelScheduler  string `env:"LOG_LEVEL_SCHEDULER" flag:"log-level-scheduler" validate:"oneof=debug info warn error dpanic panic fatal"`
		LevelApp        string `env:"LOG_LEVEL_APP" flag:"log-level-app" validate:"oneof=debug info warn error dpanic panic fatal"`
	}
	Pool struct {
		Address string `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,uri"`
		// ConnTimeout time.Duration `env:"POOL_CONN_TIMEOUT" flag:"pool-conn-timeout" validate:"duration"`
	}
	Proxy struct {
		Address string `env:"PROXY_ADDRESS" flag:"proxy-address" validate:"required,hostname_port"`
	}
	System struct {
		Enable           bool   `env:"SYS_ENABLE" flag:"sys-enable" desc:"enable system level configuration adjustments"`
		LocalPortRange   string `env:"SYS_LOCAL_PORT_RANGE" flag:"sys-local-port-range" desc:"" validate:""`
		NetdevMaxBacklog string `env:"SYS_NET_DEV_MAX_BACKLOG" flag:"sys-netdev-max-backlog" desc:"" validate:""`
		RlimitHard       uint64 `env:"SYS_RLIMIT_HARD" flag:"sys-rlimit-hard" desc:"" validate:""`
		RlimitSoft       uint64 `env:"SYS_RLIMIT_SOFT" flag:"sys-rlimit-soft" desc:"" validate:""`
		Somaxconn        string `env:"SYS_SOMAXCONN" flag:"sys-somaxconn" desc:"" validate:""`
		TcpMaxSynBacklog string `env:"SYS_TCP_MAX_SYN_BACKLOG" flag:"sys-tcp-max-syn-backlog" desc:"" validate:""`
	}
	Web struct {
		Address   string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port" default:"0.0.0.0:3333"`
		PublicUrl string `env:"WEB_PUBLIC_URL" flag:"web-public-url" desc:"public url of the proxyrouter, falls back to web-address if empty" validate:"omitempty,url"`
	}
}

func (cfg *Config) SetDefaults() {
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// Hashrate

	if cfg.Hashrate.CycleDuration == 0 {
		cfg.Hashrate.CycleDuration = time.Duration(5 * time.Minute)
	}
	if cfg.Hashrate.ValidationBufferPeriod == 0 {
		cfg.Hashrate.ValidationBufferPeriod = time.Duration(10 * time.Minute)
	}

	// Marketplace

	// normalizes private key
	cfg.Marketplace.WalletPrivateKey = strings.TrimPrefix(cfg.Marketplace.WalletPrivateKey, "0x")

	// Miner

	if cfg.Miner.VettingDuration == 0 {
		cfg.Miner.VettingDuration = time.Duration(5 * time.Minute)
	}

	if cfg.Miner.ShareTimeout == 0 {
		cfg.Miner.ShareTimeout = time.Duration(2 * time.Minute)
	}

	// Log

	if cfg.Log.LevelConnection == "" {
		cfg.Log.LevelConnection = "info"
	}
	if cfg.Log.LevelProxy == "" {
		cfg.Log.LevelProxy = "info"
	}
	if cfg.Log.LevelScheduler == "" {
		cfg.Log.LevelScheduler = "info"
	}
	if cfg.Log.LevelApp == "" {
		cfg.Log.LevelApp = "debug"
	}

	// System

	// cfg.System.Enable = true // TODO: Temporary override, remove this line

	if cfg.System.LocalPortRange == "" {
		cfg.System.LocalPortRange = "1024 65535"
	}
	if cfg.System.TcpMaxSynBacklog == "" {
		cfg.System.TcpMaxSynBacklog = "100000"
	}
	if cfg.System.Somaxconn == "" && runtime.GOOS == "linux" {
		cfg.System.Somaxconn = "100000"
	}
	if cfg.System.Somaxconn == "" && runtime.GOOS == "darwin" {
		// setting high value like 1000000 on darwin
		// for some reason blocks incoming connections
		// TODO: investigate best value for this
		cfg.System.Somaxconn = "2048"
	}
	if cfg.System.NetdevMaxBacklog == "" {
		cfg.System.NetdevMaxBacklog = "100000"
	}
	if cfg.System.RlimitSoft == 0 {
		cfg.System.RlimitSoft = 524288
	}
	if cfg.System.RlimitHard == 0 {
		cfg.System.RlimitHard = 524288
	}

	// Proxy

	if cfg.Proxy.Address == "" {
		cfg.Proxy.Address = "0.0.0.0:3333"
	}
	if cfg.Web.Address == "" {
		cfg.Web.Address = "0.0.0.0:3001"
	}
	if cfg.Web.PublicUrl == "" {
		cfg.Web.PublicUrl = "http://localhost:3001"
	}
}
