package system

import "context"

type WindowsConfigurator struct {
}

func NewOSConfigurator() *WindowsConfigurator {
	return &WindowsConfigurator{}
}

func (c *WindowsConfigurator) GetConfig() (*Config, error) {
	return &Config{}, nil
}

func (c *WindowsConfigurator) ApplyConfig(cfg *Config) error {
	return nil
}

func (*WindowsConfigurator) GetFileDescriptors(ctx context.Context, pid int) ([]FD, error) {
	return nil, nil
}
