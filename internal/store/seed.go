package store

import (
	"time"
)

type Seed struct {
	ID         string `gorm:"primarykey"`
	ExecutedAt time.Time
}

func NewSeed(id string, executedAt time.Time) *Seed {
	return &Seed{
		ID:         id,
		ExecutedAt: executedAt,
	}
}
