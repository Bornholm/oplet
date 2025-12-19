package store

import "gorm.io/gorm"

type User struct {
	gorm.Model

	DisplayName string
	Subject     string   `gorm:"index"`
	Provider    string   `gorm:"index"`
	Roles       []string `gorm:"-"`

	TaskExecutions []*TaskExecution `gorm:"constraint:OnDelete:CASCADE;"`
}

func NewUser(provider, subject, displayName string, roles ...string) *User {
	return &User{
		DisplayName: displayName,
		Subject:     subject,
		Provider:    provider,
		Roles:       roles,
	}
}
