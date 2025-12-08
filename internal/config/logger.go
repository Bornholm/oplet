package config

import "log/slog"

type Logger struct {
	Level slog.Level `env:"LEVEL,expand" envDefault:"debug"`
}
