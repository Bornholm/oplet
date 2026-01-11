package store

import "gorm.io/gorm"

type User struct {
	gorm.Model

	DisplayName string
	Subject     string `gorm:"index"`
	Provider    string `gorm:"index"`
	Email       string `gorm:"index"`
	Role        string `gorm:"default:'user'"`
	IsActive    bool   `gorm:"index"`

	TaskExecutions []*TaskExecution `gorm:"constraint:OnDelete:CASCADE;"`

	PreferredLanguage string
}

func NewUser(provider, subject, displayName, email, role string) *User {
	user := &User{
		DisplayName: displayName,
		Subject:     subject,
		Provider:    provider,
		IsActive:    true,
	}

	user.Role = role

	return user
}
