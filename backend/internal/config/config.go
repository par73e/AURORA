package config

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
)

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		username := "postgres"
		if current, err := user.Current(); err == nil && current.Username != "" {
			username = current.Username
		}
		databaseURL = fmt.Sprintf("postgres://%s@localhost:5432/aurora?sslmode=disable", url.QueryEscape(username))
	}

	return Config{Port: port, DatabaseURL: databaseURL}
}
