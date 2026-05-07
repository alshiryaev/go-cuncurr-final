package main

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	Dsn           string
	Redis         string
	MailUrlSecret string
	Port          string
}

func GetEnv() (Env, error) {
	err := godotenv.Load()
	if err != nil {
		return Env{}, err
	}
	return Env{
		Dsn:           os.Getenv("DSN"),
		Redis:         os.Getenv("REDIS"),
		MailUrlSecret: os.Getenv("MAIL_URL_SECRET"),
		Port:          os.Getenv("PORT"),
	}, nil
}
