package auth

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

// TokenVerifier xác thực chữ ký JWT (qua JWKS), kiểm tra exp/iss,
// rồi trích claims sub/email/name.
type TokenVerifier struct {
	issuer string
	jwks   *JWKSClient
}

func NewTokenVerifier(issuer string, jwks *JWKSClient) *TokenVerifier {
	return &TokenVerifier{issuer: issuer, jwks: jwks}
}

func (v *TokenVerifier) VerifyAccessToken(ctx context.Context, rawToken string) (*Claims, error) {
	set, err := v.jwks.KeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("load keys: %w", err)
	}

	token, err := jwt.Parse([]byte(rawToken),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithIssuer(v.issuer),
	)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	claims := &Claims{Sub: token.Subject()}
	if claims.Sub == "" {
		return nil, fmt.Errorf("verify token: missing sub claim")
	}
	if val, ok := token.Get("email"); ok {
		if s, ok := val.(string); ok {
			claims.Email = s
		}
	}
	if val, ok := token.Get("name"); ok {
		if s, ok := val.(string); ok {
			claims.Name = s
		}
	}
	return claims, nil
}
