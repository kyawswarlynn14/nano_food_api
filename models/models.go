package models

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TGender string

const (
	Male   TGender = "001"
	Femail TGender = "002"
)

func (g TGender) IsValid() error {
	if g != Male && g != Femail {
		return errors.New("invalid gender type: must be '001' or '002'")
	}
	return nil
}

type User struct {
	User_ID    primitive.ObjectID `json:"_id" bson:"_id"`
	Name       string             `json:"name" bson:"name"`
	Email      string             `json:"email" bson:"email"`
	Password   string             `json:"password" bson:"password"`
	Avatar     string             `json:"avatar" bson:"avatar"`
	Role       int                `json:"role" bson:"role"`
	Address    string             `json:"address" bson:"address"`
	Nrc        string             `json:"nrc" bson:"nrc"`
	Gender     string             `json:"gender" bson:"gender"`
	T1         string             `json:"t1" bson:"t1"`
	T2         string             `json:"t2" bson:"t2"`
	Created_At time.Time          `json:"created_at" bson:"created_at"`
	Updated_At time.Time          `json:"updated_at" bson:"updated_at"`
}