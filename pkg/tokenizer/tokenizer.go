package tokenizer

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"test-authservice/pkg/errs"
	"test-authservice/pkg/models"
)

type JWTTokenizer struct {
	signingKey []byte
}

type TokenClaims struct {
	IP string
	jwt.RegisteredClaims
}

func NewTokenizer(signingKey []byte) *JWTTokenizer {
	return &JWTTokenizer{
		signingKey: signingKey,
	}
}

func (t *JWTTokenizer) GenerateToken(id string, ip string, exp time.Time) (string, error) {
	claims := TokenClaims{
		ip,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        id,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	ss, err := token.SignedString(t.signingKey)
	if err != nil {
		return "", fmt.Errorf("Tokenizer.GenerateToken: %w", err)
	}
	return ss, nil
}

func (t *JWTTokenizer) GeneratePair(id string, ip string, aExp time.Time, rExp time.Time) (models.Tokens, error) {
	access, err := t.GenerateToken(id, ip, aExp)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("JWTtokenizer.GeneratePair: %w", err)
	}

	refresh, err := t.GenerateToken(id, ip, rExp)
	if err != nil {
		return models.Tokens{}, fmt.Errorf("JWTtokenizer.GeneratePair: %w", err)
	}

	return models.Tokens{
		Access:  access,
		Refresh: refresh,
	}, nil
}

func (t *JWTTokenizer) ParseToken(tokenStr string) (models.ParsedToken, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return t.signingKey, nil
	})

	if err != nil {
		return models.ParsedToken{}, err
	}
	switch {
	case token.Valid:
		if claims, ok := token.Claims.(*TokenClaims); ok {
			return models.ParsedToken{
				IP:  claims.IP,
				ID:  claims.ID,
				Exp: claims.ExpiresAt.Time,
			}, nil
		} else {
			return models.ParsedToken{}, errs.ErrInvalidToken
		}
	case errors.Is(err, jwt.ErrTokenMalformed):
		return models.ParsedToken{}, errs.ErrInvalidToken
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return models.ParsedToken{}, errs.ErrInvalidToken
	case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
		return models.ParsedToken{}, errs.ErrInvalidToken
	default:
		return models.ParsedToken{}, fmt.Errorf("Tokenizer.ParseToken: %w", err)
	}
}
