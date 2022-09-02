package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/omeid/uconfig/flat"
)

const (
	TagEnv  = "env"
	TagFlag = "flag"
	TagDesc = "desc"
)

func LoadConfig(cfg interface{}, osArgs *[]string) error {
	godotenv.Load(".env")

	// recursively iterates over each field of the nested struct
	fields, err := flat.View(cfg)
	if err != nil {
		return err
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

	var args []string
	if osArgs != nil {
		args = *osArgs
	} else {
		args = os.Args
	}

	err = flagset.Parse(args[1:])
	if err != nil {
		return err
	}

	err = validator.New().Struct(cfg)
	if err != nil {
		return fmt.Errorf("config validation error: %w", err)
	}
	return nil
}
