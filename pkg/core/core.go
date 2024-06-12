package core

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"test-authservice/pkg/errs"
	"test-authservice/pkg/models"
)

const (
	accessTTL  = time.Minute * 30
	refreshTTL = time.Hour * 24 * 30
)

type Core struct {
	db        DB
	notifier  Notifier
	tokenizer Tokenizer
}

type DB interface {
	CheckUser(ctx context.Context, id string) (bool, error)
	GetUser(ctx context.Context, id string) (models.User, error)
	SetRefresh(ctx context.Context, id string, rHash []byte) (bool, error)
}

type Tokenizer interface {
	GeneratePair(id string, ip string, aExp time.Time, rExp time.Time) (models.Tokens, error)
	ParseToken(tokenStr string) (models.ParsedToken, error)
}

type Notifier interface {
	Send(addr string, msg string) error
}

func NewCore(db DB, notifier Notifier, tokenizer Tokenizer) *Core {
	return &Core{
		db:        db,
		notifier:  notifier,
		tokenizer: tokenizer,
	}
}

// Could return ErrNoSuchUser, ErrCouldNotSetRefresh
func (c *Core) IssueTokens(ctx context.Context, id string, ip string) (models.Tokens, error) {
	ok, err := c.db.CheckUser(ctx, id)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.IssueTokens: %w", err)
	}
	if !ok {
		return models.Tokens{}, errs.ErrNoSuchUser
	}

	res, err := c.tokenizer.GeneratePair(id, ip, time.Now().Add(accessTTL), time.Now().Add(refreshTTL))
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.IssueTokens: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(res.Refresh))
	refreshHash, err := bcrypt.GenerateFromPassword(h.Sum(nil), bcrypt.DefaultCost)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.IssueTokens: %w", err)
	}

	ok, err = c.db.SetRefresh(ctx, id, refreshHash)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.IssueTokens: %w", err)
	}
	if !ok {
		return models.Tokens{}, errs.ErrCouldNotSetRefresh
	}

	return res, nil
}

// Could return ErrInvalidRefresh, ErrNoSuchUser, ErrWrongRefresh, ErrCouldNotSetRefresh
func (c *Core) RefreshTokens(ctx context.Context, rToken string, ip string) (models.Tokens, error) {
	tokenInfo, err := c.tokenizer.ParseToken(rToken)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidToken) {
			return models.Tokens{}, errs.ErrInvalidRefresh
		}
		return models.Tokens{}, fmt.Errorf("Core.RefreshTokens: %w", err)
	}

	user, err := c.db.GetUser(ctx, tokenInfo.ID)
	if err != nil {
		if errors.Is(err, errs.ErrNoRows) {
			return models.Tokens{}, errs.ErrNoSuchUser
		}
		return models.Tokens{}, fmt.Errorf("Core.RefreshTokens: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(rToken))
	err = bcrypt.CompareHashAndPassword(user.RefreshToken, h.Sum(nil))
	if err != nil {
		return models.Tokens{}, errs.ErrWrongRefresh
	}

	aTime := time.Now().Add(accessTTL)
	rTime := time.Now().Add(refreshTTL)
	res, err := c.tokenizer.GeneratePair(user.ID, ip, aTime, rTime)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.RefreshTokens: %w", err)
	}

	h = sha256.New()
	h.Write([]byte(res.Refresh))
	refreshHash, err := bcrypt.GenerateFromPassword(h.Sum(nil), bcrypt.DefaultCost)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.RefreshTokens: %w", err)
	}

	ok, err := c.db.SetRefresh(ctx, user.ID, refreshHash)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("Core.RefreshTokens: %w", err)
	}
	if !ok {
		return models.Tokens{}, errs.ErrCouldNotSetRefresh
	}

	if tokenInfo.IP != ip {
		err = c.notifier.Send(user.Email, fmt.Sprintf("Auth from unknown IP address: %s", ip))
		if err != nil {
			slog.Warn("Could not send email", "email", user.Email, "error", err)
		}
	}
	return res, nil
}
