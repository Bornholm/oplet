package config

type I18n struct {
	DefaultLanguage string `env:"DEFAULT_LANGUAGE,expand" envDefault:"en"`
}
