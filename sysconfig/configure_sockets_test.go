package sysconfig

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfigureSocketsTestSuite struct {
	suite.Suite
	// Test fixtures
	originalConfig *MockSocketConfig
	currentConfig  *MockSocketConfig
	appConfig      *AppSocketConfiguration
}

func (s *ConfigureSocketsTestSuite) SetupTest() {
	// Create mocks
	s.originalConfig = &MockSocketConfig{SysctlInputs: make(map[string]string), SocketSystemConfig: &SocketSystemConfig{}}
	s.currentConfig = &MockSocketConfig{SysctlInputs: make(map[string]string), SocketSystemConfig: &SocketSystemConfig{}}

	// Create object under test
	s.appConfig = NewAppSocketConfiguration(
		s.originalConfig,
		s.currentConfig,
		"testLocalPortRange",
		"testTcpMaxSynBacklog",
		"testSomaxconn",
		"testNetdevMaxBacklog",
		1000,
		2000,
		"linux",
	)
}

func (s *ConfigureSocketsTestSuite) TestTryInitForLinux() {
	// Execute test
	appResult := s.appConfig.TryInitForLinux()

	// Assertions
	s.True(s.originalConfig.LoadSystemConfigCalled)

	// Assert result
	s.Equal(*s.appConfig.SystemRequirements.(*SocketSystemConfig),
		*appResult.(*MockSocketConfig).SocketSystemConfig)
}

func TestConfigureSocketsTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigureSocketsTestSuite))
}

// package sysconfig

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

type MockSocketConfig struct {
	*SocketSystemConfig
	// Fields to track calls
	LoadSystemConfigCalled bool
	SysctlInputs           map[string]string
}

func (m *MockSocketConfig) LoadSystemConfig() {
	// Update field to track call
	m.LoadSystemConfigCalled = true

	m.LocalPortRange = m.SysctlInputs["net.ipv4.ip_local_port_range"]
	m.TcpMaxSynBacklog = m.SysctlInputs["net.ipv4.tcp_max_syn_backlog"]
	m.Somaxconn = m.SysctlInputs["net.core.somaxconn"]
	m.NetdevMaxBacklog = m.SysctlInputs["net.core.netdev_max_backlog"]

	if m.SysctlInputs["rlimit_hard"] != "" {
		hardLimit, err := strconv.ParseUint(m.SysctlInputs["rlimit_hard"], 10, 64)

		if err != nil {
			panic(fmt.Errorf("Error parsing rlimit_hard %w", err))
		}
		m.RlimitHard = hardLimit
	}

	if m.SysctlInputs["rlimit_soft"] != "" {
		softLimit, err := strconv.ParseUint(m.SysctlInputs["rlimit_soft"], 10, 64)

		if err != nil {
			panic(fmt.Errorf("Error parsing rlimit_soft %w", err))
		}

		m.RlimitSoft = softLimit
	}
}

func (m *MockSocketConfig) SetSysctl(key string, value string) {
	m.SysctlInputs[key] = value
}

func (m *MockSocketConfig) GetSysctl(key string) string {
	return m.SysctlInputs[key]
}

func (m *MockSocketConfig) SetRlimit(rlimit uint64) {
	m.SysctlInputs["rlimit_hard"] = strconv.Itoa(int(rlimit))
	m.SysctlInputs["rlimit_soft"] = strconv.Itoa(int(rlimit))
}

// func TestTryInitForLinux(t *testing.T) {
// 	// Create mocks
// 	var originalConfig MockSocketConfig
// 	var currentConfig MockSocketConfig

// 	// Create object under test
// 	var appConfig = NewAppSocketConfiguration(
// 		&originalConfig,
// 		&currentConfig,
// 		"testLocalPortRange",
// 		"testTcpMaxSynBacklog",
// 		"testSomaxconn",
// 		"testNetdevMaxBacklog",
// 		1000,
// 		2000,
// 		"linux",
// 	)

// 	// Execute test
// 	appResult := appConfig.TryInitForLinux()

// 	// Assert mocks were called
// 	assert.True(t, originalConfig.LoadSystemConfigCalled)

// 	// Assert result
// 	assert.Equal(t, *appConfig.SystemRequirements.(*SocketSystemConfig), *appResult.(*SocketSystemConfig))
// }
