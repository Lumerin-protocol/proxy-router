package config

import (
	"flag"
	"os"

	"github.com/joho/godotenv"
	"github.com/omeid/uconfig/flat"
)

type Config struct {
	Web struct {
		Address string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port"`
	}
	Pool struct {
		Address  string `env:"POOL_ADDRESS" flag:"pool-address"`
		User     string `env:"POOL_USER" flag:"pool-user"`
		Password string `env:"POOL_PASSWORD" flag:"pool-password"`
	}
	Contract struct {
		Address string `env:"CONTRACT_ADDRESS" flag:"contract-address"`
	}
	EthNode struct {
		Address string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address"`
	}
	Log struct {
		Syslog   bool   `env:"LOG_SYSLOG" flag:"log-syslog"`
		FilePath string `env:"LOG_FILE_PATH" flag:"log-file-path"`
	}
}

const (
	TagEnv  = "env"
	TagFlag = "flag"
	TagDesc = "desc"
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	godotenv.Load(".env")

	// iterates over nested struct
	fields, err := flat.View(cfg)
	if err != nil {
		return nil, err
	}

	flagset := flag.NewFlagSet("", flag.ContinueOnError)

	for _, field := range fields {

		envName, ok := field.Tag(TagEnv)
		if !ok {
			continue
		}
		value := os.Getenv(envName)
		field.Set(value)

		flagName, ok := field.Tag(TagFlag)
		if !ok {
			continue
		}

		flagDesc, _ := field.Tag(TagDesc)

		flagset.Var(field, flagName, flagDesc)
	}

	err = flagset.Parse(os.Args[1:])

	return cfg, err
}
