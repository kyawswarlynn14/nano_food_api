package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	database "github.com/kyawswarlynn14/nano_food_api/database"
	helpers "github.com/kyawswarlynn14/nano_food_api/helpers"
	models "github.com/kyawswarlynn14/nano_food_api/models"
	token "github.com/kyawswarlynn14/nano_food_api/tokens"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var UserCollection *mongo.Collection = database.NanoFoodData(database.Client, "Users")

func RegisterUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if !govalidator.IsEmail(user.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid email format!"})
			return
		}
		if len(user.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid password!"})
			return
		}

		var exist_user models.User
		err := UserCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&exist_user)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "User already exists!"})
			return
		}
		if err != mongo.ErrNoDocuments {
			log.Printf("Error retrieving user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving user"})
			return
		}

		hashedPassword, hashErr := helpers.HashPassword(user.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error hashing password"})
			return
		}
		user.Password = hashedPassword

		user.User_ID = primitive.NewObjectID()
		user.Role = 0
		user.Created_At = time.Now()
		user.Updated_At = time.Now()

		_, err = UserCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "User created successfully"})
	}
}

func CreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if !govalidator.IsEmail(user.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid email format!"})
			return
		}
		if len(user.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid password!"})
			return
		}

		var exist_user models.User
		err := UserCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&exist_user)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "User already exists!"})
			return
		}
		if err != mongo.ErrNoDocuments {
			log.Printf("Error retrieving user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving user"})
			return
		}

		hashedPassword, hashErr := helpers.HashPassword(user.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error hashing password"})
			return
		}
		user.Password = hashedPassword

		user.User_ID = primitive.NewObjectID()
		user.Created_At = time.Now()
		user.Updated_At = time.Now()

		_, err = UserCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Created user successfully"})
	}
}

func LoginUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var loginData struct {
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.BindJSON(&loginData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		var user models.User
		err := UserCollection.FindOne(ctx, bson.M{"email": loginData.Email}).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid email"})
				return
			}
			log.Printf("Error finding user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Internal server error"})
			return
		}

		isValidPassword := helpers.CheckPassword(user.Password, loginData.Password)
		if !isValidPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid password"})
			return
		}

		accessToken, err := token.TokenGenerator(user.Email, user.User_ID.Hex(), user.Role)
		if err != nil {
			log.Printf("Error generating token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error generating token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"message":     "Login successful",
			"accessToken": accessToken,
			"user":        user,
		})
	}
}

func GetCurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDFromMdw, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized"})
			return
		}
		log.Printf("userId from mdw: %v", userIDFromMdw)

		userIDStr, ok := userIDFromMdw.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Invalid user ID format"})
			return
		}
		objID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid expense item ID"})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
		if err != nil {
			log.Printf("Error retrieving user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving user", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User retrieved successfully",
			"users":   user,
		})
	}
}

func UpdateUserInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		userIDFromMdw, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User ID not found in request context"})
			return
		}

		userIDStr, ok := userIDFromMdw.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Invalid user ID format"})
			return
		}

		objID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		var updateData struct {
			Name    string `json:"name"`
			Email   string `json:"email"`
			Avatar  string `json:"avatar"`
			Address string `json:"address"`
			Nrc     string `json:"nrc"`
			Gender  string `json:"gender"`
			T1      string `json:"t1"`
			T2      string `json:"t2"`
		}
		if err := c.BindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if updateData.Email != "" {
			var existingUser models.User
			err := UserCollection.FindOne(ctx, bson.M{"email": updateData.Email}).Decode(&existingUser)
			if err == nil && existingUser.User_ID != objID {
				c.JSON(http.StatusConflict, gin.H{"success": false, "error": "Email is already in use"})
				return
			}
			if err != nil && err != mongo.ErrNoDocuments {
				log.Printf("Error checking email: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error checking email"})
				return
			}
		}

		updateFields := bson.M{}
		if updateData.Name != "" {
			updateFields["name"] = updateData.Name
		}
		if updateData.Email != "" {
			updateFields["email"] = updateData.Email
		}
		if updateData.Avatar != "" {
			updateFields["avatar"] = updateData.Avatar
		}
		if updateData.Address != "" {
			updateFields["address"] = updateData.Address
		}
		if updateData.Nrc != "" {
			updateFields["nrc"] = updateData.Nrc
		}
		if updateData.Gender != "" {
			updateFields["gender"] = updateData.Gender
		}
		updateFields["updated_at"] = time.Now()

		_, err = UserCollection.UpdateOne(
			ctx,
			bson.M{"_id": objID},
			bson.M{"$set": updateFields},
		)
		if err != nil {
			log.Printf("Error updating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user"})
			return
		}

		var updatedUser models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedUser)
		if err != nil {
			log.Printf("Error retrieving updated user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving updated user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User updated successfully",
			"user":    updatedUser,
		})
	}
}

func UpdateUserPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		userIDFromMdw, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User ID not found in request context"})
			return
		}

		userIDStr, ok := userIDFromMdw.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Invalid user ID format"})
			return
		}

		objID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		var passwordData struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required"`
		}
		if err := c.BindJSON(&passwordData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		var user models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not found"})
			return
		}

		if !helpers.CheckPassword(user.Password, passwordData.CurrentPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Current password is incorrect"})
			return
		}

		hashedPassword, err := helpers.HashPassword(passwordData.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error hashing new password"})
			return
		}

		_, err = UserCollection.UpdateOne(
			ctx,
			bson.M{"_id": objID},
			bson.M{"$set": bson.M{"password": hashedPassword, "updated_at": time.Now()}},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password updated successfully"})
	}
}

func UpdateUserRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var roleData struct {
			UserID string `json:"user_id" binding:"required"`
			Role   int    `json:"role" binding:"required"`
		}
		if err := c.BindJSON(&roleData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		objID, err := primitive.ObjectIDFromHex(roleData.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		_, err = UserCollection.UpdateOne(
			ctx,
			bson.M{"_id": objID},
			bson.M{"$set": bson.M{"role": roleData.Role, "updated_at": time.Now()}},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user role"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User role updated successfully"})
	}
}

func GetAllUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		filter := bson.M{}

		role := c.Query("role")
		if role != "" {
			filter["role"] = role
		}

		var users []models.User
		cursor, err := UserCollection.Find(ctx, filter)
		if err != nil {
			log.Printf("Error finding users: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving users"})
			return
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var user models.User
			if err := cursor.Decode(&user); err != nil {
				log.Printf("Error decoding user: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding user"})
				return
			}
			users = append(users, user)
		}

		if err := cursor.Err(); err != nil {
			log.Printf("Cursor error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error iterating over users"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Users retrieved successfully",
			"users":   users,
		})
	}
}

func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		userID := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		result, err := UserCollection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			log.Printf("Error deleting user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting user"})
			return
		}

		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User deleted successfully",
		})
	}
}
