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

var CategoryCollection *mongo.Collection = database.CategoryCollection

func CreateCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var category models.Category
		if err := c.BindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": category.Branch_ID}).Decode(&branch)
		if err != nil {
			log.Printf("Error retrieving branch: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving branch", "details": err.Error()})
			return
		}

		category.Category_ID = primitive.NewObjectID().Hex()
		category.Created_At = time.Now()
		category.Updated_At = time.Now()

		_, err = CategoryCollection.InsertOne(ctx, category)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating category"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Category created successfully"})
	}
}

func GetAllCategories() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		branchID := c.Param("branch_id")

		// MongoDB Aggregation Pipeline
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

		cursor, err := CategoryCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving categories"})
			return
		}
		defer cursor.Close(ctx)

		var categories []bson.M
		if err := cursor.All(ctx, &categories); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding categories"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Categories retrieved successfully",
			"data":    categories,
		})
	}
}

func GetOneCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		categoryID := c.Param("category_id")

		// MongoDB Aggregation Pipeline
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"_id": categoryID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "branches",
				"localField":   "branch_id",
				"foreignField": "_id",
				"as":           "branch",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$branch", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := CategoryCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving category"})
			return
		}
		defer cursor.Close(ctx)

		var categories []bson.M
		if err := cursor.All(ctx, &categories); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding category"})
			return
		}

		if len(categories) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Category not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Category retrieved successfully",
			"data":    categories[0],
		})
	}
}

func UpdateCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		categoryID := c.Param("category_id")

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		var category models.Category
		if err := c.BindJSON(&category); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if userInfo.Role != 100 && userInfo.Branch_ID != category.Branch_ID {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unauthorized Access"})
			return
		}

		filter := bson.M{"_id": categoryID}
		update := bson.M{
			"$set": bson.M{
				"title":       category.Title,
				"description": category.Description,
				"updated_at":  time.Now(),
			},
		}

		result, err := CategoryCollection.UpdateOne(ctx, filter, update)
		if err != nil || result.MatchedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating category"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "category updated successfully"})
	}
}

func DeleteCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		categoryID := c.Param("category_id")

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		var category models.Category
		err = CategoryCollection.FindOne(ctx, bson.M{"_id": categoryID}).Decode(&category)
		if err != nil {
			log.Printf("Error retrieving category: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving category", "details": err.Error()})
			return
		}

		if userInfo.Role != 100 && userInfo.Branch_ID != category.Branch_ID {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unauthorized Access"})
			return
		}

		_, err = CategoryCollection.DeleteOne(ctx, bson.M{"_id": categoryID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting category"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "category deleted successfully"})
	}
}
