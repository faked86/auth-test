package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"test-authservice/pkg/core"
	"test-authservice/pkg/db"
	"test-authservice/pkg/notifier"
	"test-authservice/pkg/server"
	"test-authservice/pkg/tokenizer"
)

func run(ctx context.Context) error {
	mainCtx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := godotenv.Load(".env"); err != nil {
		return err
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
		return err
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	senderEmail := os.Getenv("SENDER_EMAIL")
	senderPassword := os.Getenv("SENDER_PASSWORD")
	email := notifier.NewEmail(smtpHost, smtpPort, senderEmail, senderPassword)
	JWTTokenizer := tokenizer.NewTokenizer([]byte(os.Getenv("SIGNING_KEY")))

	appCore := core.NewCore(database, email, JWTTokenizer)

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		return err
	}

	httpServer := server.NewServer(mainCtx, port, appCore)

	g, gCtx := errgroup.WithContext(mainCtx)
	g.Go(func() error {
		slog.Info("Server listens and serve")
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		slog.Info("Server stopped listening...")
		mainCtx, cancel = context.WithTimeout(mainCtx, time.Second)
		defer cancel()
		return httpServer.Shutdown(mainCtx)
	})

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
