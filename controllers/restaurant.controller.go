package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	database "nano_food_api/database"
	helpers "nano_food_api/helpers"
	models "nano_food_api/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var RestaurantCollection *mongo.Collection = database.RestaurantCollection

// CreateRestaurant: Root Admin only
func CreateRestaurant() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var restaurant models.Restaurant
		if err := c.BindJSON(&restaurant); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		restaurant.ID = primitive.NewObjectID()
		restaurant.Created_At = time.Now()
		restaurant.Updated_At = time.Now()

		_, err := RestaurantCollection.InsertOne(ctx, restaurant)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating restaurant"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Restaurant created successfully"})
	}
}

// GetRestaurants: Root Admin gets all, others get only their restaurants
func GetRestaurants() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		filter := bson.M{}
		if userInfo.Role != 100 {
			filter = bson.M{"owner_id": userInfo.User_ID}
		}

		var restaurants []models.Restaurant
		cursor, err := RestaurantCollection.Find(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving restaurants"})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &restaurants); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding restaurants"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Restaurants retrieved successfully",
			"data":    restaurants,
		})
	}
}

func GetOneRestaurant() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		restaurantID := c.Param("restaurnt_id")
		restaurnat_objID, err := primitive.ObjectIDFromHex(restaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurnat_objID}).Decode(&restaurant)
		if err != nil {
			log.Printf("Error retrieving restaurant: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving restaurant", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Restaurant retrieved successfully",
			"data":    restaurant,
		})
	}
}

// UpdateRestaurant: Root Admin and Owner can update restaurants
func UpdateRestaurant() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		restaurantID := c.Param("restaurant_id")
		objID, err := primitive.ObjectIDFromHex(restaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		var restaurant models.Restaurant
		if err := c.BindJSON(&restaurant); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		filter := bson.M{"_id": objID}
		if userInfo.Role != 100 {
			filter["owner_id"] = userInfo.User_ID
		}

		update := bson.M{
			"$set": bson.M{
				"name":       restaurant.Name,
				"address":    restaurant.Address,
				"contact":    restaurant.Contact,
				"updated_at": time.Now(),
			},
		}

		result, err := RestaurantCollection.UpdateOne(ctx, filter, update)
		if err != nil || result.MatchedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating restaurant"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Restaurant updated successfully"})
	}
}

// DeleteRestaurant: Root Admin only
func DeleteRestaurant() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		restaurantID := c.Param("restaurant_id")
		objID, err := primitive.ObjectIDFromHex(restaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurant ID"})
			return
		}

		_, err = RestaurantCollection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting restaurant"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Restaurant deleted successfully"})
	}
}

func AddRestaurantMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var input struct {
			UserID       string `json:"user_id" binding:"required"`
			RestaurantID string `json:"restaurant_id" binding:"required"`
		}
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		currentUserObjID, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		userRoleFromMdw, _ := c.Get("role")
		userRole, ok := userRoleFromMdw.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Invalid role format"})
			c.Abort()
			return
		}

		userObjID, err := primitive.ObjectIDFromHex(input.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		restaurantObjID, err := primitive.ObjectIDFromHex(input.RestaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurnat ID"})
			return
		}

		// Check if the restaurant exists
		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurantObjID}).Decode(&restaurant)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Restaurant not found"})
			return
		}

		if userRole != 100 && restaurant.Owner_ID != currentUserObjID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Access denied"})
			return
		}

		// Check if the user exists
		var user models.User
		err = database.UserCollection.FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}

		if helpers.IsRestaurntMember(restaurant.Members, userObjID) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "User is already in restaurant"})
			return
		}

		// Add the user to the restaurant's members list
		update := bson.M{"$addToSet": bson.M{"members": userObjID}}
		_, err = RestaurantCollection.UpdateOne(ctx, bson.M{"_id": restaurantObjID}, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error adding user to members"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User added to restaurant members successfully"})
	}
}

func RemoveRestaurantMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var input struct {
			UserID       string `json:"user_id" binding:"required"`
			RestaurantID string `json:"restaurant_id" binding:"required"`
		}
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		currentUserObjID, err := helpers.GetUserIDFromMdw(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		userRoleFromMdw, _ := c.Get("role")
		userRole, ok := userRoleFromMdw.(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Invalid role format"})
			c.Abort()
			return
		}

		userObjID, err := primitive.ObjectIDFromHex(input.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid user ID"})
			return
		}

		restaurantObjID, err := primitive.ObjectIDFromHex(input.RestaurantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid restaurnat ID"})
			return
		}

		// Check if the restaurant exists
		var restaurant models.Restaurant
		err = RestaurantCollection.FindOne(ctx, bson.M{"_id": restaurantObjID}).Decode(&restaurant)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Restaurant not found"})
			return
		}

		if userRole != 100 && restaurant.Owner_ID != currentUserObjID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Access denied"})
			return
		}

		// Check if the user exists
		var user models.User
		err = database.UserCollection.FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "User not found"})
			return
		}

		if !helpers.IsRestaurntMember(restaurant.Members, userObjID) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "User is not in restaurant"})
			return
		}

		update := bson.M{"$pull": bson.M{"members": userObjID}}
		_, err = RestaurantCollection.UpdateOne(ctx, bson.M{"_id": restaurantObjID}, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error removing user from members"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User removed from restaurant members successfully"})
	}
}
