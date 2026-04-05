package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("auth: invalid or expired token")
)

// JWTManager handles creation and validation of JWT tokens.
type JWTManager struct {
	secret []byte
	expiry time.Duration
}

// NewJWTManager creates a JWTManager with the given HMAC secret and token expiry duration.
func NewJWTManager(secret []byte, expiry time.Duration) *JWTManager {
	return &JWTManager{
		secret: secret,
		expiry: expiry,
	}
}

// Issue creates a signed JWT for the given user ID and email.
// The token contains sub (userID as string), email, exp, and iat claims.
func (j *JWTManager) Issue(userID int64, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   strconv.FormatInt(userID, 10),
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(j.expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

// Claims holds the user identity extracted from a validated JWT.
type Claims struct {
	UserID int64
	Email  string
}

// Validate parses and validates a JWT token string.
// Returns the extracted claims if the token is valid, or an error otherwise.
func (j *JWTManager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	sub, ok := mapClaims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	email, ok := mapClaims["email"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	return &Claims{
		UserID: userID,
		Email:  email,
	}, nil
}
