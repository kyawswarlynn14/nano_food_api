package helpers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"nano_food_api/models"

	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/option"
	"gopkg.in/gomail.v2"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func CheckPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

func InitializeFirebaseApp() (*firebase.App, error) {
	serviceAccountKeyPath := "../keys/se-reactjs-firebase-adminsdk.json"
	opt := option.WithCredentialsFile(serviceAccountKeyPath)

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %v", err)
	}

	return app, nil
}

func SendEmail(recept_mail string, subject string, body string) error {
	SMIP_HOST := os.Getenv("SMIP_HOST")
	SMIP_PORT, portErr := strconv.Atoi(os.Getenv("SMIP_PORT"))
	SMIP_MAIL := os.Getenv("SMIP_MAIL")
	SMIP_PASSWORD := os.Getenv("SMIP_PASSWORD")

	if portErr != nil {
		log.Printf("Error converting SMTP_PORT to integer: %v", portErr)
		return portErr
	}

	m := gomail.NewMessage()
	m.SetHeader("From", SMIP_MAIL)
	m.SetHeader("To", recept_mail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(SMIP_HOST, SMIP_PORT, SMIP_MAIL, SMIP_PASSWORD)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func GetUserIDFromMdw(c *gin.Context) (user_obj_id primitive.ObjectID, err error) {
	userIDFromMdw, exists := c.Get("userId")
	if !exists {
		return primitive.NilObjectID, fmt.Errorf("user ID not found in request context")
	}

	userIDStr, ok := userIDFromMdw.(string)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("invalid user ID format")
	}

	objID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid user ID")
	}

	return objID, nil
}

func GetUserInfoByID(user_id primitive.ObjectID, userCollection *mongo.Collection) (models.User, error) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	var user models.User
	err := userCollection.FindOne(ctx, bson.M{"_id": user_id}).Decode(&user)
	if err != nil {
		return models.User{}, err
	}

	if user.User_ID.IsZero() {
		return models.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}

func GetCurrentUser(c *gin.Context, userCollection *mongo.Collection) (models.User, error) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	userIDFromMdw, exists := c.Get("userId")
	if !exists {
		return models.User{}, fmt.Errorf("user ID not found in request context")
	}

	userIDStr, ok := userIDFromMdw.(string)
	if !ok {
		return models.User{}, fmt.Errorf("invalid user ID format")
	}

	objID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return models.User{}, fmt.Errorf("invalid user ID")
	}

	var user models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return models.User{}, err
	}

	if user.User_ID.IsZero() {
		return models.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}

func Contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func IsRestaurntMember(members []primitive.ObjectID, userID primitive.ObjectID) bool {
	isMember := false
	for _, member := range members {
		if member == userID {
			isMember = true
			break
		}
	}
	return isMember
}
