package system

import (
	"fmt"
	"runtime"
)

func OSConfiguratorFactory() (osConfigurator, error) {
	switch runtime.GOOS {
	case "linux":
		return NewLinuxConfigurator(), nil
	case "darwin":
		return NewDarwinConfigurator(), nil
	case "windows":
		return NewWindowsConfigurator(), nil
	}
	return nil, fmt.Errorf("unsupported OS %s, only linux, darwin and windows are supported", runtime.GOOS)
}
