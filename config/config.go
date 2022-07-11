package config

import (
	"flag"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/omeid/uconfig/flat"
)

type Config struct {
	Web struct {
		Address string `env:"WEB_ADDRESS" flag:"web-address" desc:"http server address host:port" validate:"required,hostname_port"`
	}
	Pool struct {
		Address  string `env:"POOL_ADDRESS" flag:"pool-address" validate:"required,hostname_port"`
		User     string `env:"POOL_USER" flag:"pool-user" validate:"required"`
		Password string `env:"POOL_PASSWORD" flag:"pool-password"`
	}
	Contract struct {
		Address string `env:"CONTRACT_ADDRESS" flag:"contract-address" validate:"required,eth_addr"`
	}
	EthNode struct {
		Address string `env:"ETH_NODE_ADDRESS" flag:"eth-node-address" validate:"required,url"`
	}
	Log struct {
		Syslog bool `env:"LOG_SYSLOG" flag:"log-syslog"`
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

	// iterates over each field of the nested struct
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
		envValue := os.Getenv(envName)
		field.Set(envValue)

		flagName, ok := field.Tag(TagFlag)
		if !ok {
			continue
		}

		flagDesc, _ := field.Tag(TagDesc)

		// writes flag value to variable
		flagset.Var(field, flagName, flagDesc)
	}

	err = flagset.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	return cfg, validator.New().Struct(cfg)
}
