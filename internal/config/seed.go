package config

type Seed struct {
	Enabled      bool     `env:"ENABLED" envDefault:"true"`
	DefaultTasks []string `env:"DEFAULT_TASKS" envDefault:"docker.io/bornholm/oplet-hello-world-task:latest"`
}
