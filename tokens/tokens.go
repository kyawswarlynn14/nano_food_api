package tokens

import (
	"os"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type SignedDetails struct {
	Email   string
	User_ID string
	Role    int
	jwt.StandardClaims
}

var SECRET_KEY = os.Getenv("SECRET_KEY")
var TOKEN_EXPIRE = os.Getenv("TOKEN_EXPIRE")
var TOKEN_EXPIRE_INT int

func TokenGenerator(email string, userId string, role int) (signedtoken string, err error) {
	TOKEN_EXPIRE_INT, err := strconv.Atoi(TOKEN_EXPIRE)

	if err != nil {
		TOKEN_EXPIRE_INT = 1
	}

	claims := &SignedDetails{
		Email:   email,
		User_ID: userId,
		Role:    role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Duration(TOKEN_EXPIRE_INT) * time.Hour * 24).Unix(),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}
	return token, err
}

func ValidateToken(signedtoken string) (claims *SignedDetails, msg string) {
	token, err := jwt.ParseWithClaims(signedtoken, &SignedDetails{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SECRET_KEY), nil
	})

	if err != nil {
		msg = err.Error()
		return
	}
	claims, ok := token.Claims.(*SignedDetails)
	if !ok {
		msg = "The Token is invalid"
		return
	}
	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = "token is expired"
		return
	}
	return claims, msg
}
