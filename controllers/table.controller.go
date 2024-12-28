package controllers

import (
	"context"
	"log"
	"nano_food_api/database"
	"nano_food_api/helpers"
	"nano_food_api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var TableCollection *mongo.Collection = database.TableCollection

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var table models.Table
		if err := c.BindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var branch models.Branch
		err := BranchCollection.FindOne(ctx, bson.M{"_id": table.Branch_ID}).Decode(&branch)
		if err != nil {
			log.Printf("Error retrieving branch: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving branch", "details": err.Error()})
			return
		}

		table.Table_ID = primitive.NewObjectID().Hex()
		table.Created_At = time.Now()
		table.Updated_At = time.Now()

		result, err := TableCollection.InsertOne(ctx, table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": true, "error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Table created successfully", "data": result})
	}
}

func GetAllTables() gin.HandlerFunc {
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

		cursor, err := TableCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Error retrieving tables,",
				"details": err.Error(),
			})
			return
		}
		defer cursor.Close(ctx)

		var tables []bson.M
		if err := cursor.All(ctx, &tables); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding tables", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Tables retrieved successfully",
			"data":    tables,
		})
	}
}

func GetOneTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		tableID := c.Param("table_id")

		// MongoDB Aggregation Pipeline
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"_id": tableID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "branches",
				"localField":   "branch_id",
				"foreignField": "_id",
				"as":           "branch",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$branch", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := TableCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving table", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var tables []bson.M
		if err := cursor.All(ctx, &tables); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding table", "details": err.Error()})
			return
		}

		if len(tables) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Table not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Table retrieved successfully",
			"data":    tables[0],
		})
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		tableID := c.Param("table_id")

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		var table models.Table
		if err := c.BindJSON(&table); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if userInfo.Role != 100 && userInfo.Branch_ID != table.Branch_ID {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unauthorized Access"})
			return
		}

		filter := bson.M{"_id": tableID}
		update := bson.M{
			"$set": bson.M{
				"name":        table.Name,
				"seats":       table.Seats,
				"status":      table.Status,
				"is_reserved": table.IsReserved,
				"updated_at":  time.Now(),
			},
		}

		result, err := TableCollection.UpdateOne(ctx, filter, update)
		if err != nil || result.MatchedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating table", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Table updated successfully"})
	}
}

func DeleteTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		tableID := c.Param("table_id")

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		var table models.Table
		err = TableCollection.FindOne(ctx, bson.M{"_id": tableID}).Decode(&table)
		if err != nil {
			log.Printf("Error retrieving table: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving table", "details": err.Error()})
			return
		}

		if userInfo.Role != 100 && userInfo.Branch_ID != table.Branch_ID {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unauthorized Access"})
			return
		}

		_, err = TableCollection.DeleteOne(ctx, bson.M{"_id": tableID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting table", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Table deleted successfully"})
	}
}
