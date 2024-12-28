package controllers

import (
	"context"
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

var SaleCollection *mongo.Collection = database.SaleCollection

func CreateSale() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var sale models.Sale
		if err := c.BindJSON(&sale); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate Branch ID
		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"_id": sale.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch", "details": err.Error()})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		// Validate Table ID
		tableExists, err := helpers.CheckDataExist(ctx, database.TableCollection, bson.M{"_id": sale.Table_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate table", "details": err.Error()})
			return
		}
		if !tableExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid table ID"})
			return
		}

		// Update order statuses and calculate total amounts
		totalAmount := 0.0
		for _, orderID := range sale.OrderIDs {
			filter := bson.M{"_id": orderID}
			update := bson.M{
				"$set": bson.M{
					"status":  "003", // Completed
					"is_paid": true,
				},
			}
			order := models.Order{}
			err := OrderCollection.FindOneAndUpdate(ctx, filter, update).Decode(&order)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update order status", "details": err.Error()})
				return
			}
			totalAmount += order.TotalAmount
		}

		// Assign IDs and calculate GrandTotal
		sale.Sale_ID = primitive.NewObjectID().Hex()
		sale.TotalAmount = totalAmount
		sale.GrandTotal = totalAmount - sale.Discount + sale.Tax
		sale.Created_At = time.Now()

		result, err := SaleCollection.InsertOne(ctx, sale)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error creating sale", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Sale created successfully", "data": result})
	}
}

func DeleteSale() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		saleID := c.Param("sale_id")

		_, err := SaleCollection.DeleteOne(ctx, bson.M{"_id": saleID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting sale", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Sale deleted successfully"})
	}
}

func GetAllSales() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		branchID := c.Query("branch_id")
		tableID := c.Query("table_id")

		filter := bson.M{}
		if branchID != "" {
			filter["branch_id"] = branchID
		}
		if tableID != "" {
			filter["table_id"] = tableID
		}

		// MongoDB aggregation pipeline
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: filter}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "tables",
				"localField":   "table_id",
				"foreignField": "_id",
				"as":           "table_details",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$table_details", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := SaleCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving sales", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var sales []bson.M
		if err := cursor.All(ctx, &sales); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding sales", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Sales retrieved successfully", "data": sales})
	}
}

func GetOneSale() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		saleID := c.Param("sale_id")

		// MongoDB aggregation pipeline
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"_id": saleID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "tables",
				"localField":   "table_id",
				"foreignField": "_id",
				"as":           "table_details",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$table_details", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := SaleCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving sale", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var sales []bson.M
		if err := cursor.All(ctx, &sales); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding sale", "details": err.Error()})
			return
		}

		if len(sales) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Sale not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Sale retrieved successfully", "data": sales[0]})
	}
}
