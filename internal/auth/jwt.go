package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TODO
var mySecret = []byte("hello-world")

type Claims struct {
	UUID string
	Name string
	jwt.RegisteredClaims
}

// Return token string, jti string, error
func SignToken(uid, name string) (string, string, error) {
	jti := uuid.NewString()
	claims := &Claims{
		uid,
		name,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(144 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ktauth",
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySecret)
	return ss, jti, err
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return mySecret, nil
	})
	if err != nil {
		return nil, err
	} else if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	} else {
		return nil, fmt.Errorf("Failed to parse token")
	}
}
