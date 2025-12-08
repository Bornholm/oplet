package config

type Storage struct {
	Database Database `envPrefix:"DATABASE_"`
	File     File     `envPrefix:"FILE_"`
}

type Database struct {
	DSN string `env:"DSN,expand" envDefault:"data/store.sqlite"`
}

type File struct {
	Dir string `env:"DIR,expand" envDefault:"data/files"`
}
