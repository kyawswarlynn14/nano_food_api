package helpers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"nano_food_api/models"

	cloudStorage "cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/storage"
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
	serviceAccountKeyPath := "keys/se-reactjs-firebase-adminsdk.json"
	opt := option.WithCredentialsFile(serviceAccountKeyPath)

	config := &firebase.Config{
		StorageBucket: os.Getenv("STORAGE_BUCKET"),
	}

	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %v", err)
	}

	return app, nil
}

func UploadFileToFirebase(client *storage.Client, file io.Reader, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	bucket, err := client.DefaultBucket()
	if err != nil {
		return "", err
	}

	writer := bucket.Object(filename).NewWriter(ctx)
	writer.ContentType = "image/" + strings.Split(filename, ".")[1]

	if _, err := io.Copy(writer, file); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	object := bucket.Object(filename)
	acl := object.ACL()
	if err := acl.Set(ctx, cloudStorage.AllUsers, cloudStorage.RoleReader); err != nil {
		return "", fmt.Errorf("failed to set public read access: %v", err)
	}

	bucketAttrs, err := bucket.Attrs(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketAttrs.Name, filename), nil
}

func DeleteFileFromFirebase(client *storage.Client, filePath string) error {
	ctx := context.Background()

	bucket, err := client.DefaultBucket()
	if err != nil {
		return fmt.Errorf("failed to get default bucket: %v", err)
	}

	storageBucket := os.Getenv("STORAGE_BUCKET") + "/"

	newFilePatch := strings.Split(filePath, storageBucket)[1]
	fmt.Println("Deleting file from path: ", newFilePatch)

	object := bucket.Object(newFilePatch)

	err = object.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete file from Firebase Storage: %v", err)
	}

	return nil
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

func GetUserIDFromMdw(c *gin.Context) (user_obj_id string, err error) {
	userIDFromMdw, exists := c.Get("userId")
	if !exists {
		return primitive.NilObjectID.String(), fmt.Errorf("user ID not found in request context")
	}

	userIDStr, ok := userIDFromMdw.(string)
	if !ok {
		return primitive.NilObjectID.String(), fmt.Errorf("invalid user ID format")
	}

	return userIDStr, nil
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

	var user models.User
	err := userCollection.FindOne(ctx, bson.M{"_id": userIDStr}).Decode(&user)
	if err != nil {
		return models.User{}, err
	}

	if user.User_ID == "" {
		return models.User{}, fmt.Errorf("user not found")
	}

	return user, nil
}

func CheckDataExist(ctx context.Context, collection *mongo.Collection, filter bson.M) (bool, error) {
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func Contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func ParseFloat(value string) float64 {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0
	}
	return result
}
