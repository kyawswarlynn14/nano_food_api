package models

import (
	"errors"
	"time"
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
	User_ID          string    `json:"_id" bson:"_id"`
	Branch_ID        string    `json:"branch_id" bson:"branch_id"`
	Name             string    `json:"name" bson:"name"`
	Email            string    `json:"email" bson:"email"`
	Password         string    `json:"password" bson:"password"`
	Avatar           string    `json:"avatar" bson:"avatar"`
	Role             int       `json:"role" bson:"role"`
	Address          string    `json:"address" bson:"address"`
	Nrc              string    `json:"nrc" bson:"nrc"`
	Gender           string    `json:"gender" bson:"gender"`
	VerificationCode string    `json:"verification_code,omitempty" bson:"verification_code,omitempty"`
	IsVerified       bool      `json:"is_verified" bson:"is_verified"`
	T1               string    `json:"t1" bson:"t1"`
	T2               string    `json:"t2" bson:"t2"`
	Created_At       time.Time `json:"created_at" bson:"created_at"`
	Updated_At       time.Time `json:"updated_at" bson:"updated_at"`
}

/**
User Roles
0 - waiter or chef
1 - assistant
2 - manager
3 - owner
100 - root admin
**/

type Branch struct {
	Branch_ID   string    `json:"_id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	Address     string    `json:"address" bson:"address"`
	Contact     string    `json:"contact" bson:"contact"`
	Created_At  time.Time `json:"created_at" bson:"created_at"`
	Updated_At  time.Time `json:"updated_at" bson:"updated_at"`
}

type Category struct {
	Category_ID string    `json:"_id" bson:"_id"`
	Branch_ID   string    `json:"branch_id" bson:"branch_id"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	Created_At  time.Time `json:"created_at" bson:"created_at"`
	Updated_At  time.Time `json:"updated_at" bson:"updated_at"`
}

type AddOn struct {
	AddOn_ID    string  `json:"add_on_id" bson:"add_on_id"`
	Title       string  `json:"title" bson:"title"`
	Price       float64 `json:"price" bson:"price"`
	Cover       string  `json:"cover,omitempty" bson:"cover,omitempty"`
	Description string  `json:"description" bson:"description"`
	IsAvailable bool    `json:"is_available" bson:"is_available"`
}

type Menu struct {
	Menu_ID     string    `json:"_id" bson:"_id"`
	Branch_ID   string    `json:"branch_id" bson:"branch_id"`
	Category_ID string    `json:"category_id" bson:"category_id"`
	Title       string    `json:"title,omitempty" bson:"title,omitempty"`
	Short_Title string    `json:"short_title" bson:"short_title"`
	Description string    `json:"description" bson:"description"`
	Price       float64   `json:"price" bson:"price"`
	Cover       string    `json:"cover,omitempty" bson:"cover,omitempty"`
	Images      []string  `json:"images,omitempty" bson:"images,omitempty"`
	IsAvailable bool      `json:"is_available" bson:"is_available"`
	AddOns      []AddOn   `json:"add_ons,omitempty" bson:"add_ons,omitempty"`
	Created_At  time.Time `json:"created_at" bson:"created_at"`
	Updated_At  time.Time `json:"updated_at" bson:"updated_at"`
}
