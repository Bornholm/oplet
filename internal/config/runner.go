package config

type Runner struct {
	Enabled   bool   `env:"ENABLED,expand" envDefault:"true"`
	ServerURL string `env:"SERVER,expand" envDefault:"http://127.0.0.1:3002"`
}
