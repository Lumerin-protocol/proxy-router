package sysconfig

import (
	"log"
	"os/exec"
	"runtime"
	"syscall"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

var originalConfig interfaces.ISocketConfig

type SocketSystemConfig struct {
	LocalPortRange   string
	TcpMaxSynBacklog string
	Somaxconn        string
	NetdevMaxBacklog string
	RlimitSoft       uint64
	RlimitHard       uint64
}

type AppSocketConfiguration struct {
	SystemRequirements interfaces.ISocketConfig
	OriginalConfig     interfaces.ISocketConfig
	CurrentConfig      interfaces.ISocketConfig
	GOOS               string
}

var ConfiguredValues = &SocketSystemConfig{
	LocalPortRange:   "1024 65535",
	TcpMaxSynBacklog: "100000",
	Somaxconn:        "100000",
	NetdevMaxBacklog: "100000",
	RlimitSoft:       524288,
	RlimitHard:       524288,
}

func (c *AppSocketConfiguration) TryInitForLinux() interfaces.ISocketConfig {

	if c.GOOS != "linux" {
		log.Println("Not initializing on non-Linux OS")
		return nil
	}

	c.OriginalConfig.LoadSystemConfig()

	c.CurrentConfig.SetSysctl("net.ipv4.ip_local_port_range", ConfiguredValues.LocalPortRange)
	c.CurrentConfig.SetSysctl("net.ipv4.tcp_max_syn_backlog", ConfiguredValues.TcpMaxSynBacklog)
	c.CurrentConfig.SetSysctl("net.core.somaxconn", ConfiguredValues.Somaxconn)
	c.CurrentConfig.SetSysctl("net.core.netdev_max_backlog", ConfiguredValues.NetdevMaxBacklog)

	c.CurrentConfig.SetRlimit(ConfiguredValues.RlimitHard)

	c.CurrentConfig.LoadSystemConfig()

	log.Printf("Original config: %+v", originalConfig)
	log.Printf("New config: %+v", c.CurrentConfig)

	return c.CurrentConfig
}

func (c *AppSocketConfiguration) InitSocketConfig() interfaces.ISocketConfig {

	return c.TryInitForLinux()
}

func (c *AppSocketConfiguration) TryCleanupForLinux() interfaces.ISocketConfig {

	if c.GOOS != "linux" {
		log.Println("Not cleaning up on non-Linux OS")
		return nil
	}

	config := c.OriginalConfig

	config.SetSysctl("net.ipv4.ip_local_port_range", config.GetLocalPortRange())
	config.SetSysctl("net.ipv4.tcp_max_syn_backlog", config.GetTcpMaxSynBacklog())
	config.SetSysctl("net.core.somaxconn", config.GetSomaxconn())
	config.SetSysctl("net.core.netdev_max_backlog", config.GetNetdevMaxBacklog())

	config.SetRlimit(config.GetRlimitHard())

	c.CurrentConfig.LoadSystemConfig()

	log.Printf("Original config: %+v", originalConfig)
	log.Printf("Current config: %+v", c.CurrentConfig)

	return c.CurrentConfig
}

func (c *AppSocketConfiguration) CleanupSocketConfig() interfaces.ISocketConfig {

	return  c.TryCleanupForLinux()
}

// Move functions into Config struct
func (c *SocketSystemConfig) LoadSystemConfig() {

	c.LocalPortRange = c.GetSysctl("net.ipv4.ip_local_port_range")
	c.TcpMaxSynBacklog = c.GetSysctl("net.ipv4.tcp_max_syn_backlog")
	c.Somaxconn = c.GetSysctl("net.core.somaxconn")
	c.NetdevMaxBacklog = c.GetSysctl("net.core.netdev_max_backlog")

	var rlimit syscall.Rlimit
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	c.RlimitSoft = rlimit.Cur
	c.RlimitHard = rlimit.Max
}

func (c *SocketSystemConfig) GetSysctl(name string) string {
	out, _ := exec.Command("sysctl", "-n", name).Output()
	return string(out)
}

func (c *SocketSystemConfig) SetSysctl(name, value string) {
	exec.Command("sysctl", "-w", name+"="+value).Run()
}

func (c *SocketSystemConfig) SetRlimit(limit uint64) {
	var rlim syscall.Rlimit
	rlim.Cur = limit
	rlim.Max = limit
	syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
}

func (c *SocketSystemConfig) GetLocalPortRange() string {
	return c.LocalPortRange
}

func (c *SocketSystemConfig) SetLocalPortRange(value string) {
	c.LocalPortRange = value
}

func (c *SocketSystemConfig) GetTcpMaxSynBacklog() string {
	return c.TcpMaxSynBacklog
}

func (c *SocketSystemConfig) SetTcpMaxSynBacklog(value string) {
	c.TcpMaxSynBacklog = value
}

func (c *SocketSystemConfig) GetSomaxconn() string {
	return c.Somaxconn
}

func (c *SocketSystemConfig) SetSomaxconn(value string) {
	c.Somaxconn = value
}

func (c *SocketSystemConfig) GetNetdevMaxBacklog() string {
	return c.NetdevMaxBacklog
}

func (c *SocketSystemConfig) SetNetdevMaxBacklog(value string) {
	c.NetdevMaxBacklog = value
}

func (c *SocketSystemConfig) GetRlimitSoft() uint64 {
	return c.RlimitSoft
}

func (c *SocketSystemConfig) SetRlimitSoft(value uint64) {
	c.RlimitSoft = value
}

func (c *SocketSystemConfig) GetRlimitHard() uint64 {
	return c.RlimitHard
}

func (c *SocketSystemConfig) SetRlimitHard(value uint64) {
	c.RlimitHard = value
}

var Config *AppSocketConfiguration

func init() {
	Config = NewAppSocketConfiguration(
		&SocketSystemConfig{},
		&SocketSystemConfig{},
		ConfiguredValues.LocalPortRange,
		ConfiguredValues.TcpMaxSynBacklog,
		ConfiguredValues.Somaxconn,
		ConfiguredValues.NetdevMaxBacklog,
		ConfiguredValues.RlimitSoft,
		ConfiguredValues.RlimitHard,
		runtime.GOOS)
}

func NewAppSocketConfiguration(
	originalConfig interfaces.ISocketConfig,
	currentConfig interfaces.ISocketConfig,
	LocalPortRange string,
	TcpMaxSynBacklog string,
	Somaxconn string,
	NetdevMaxBacklog string,
	RlimitSoft uint64,
	RlimitHard uint64, GOOS string) *AppSocketConfiguration {
	return &AppSocketConfiguration{
		SystemRequirements: &SocketSystemConfig{
			LocalPortRange:   LocalPortRange,
			TcpMaxSynBacklog: TcpMaxSynBacklog,
			Somaxconn:        Somaxconn,
			NetdevMaxBacklog: NetdevMaxBacklog,

			RlimitSoft: RlimitSoft,
			RlimitHard: RlimitHard,
		},
		OriginalConfig: originalConfig,
		CurrentConfig:  currentConfig,
		GOOS:           GOOS,
	}
}

func NewSocketSystemConfiguration() *SocketSystemConfig {
	return &SocketSystemConfig{}
}
