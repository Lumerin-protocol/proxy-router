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

var recommendedValues = &SocketSystemConfig{
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

	c.CurrentConfig.SetSysctl("net.ipv4.ip_local_port_range", "1024 65535")
	c.CurrentConfig.SetSysctl("net.ipv4.tcp_max_syn_backlog", "100000")
	c.CurrentConfig.SetSysctl("net.core.somaxconn", "100000")
	c.CurrentConfig.SetSysctl("net.core.netdev_max_backlog", "100000")

	c.CurrentConfig.SetRlimit(524288)

	c.CurrentConfig.LoadSystemConfig()

	log.Printf("Original config: %+v", originalConfig)
	log.Printf("New config: %+v", c.CurrentConfig)

	return c.CurrentConfig
}

func (c *AppSocketConfiguration) InitSocketConfig() interfaces.ISocketConfig {

	systemConfig := &SocketSystemConfig{}

	linuxResult := c.TryInitForLinux()

	windowsResult := c.TryInitForWindows(systemConfig)

	if linuxResult != nil {
		return linuxResult
	} else {
		return windowsResult
	}
}

func (c *AppSocketConfiguration) CleanupSocketConfig() *SocketSystemConfig {

	systemConfig := &SocketSystemConfig{}

	linuxResult := c.TryCleanupForLinux(systemConfig)

	windowsResult := c.TryCleanupForWindows(systemConfig)

	if linuxResult != nil {
		return linuxResult
	} else {
		return windowsResult
	}
}

func (c *AppSocketConfiguration) TryCleanupForWindows(systemConfig *SocketSystemConfig) *SocketSystemConfig {
	panic("unimplemented")
}

func (c *AppSocketConfiguration) TryCleanupForLinux(systemConfig *SocketSystemConfig) *SocketSystemConfig {

	if c.GOOS != "linux" {
		log.Println("Not cleaning up on non-Linux OS")
		return systemConfig
	}

	config := originalConfig.(*SocketSystemConfig)

	systemConfig.LoadSystemConfig()

	systemConfig.SetSysctl("net.ipv4.ip_local_port_range", config.LocalPortRange)
	systemConfig.SetSysctl("net.ipv4.tcp_max_syn_backlog", config.TcpMaxSynBacklog)
	systemConfig.SetSysctl("net.core.somaxconn", config.Somaxconn)
	systemConfig.SetSysctl("net.core.netdev_max_backlog", config.NetdevMaxBacklog)

	systemConfig.SetRlimit(config.RlimitSoft)

	systemConfig.LoadSystemConfig()

	log.Printf("Original config: %+v", originalConfig)
	log.Printf("New config: %+v", systemConfig)

	return systemConfig
}

func (c *AppSocketConfiguration) TryInitForWindows(config *SocketSystemConfig) *SocketSystemConfig {
	if runtime.GOOS == "windows" {
		commands := []string{
			"netsh int ipv4 set dynamicport tcp start=1024 num=64512", // sudo sysctl -w net.ipv4.ip_local_port_range="1024 65535"
			"netsh int tcp set global synattackprotect=0",             // sudo sysctl -w net.ipv4.tcp_max_syn_backlog=100000
			"netsh int tcp set global maxsynbacklog=100000",           // sudo sysctl -w net.core.somaxconn=100000
			"netsh int tcp set global netdma=1",                       // sudo sysctl -w net.core.netdev_max_backlog=100000
			"setmaxstdio.exe 524288",                                  // sudo ulimit -n 524288
			"netsh int tcp set global synbacklog=511",                 // add command to set syn backlog
		}

		for _, command := range commands {
			exec.Command(command)
		}
	} else {
		log.Println("Not running reg commands as OS is not Windows")
	}

	return config
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

var Config *AppSocketConfiguration

func init() {
	Config = NewAppSocketConfiguration(
		&SocketSystemConfig{},
		&SocketSystemConfig{},
		recommendedValues.LocalPortRange,
		recommendedValues.TcpMaxSynBacklog,
		recommendedValues.Somaxconn,
		recommendedValues.NetdevMaxBacklog,
		recommendedValues.RlimitSoft,
		recommendedValues.RlimitHard,
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
