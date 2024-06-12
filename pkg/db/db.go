package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"test-authservice/pkg/errs"
	"test-authservice/pkg/models"

	_ "github.com/lib/pq"
)

type DBCredentials struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type PostgresDB struct {
	Conn *sql.DB
}

func NewPostgres(creds DBCredentials) (*PostgresDB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		creds.Host, creds.Port, creds.User, creds.Password, creds.DBName,
	)
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = Ping(context.Background(), conn); err != nil {
		return nil, err
	}

	res := PostgresDB{Conn: conn}
	return &res, nil
}

// Checks if user with "id" exists in db
func (db *PostgresDB) CheckUser(ctx context.Context, id string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var flag int
	err := db.Conn.QueryRowContext(ctx, `
		select
			1
		from
			users
		where
			users.id = $1;
		`,
		id,
	).Scan(&flag)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("DB.CheckUser: %w", err)
	}
	return true, nil
}

// Gets User model from db.
// Returns empty User if no such user.
func (db *PostgresDB) GetUser(ctx context.Context, id string) (models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	u := models.User{
		ID: id,
	}
	err := db.Conn.QueryRowContext(
		ctx,
		`
		select
			users.refresh_token, users.email
		from
			users
		where
			users.id = $1;
		`,
		id,
	).Scan(&u.RefreshToken, &u.Email)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, errs.ErrNoRows
		}
		return models.User{}, fmt.Errorf("DB.GetUser: %w", err)
	}

	return u, nil
}

// Gets refresh token from db.
// Returns empty []byte if no such user.
func (db *PostgresDB) GetRefresh(ctx context.Context, id string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	r := make([]byte, 16)
	err := db.Conn.QueryRowContext(
		ctx,
		`
		select
			users.refresh_token
		from
			users
		where
			users.id = $1;
		`,
		id,
	).Scan(&r)

	if err != nil {
		if err == sql.ErrNoRows {
			return r, nil
		}
		return r, fmt.Errorf("DB.GetRefresh: %w", err)
	}

	return r, nil
}

// Sets hash of refresh token to user.
// Returns bool to show if token setted.
func (db *PostgresDB) SetRefresh(ctx context.Context, id string, rHash []byte) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := db.Conn.ExecContext(
		ctx,
		`
		update
			users
		set
			refresh_token = $1
		where
			users.id = $2;
		`,
		rHash,
		id,
	)
	if err != nil {
		return false, fmt.Errorf("db.SetRefresh: %w", err)
	}

	ra, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("db.SetRefresh: %w", err)
	}
	if ra < 1 {
		return false, nil
	}
	return true, nil
}

// Gets user email from db.
// Returns empty string if no such user.
func (db *PostgresDB) GetEmail(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var ext string
	err := db.Conn.QueryRowContext(
		ctx,
		`
		select
			users.email
		from
			users
		where
			users.id = $1;
		`,
		id,
	).Scan(&ext)

	if err != nil {
		if err == sql.ErrNoRows {
			return ext, nil
		}
		return ext, fmt.Errorf("DB.GetEmail: %w", err)
	}

	return ext, nil
}

func (db *PostgresDB) CreateTable() error {
	_, err := db.Conn.Exec(
		`
		CREATE TABLE IF NOT EXISTS users (
			id            UUID PRIMARY KEY,
			refresh_token bytea,
			email         varchar(255)
		);
		`,
	)
	if err != nil {
		fmt.Println("Error creating")
	}

	return err
}

func (db *PostgresDB) PopulateTable(id string, email string) error {
	_, err := db.Conn.Exec(
		`
		INSERT INTO users (id, email)
			VALUES ($1, $2); 
		`,
		id,
		email,
	)
	if err != nil {
		return err
	}
	return nil
}

func Ping(ctx context.Context, conn *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return err
	}
	return nil
}
