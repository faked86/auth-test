package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"test-authservice/pkg/db"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}

	pgUser := os.Getenv("POSTGRES_USER")
	pgPass := os.Getenv("POSTGRES_PASSWORD")
	pgHost := os.Getenv("POSTGRES_HOST")
	pgPort := os.Getenv("POSTGRES_PORT")
	pgDBname := os.Getenv("POSTGRES_DB")

	DBCreds := db.DBCredentials{
		Host:     pgHost,
		Port:     pgPort,
		User:     pgUser,
		Password: pgPass,
		DBName:   pgDBname,
	}

	database, err := db.NewPostgres(DBCreds)
	if err != nil {
		log.Fatal(err)
	}

	err = database.CreateTable()
	if err != nil {
		log.Fatal(err)
	}
	err = database.PopulateTable("56c3079c-f1c9-405f-85d7-574dd8c65771", "a@mail.com")
	if err != nil {
		log.Fatal(err)
	}
}
