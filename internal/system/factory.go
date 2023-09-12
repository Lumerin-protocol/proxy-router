package system

func OSConfiguratorFactory() (osConfigurator, error) {
	return NewOSConfigurator(), nil
}
