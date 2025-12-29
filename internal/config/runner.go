package config

type Runner struct {
	Enabled bool `env:"ENABLED,expand" envDefault:"true"`
}
