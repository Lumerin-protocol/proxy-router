package config

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

type TestConfig struct {
	StringVal    string `env:"STRING_VAL" flag:"string-val"`
	BoolVal      bool   `env:"BOOL_VAL" flag:"bool-val"`
	NestedStruct struct {
		NestedVal string `env:"NESTED_VAL" flag:"nested-val"`
	}
	DurationVal time.Duration `env:"DURATION_VAL"`
}

type TestConfigRequired struct {
	RequiredVal string `env:"REQUIRED_VAL" flag:"required-val" validate:"required"`
}

func TestEnvString(t *testing.T) {
	val := "string-value"
	t.Setenv("STRING_VAL", val)
	var config TestConfig

	err := LoadConfig(&config, &[]string{"cmd"})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.StringVal != val {
		t.Fatal()
	}
}

func TestEnvBoolTrue(t *testing.T) {
	t.Setenv("BOOL_VAL", "true")
	var config TestConfig

	err := LoadConfig(&config, &[]string{"cmd"})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.BoolVal != true {
		t.Fatal()
	}
}

func TestEnvBoolFalse(t *testing.T) {
	t.Setenv("BOOL_VAL", "false")
	var config TestConfig

	err := LoadConfig(&config, &[]string{"cmd"})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.BoolVal != false {
		t.Fatal()
	}
}

func TestEnvNestedVal(t *testing.T) {
	var config TestConfig
	val := "test-value"
	t.Setenv("NESTED_VAL", val)
	err := LoadConfig(&config, &[]string{"cmd"})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.NestedStruct.NestedVal != val {
		t.Fatal()
	}
}

func TestEnvDuration(t *testing.T) {
	expectedSeconds := 123
	t.Setenv("DURATION_VAL", fmt.Sprintf("%ds", expectedSeconds))
	var config TestConfig

	err := LoadConfig(&config, &[]string{"cmd"})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	actualSeconds := int(config.DurationVal.Seconds())
	if actualSeconds != expectedSeconds {
		t.Fatalf("expected (%d) actual (%d)", expectedSeconds, actualSeconds)
	}
}

func TestFlagString(t *testing.T) {
	var config TestConfig
	val := "string-value"

	err := LoadConfig(&config, &[]string{"cmd", fmt.Sprintf("-string-val=%s", val)})

	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.StringVal != val {
		t.Fatal()
	}
}

func TestFlagBoolTrue(t *testing.T) {
	var config TestConfig

	err := LoadConfig(&config, &[]string{"cmd", "-bool-val=true"})

	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.BoolVal != true {
		t.Fatal()
	}
}

func TestFlagBoolFalse(t *testing.T) {
	var config TestConfig

	err := LoadConfig(&config, &[]string{"cmd", "-bool-val=false"})

	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.BoolVal != false {
		t.Fatal()
	}
}

func TestFlagNestedVal(t *testing.T) {
	var config TestConfig
	val := "test-value"
	err := LoadConfig(&config, &[]string{"cmd", fmt.Sprintf("-nested-val=%s", val)})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if config.NestedStruct.NestedVal != val {
		t.Fatal()
	}
}

func TestEnvRequired(t *testing.T) {
	var config TestConfigRequired
	err := LoadConfig(&config, &[]string{"cmd"})
	if err == nil {
		t.Fatal("should error if required field missing")
	}
	var vErrs validator.ValidationErrors

	if !errors.As(err, &vErrs) {
		fmt.Printf("%v\n\n\n", err)
		t.Fatal("invalid error type")
	}
	vErr := vErrs[0]

	if vErr.Field() != "RequiredVal" || vErr.ActualTag() != "required" {
		t.Fatal("unexpected field/tag")
	}

}

func TestFlagRequired(t *testing.T) {
	var config TestConfigRequired
	err := LoadConfig(&config, &[]string{"cmd"})

	if err == nil {
		t.Fatal("should error if required field missing")
	}
	var vErrs validator.ValidationErrors
	if !errors.As(err, &vErrs) {
		t.Fatalf("invalid error type %v", err)
	}
	vErr := vErrs[0]

	if vErr.Field() != "RequiredVal" || vErr.ActualTag() != "required" {
		t.Fatal("unexpected field/tag")
	}
}
