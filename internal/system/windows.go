package system

type WindowsConfigurator struct {
}

func NewWindowsConfigurator() *WindowsConfigurator {
	return &WindowsConfigurator{}
}

func (c *WindowsConfigurator) GetConfig() (*Config, error) {
	return &Config{}, nil
}

func (c *WindowsConfigurator) ApplyConfig(cfg *Config) error {
	return nil
}