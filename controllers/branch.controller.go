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
)

// CreateBranch: Root Admin only
func CreateBranch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var branch models.Branch
		if err := c.BindJSON(&branch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		branch.Branch_ID = primitive.NewObjectID()
		branch.Created_At = time.Now()
		branch.Updated_At = time.Now()

		_, err := BranchCollection.InsertOne(ctx, branch)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating branch"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Branch created successfully"})
	}
}

// GetBranches: Root Admin gets all, others get only their branchs
func GetBranches() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var branches []models.Branch
		cursor, err := BranchCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving branches"})
			return
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &branches); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding branches"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Branches retrieved successfully",
			"data":    branches,
		})
	}
}

func GetOneBranch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		branchID := c.Param("branch_id")
		branchObjID, err := primitive.ObjectIDFromHex(branchID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		var branch models.Branch
		err = BranchCollection.FindOne(ctx, bson.M{"_id": branchObjID}).Decode(&branch)
		if err != nil {
			log.Printf("Error retrieving branch: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving branch", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "branch retrieved successfully",
			"data":    branch,
		})
	}
}

// UpdateBranch: Root Admin and Owner can update branchs
func UpdateBranch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		branchID := c.Param("branch_id")
		branchObjID, err := primitive.ObjectIDFromHex(branchID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		userInfo, err := helpers.GetCurrentUser(c, database.UserCollection)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		var branch models.Branch
		if err := c.BindJSON(&branch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		if userInfo.Branch_ID != branchObjID {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Unauthorized Access"})
			return
		}

		filter := bson.M{"_id": branchObjID}
		update := bson.M{
			"$set": bson.M{
				"name":       branch.Name,
				"address":    branch.Address,
				"contact":    branch.Contact,
				"updated_at": time.Now(),
			},
		}

		result, err := BranchCollection.UpdateOne(ctx, filter, update)
		if err != nil || result.MatchedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating branch"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "branch updated successfully"})
	}
}

// DeleteBranch: Root Admin only
func DeleteBranch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		branchID := c.Param("branch_id")
		objID, err := primitive.ObjectIDFromHex(branchID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		_, err = BranchCollection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting branch"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Branch deleted successfully"})
	}
}