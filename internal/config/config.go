package config

type Config struct {
	Version string `yaml:"version" json:"version"`
	Service string `yaml:"service" json:"service"`
}

func NewDefaultConfig() *Config {
	return &Config{
		Version: "0.1.0",
		Service: "isame-lb",
	}
}

func (c *Config) Validate() error {
	if c.Service == "" {
		c.Service = "isame-lb"
	}
	if c.Version == "" {
		c.Version = "0.1.0"
	}
	return nil
}
