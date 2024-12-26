package controllers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	database "nano_food_api/database"
	helpers "nano_food_api/helpers"
	models "nano_food_api/models"
	token "nano_food_api/tokens"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var UserCollection *mongo.Collection = database.UserCollection

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

		verificationCode := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
		user.VerificationCode = verificationCode
		user.IsVerified = false
		user.User_ID = primitive.NewObjectID()
		user.Role = 0
		user.Created_At = time.Now()
		user.Updated_At = time.Now()

		subject := "Your Verification Code From NanoFood"
		body := fmt.Sprintf("Your verification code is: <b>%s</b>", verificationCode)
		if emailErr := helpers.SendEmail(user.Email, subject, body); emailErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to send verification email"})
			return
		}

		_, err = UserCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating user"})
			return
		}

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "User created successfully. Please check your email to verify.",
			},
		)
	}
}

func VerifyUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		type VerificationRequest struct {
			Email            string `json:"email"`
			VerificationCode string `json:"verification_code"`
		}

		var request VerificationRequest
		if err := c.BindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		var user models.User
		err := UserCollection.FindOne(ctx, bson.M{"email": request.Email}).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
				return
			}
			log.Printf("Error retrieving user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving user"})
			return
		}

		if user.VerificationCode != request.VerificationCode {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid verification code"})
			return
		}

		update := bson.M{"$set": bson.M{"is_verified": true, "verification_code": ""}}
		_, updateErr := UserCollection.UpdateOne(ctx, bson.M{"_id": user.User_ID}, update)
		if updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to verify user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User verified successfully"})
	}
}

func CreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var input struct {
			RestaurantID string `json:"restaurant_id" binding:"required"`
			User         models.User
		}
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		restaurantObjID, err := primitive.ObjectIDFromHex(input.RestaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		if !helpers.Contains([]int{0, 1, 2}, input.User.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid role! Allowed roles are 0, 1, or 2."})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurantObjID}).Decode(&restaurant)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving restaurant"})
			return
		}

		if currentUser.Role == 3 && !helpers.IsRestaurntMember(restaurant.Members, currentUser.User_ID) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to create a user for this restaurant"})
			return
		}

		// Validate email and password
		if !govalidator.IsEmail(input.User.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid email format!"})
			return
		}
		if len(input.User.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Password must be at least 6 characters!"})
			return
		}

		// Check if user already exists
		var existUser models.User
		err = UserCollection.FindOne(ctx, bson.M{"email": input.User.Email}).Decode(&existUser)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "User already exists!"})
			return
		}
		if err != mongo.ErrNoDocuments {
			log.Printf("Error retrieving user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving user"})
			return
		}

		// Hash the password
		hashedPassword, hashErr := helpers.HashPassword(input.User.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error hashing password"})
			return
		}
		input.User.Password = hashedPassword

		// Create user
		input.User.User_ID = primitive.NewObjectID()
		input.User.Created_At = time.Now()
		input.User.Updated_At = time.Now()

		_, err = UserCollection.InsertOne(ctx, input.User)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating user"})
			return
		}

		// Add user to the restaurant's members
		updateResult, err := RestaurantCollection.UpdateOne(
			ctx,
			bson.M{"_id": restaurantObjID},
			bson.M{"$addToSet": bson.M{"members": input.User.User_ID}},
		)
		if err != nil || updateResult.MatchedCount == 0 {
			log.Printf("Error updating restaurant: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error adding user to restaurant members"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Created user and added to restaurant successfully"})
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
		user_obj_id, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": user_obj_id}).Decode(&user)
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

		user_obj_id, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
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
			if err == nil && existingUser.User_ID != user_obj_id {
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
			bson.M{"_id": user_obj_id},
			bson.M{"$set": updateFields},
		)
		if err != nil {
			log.Printf("Error updating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user"})
			return
		}

		var updatedUser models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": user_obj_id}).Decode(&updatedUser)
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

func UploadAvatar() gin.HandlerFunc {
	return func(c *gin.Context) {
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to initialize Firebase"})
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage"})
			return
		}

		file, header, err := c.Request.FormFile("avatar")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Failed to retrieve avatar file"})
			return
		}
		defer file.Close()

		bucket, err := client.DefaultBucket()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to get default bucket"})
			return
		}
		filename := fmt.Sprintf("avatars/%d_%s", time.Now().Unix(), header.Filename)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
		defer cancel()

		writer := bucket.Object(filename).NewWriter(ctx)
		writer.ContentType = header.Header.Get("Content-Type")

		if _, err := io.Copy(writer, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload avatar"})
			return
		}

		if err := writer.Close(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to finalize avatar upload"})
			return
		}

		bucketAttrs, err := bucket.Attrs(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to get bucket attributes"})
			return
		}
		avatarURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketAttrs.Name, filename)

		user_obj_id, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		var ctxUpdate, cancelUpdate = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancelUpdate()

		filter := bson.M{"_id": user_obj_id}
		update := bson.M{"$set": bson.M{"avatar": avatarURL, "updated_at": time.Now()}}
		_, err = UserCollection.UpdateOne(ctxUpdate, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update user avatar"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"message":   "Avatar uploaded and updated successfully",
			"avatarURL": avatarURL,
		})
	}
}

func UpdateUserPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		user_obj_id, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
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
		err = UserCollection.FindOne(ctx, bson.M{"_id": user_obj_id}).Decode(&user)
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
			bson.M{"_id": user_obj_id},
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
			UserID       string `json:"user_id" binding:"required"`
			Role         int    `json:"role" binding:"required"`
			RestaurantID string `json:"restaurant_id" binding:"required"`
		}
		if err := c.BindJSON(&roleData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		userObjID, err := primitive.ObjectIDFromHex(roleData.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		restaurantObjID, err := primitive.ObjectIDFromHex(roleData.RestaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		// Fetch restaurant details
		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurantObjID}).Decode(&restaurant)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Restaurant not found"})
			return
		}

		isMember := helpers.IsRestaurntMember(restaurant.Members, userObjID)
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "User is not a member of this restaurant"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			isMember := helpers.IsRestaurntMember(restaurant.Members, currentUser.User_ID)
			if !isMember {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to update user role"})
				return
			}
		}

		var availableRoles = []int{0, 1}
		if currentUser.Role == 3 {
			availableRoles = append(availableRoles, 2)
		} else if currentUser.Role == 100 {
			availableRoles = append(availableRoles, 2, 3)
		}
		if !helpers.Contains(availableRoles, roleData.Role) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to update user to this role"})
		}

		_, err = UserCollection.UpdateOne(
			ctx,
			bson.M{"_id": userObjID},
			bson.M{"$set": bson.M{"role": roleData.Role, "updated_at": time.Now()}},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user role"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User role updated successfully"})
	}
}

func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var reqBody struct {
			UserID       string `json:"user_id" binding:"required"`
			RestaurantID string `json:"restaurant_id" binding:"required"`
		}
		if err := c.BindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		userObjID, err := primitive.ObjectIDFromHex(reqBody.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		restaurantObjID, err := primitive.ObjectIDFromHex(reqBody.RestaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		// Fetch restaurant details
		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurantObjID}).Decode(&restaurant)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Restaurant not found"})
			return
		}

		isMember := helpers.IsRestaurntMember(restaurant.Members, userObjID)
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "User is not a member of this restaurant"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			isMember := helpers.IsRestaurntMember(restaurant.Members, currentUser.User_ID)
			if !isMember {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to update user role"})
				return
			}
		}

		result, err := UserCollection.DeleteOne(ctx, bson.M{"_id": userObjID})
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

func GetAllRestaurantUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		restaurantID := c.Param("restaurant_id")

		// Validate restaurant ID
		restaurantObjID, err := primitive.ObjectIDFromHex(restaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		// Fetch restaurant
		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurantObjID}).Decode(&restaurant)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Restaurant not found"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			isMember := helpers.IsRestaurntMember(restaurant.Members, currentUser.User_ID)
			if !isMember {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to view restaurant users"})
				return
			}
		}

		// Find all users who are members of the restaurant
		cursor, err := UserCollection.Find(ctx, bson.M{"_id": bson.M{"$in": restaurant.Members}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error fetching users"})
			return
		}
		defer cursor.Close(ctx)

		// Decode users into a slice
		var users []models.User
		if err = cursor.All(ctx, &users); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding users"})
			return
		}

		// Return the list of users
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    users,
		})
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
