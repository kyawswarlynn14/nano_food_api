package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
var BranchCollection *mongo.Collection = database.BranchCollection

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
		user.User_ID = primitive.NewObjectID().Hex()
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

		var user models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if !helpers.Contains([]int{0, 1, 2}, user.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid role! Allowed roles are 0, 1, or 2."})
			return
		}

		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": user.Branch_ID}).Decode(&branch)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving branch"})
			return
		}

		// Validate email and password
		if !govalidator.IsEmail(user.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid email format!"})
			return
		}
		if len(user.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Password must be at least 6 characters!"})
			return
		}

		// Check if user already exists
		var existUser models.User
		err = UserCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&existUser)
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
		hashedPassword, hashErr := helpers.HashPassword(user.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error hashing password"})
			return
		}
		user.Password = hashedPassword

		// Create user
		user.User_ID = primitive.NewObjectID().Hex()
		user.Created_At = time.Now()
		user.Updated_At = time.Now()
		user.IsVerified = true

		_, err = UserCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Created user and added to branch successfully"})
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

		accessToken, err := token.TokenGenerator(user.Email, user.User_ID, user.Role)
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
		user, err := helpers.GetCurrentUser(c, UserCollection)
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

		user_id, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		var updateData struct {
			Name    string `json:"name"`
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

		updateFields := bson.M{}
		if updateData.Name != "" {
			updateFields["name"] = updateData.Name
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
			bson.M{"_id": user_id},
			bson.M{"$set": updateFields},
		)
		if err != nil {
			log.Printf("Error updating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user"})
			return
		}

		var updatedUser models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": user_id}).Decode(&updatedUser)
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage", "details": err.Error()})
			fmt.Printf("Firebase error >>> %v \n", err)
			return
		}

		userInfo, _ := helpers.GetCurrentUser(c, UserCollection)

		if userInfo.Avatar != "" {
			deleteErr := helpers.DeleteFileFromFirebase(client, userInfo.Avatar)
			if deleteErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete old avatar", "details": deleteErr.Error()})
				return
			}
		}

		avatarFile, avatarHeader, err := c.Request.FormFile("avatar")
		if err == nil {
			defer avatarFile.Close()
			avatarURL, uploadErr := helpers.UploadFileToFirebase(client, avatarFile, fmt.Sprintf("avatars/%d_%s", time.Now().Unix(), avatarHeader.Filename))
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload avatar", "details": uploadErr.Error()})
				return
			}

			user_id, err := helpers.GetUserIDFromMdw(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
				return
			}

			var ctxUpdate, cancelUpdate = context.WithTimeout(context.Background(), 100*time.Second)
			defer cancelUpdate()

			filter := bson.M{"_id": user_id}
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
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
	}
}

func UpdateUserPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		user_id, err := helpers.GetUserIDFromMdw(c)
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
		err = UserCollection.FindOne(ctx, bson.M{"_id": user_id}).Decode(&user)
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
			bson.M{"_id": user_id},
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
			User_ID   string `json:"user_id" binding:"required"`
			Role      int    `json:"role" binding:"required"`
			Branch_ID string `json:"branch_id" binding:"required"`
		}
		if err := c.BindJSON(&roleData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		// Fetch branch details
		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": roleData.Branch_ID}).Decode(&branch)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Branch not found"})
			return
		}

		var user models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": roleData.User_ID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}

		if user.Branch_ID != branch.Branch_ID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "User is not a member of this branch"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			if currentUser.Branch_ID != branch.Branch_ID {
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
			bson.M{"_id": roleData.User_ID},
			bson.M{"$set": bson.M{"role": roleData.Role, "updated_at": time.Now()}},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user role"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User role updated successfully"})
	}
}

func UpdateUserBranch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var reqBody struct {
			User_ID   string `json:"user_id" binding:"required"`
			Branch_ID string `json:"branch_id" binding:"required"`
		}
		if err := c.BindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		// Fetch branch details
		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": reqBody.Branch_ID}).Decode(&branch)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Branch not found"})
			return
		}

		var user models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": reqBody.User_ID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			if currentUser.Branch_ID != branch.Branch_ID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to update user branch"})
				return
			}
		}

		_, err = UserCollection.UpdateOne(
			ctx,
			bson.M{"_id": reqBody.User_ID},
			bson.M{"$set": bson.M{"branch_id": branch.Branch_ID, "updated_at": time.Now()}},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating user branch"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User branch updated successfully"})
	}
}

func DeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		User_ID := c.Param("user_id")
		Branch_ID := c.Param("branch_id")

		// Fetch branch details
		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": Branch_ID}).Decode(&branch)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Branch not found"})
			return
		}

		var user models.User
		err = UserCollection.FindOne(ctx, bson.M{"_id": User_ID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}

		if user.Branch_ID != branch.Branch_ID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "User is not a member of this restaurant"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			if currentUser.Branch_ID != branch.Branch_ID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to delete user"})
				return
			}
		}

		result, err := UserCollection.DeleteOne(ctx, bson.M{"_id": User_ID})
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

func GetAllBranchUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		branchID := c.Param("branch_id")

		// Fetch branch
		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": branchID}).Decode(&branch)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Branch not found"})
			return
		}

		currentUser, err := helpers.GetCurrentUser(c, UserCollection)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}

		if currentUser.Role != 100 {
			if currentUser.Branch_ID != branch.Branch_ID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You are not authorized to view branch users"})
				return
			}
		}

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"branch_id": branchID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "branches",
				"localField":   "branch_id",
				"foreignField": "_id",
				"as":           "branch",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$branch", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := UserCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving users"})
			return
		}
		defer cursor.Close(ctx)

		var users []bson.M
		if err := cursor.All(ctx, &users); err != nil {
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
			roleInt, _ := strconv.Atoi(role)
			filter["role"] = roleInt
		}

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: filter}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "branches",
				"localField":   "branch_id",
				"foreignField": "_id",
				"as":           "branch",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$branch", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := UserCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving users"})
			return
		}
		defer cursor.Close(ctx)

		var users []bson.M
		if err := cursor.All(ctx, &users); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding users"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Users retrieved successfully",
			"users":   users,
		})
	}
}
