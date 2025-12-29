package config

type Seed struct {
	Enabled      bool     `env:"ENABLED,expand" envDefault:"true"`
	DefaultTasks []string `env:"DEFAULT_TASKS,expand" envDefault:"docker.io/bornholm/oplet-hello-world-task:latest"`
}
