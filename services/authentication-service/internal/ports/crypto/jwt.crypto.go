package crypto

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID       uuid.UUID `json:"sub"`
	UserEmail    string    `json:"email"`
	IsSuperAdmin bool      `json:"is_super_admin"`
	Role         string    `json:"role,omitempty"`
	Issuer       string    `json:"iss"`
	Audience     string    `json:"aud"`
	ExpiresAt    int64     `json:"exp"`
	IssuedAt     int64     `json:"iat"`
	JTI          string    `json:"jti"`
}

type HMACTokenSigner struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

func NewHMACTokenSigner(secret, issuer, audience string, ttl time.Duration) *HMACTokenSigner {
	return &HMACTokenSigner{
		secret:   []byte(secret),
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}
}

func (s *HMACTokenSigner) SignAccessToken(ctx context.Context, claims AccessClaims) (string, time.Duration, error) {
	now := time.Now().UTC()
	jwtClaims := JWTClaims{
		UserID:       claims.UserID,
		UserEmail:    claims.UserEmail,
		IsSuperAdmin: claims.IsSuperAdmin,
		Role:         claims.Role,
		Issuer:       s.issuer,
		Audience:     s.audience,
		IssuedAt:     now.Unix(),
		ExpiresAt:    now.Add(s.ttl).Unix(),
		JTI:          uuid.NewString(),
	}
	token, err := s.sign(jwtClaims)
	return token, s.ttl, err
}

func (s *HMACTokenSigner) VerifyAccessToken(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	expected := signBytes([]byte(signingInput), s.secret)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(expected, actual) {
		return nil, errors.New("invalid token signature")
	}

	var claims JWTClaims
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Unix()
	if claims.ExpiresAt <= now {
		return nil, errors.New("token expired")
	}
	if claims.Issuer != s.issuer || claims.Audience != s.audience {
		return nil, errors.New("token issuer or audience mismatch")
	}
	return &claims, nil
}

func (s *HMACTokenSigner) sign(claims JWTClaims) (string, error) {
	headerBytes, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claimBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedPayload := base64.RawURLEncoding.EncodeToString(claimBytes)
	signingInput := encodedHeader + "." + encodedPayload
	signature := base64.RawURLEncoding.EncodeToString(signBytes([]byte(signingInput), s.secret))
	return fmt.Sprintf("%s.%s", signingInput, signature), nil
}

func signBytes(input, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(input)
	return mac.Sum(nil)
}
