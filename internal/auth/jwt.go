package auth

import (
	"fmt"
	"time"

	"github.com/BygTTeKep/AiDreamsStudio/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	cfg    *config.JWTConfig
	secret []byte
}

type claims struct {
	UserID int    `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewJWT(cfg *config.JWTConfig) *JWT {
	return &JWT{
		cfg:    cfg,
		secret: []byte(cfg.SecretKey),
	}
}

func (j *JWT) GenerateToken(userid int, email string) (string, error) {
	claims := claims{
		UserID: userid,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(j.cfg.ExpiresHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "aidreamsstudio",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (j *JWT) ValidateToken(tokenString string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
