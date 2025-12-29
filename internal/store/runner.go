package store

import (
	"time"

	"gorm.io/gorm"
)

type Runner struct {
	gorm.Model

	Name  string `gorm:"unique"`
	Token string `gorm:"unique"`

	ContactedAt *time.Time
}
