package configurator

type MetricsConfig struct {
	Prefix  string `mapstructure:"prefix"`
	Enabled bool   `mapstructure:"enabled"`
}
