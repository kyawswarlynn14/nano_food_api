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

var OrderCollection *mongo.Collection = database.OrderCollection

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var order models.Order
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"_id": order.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch", "details": err.Error()})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		tableExists, err := helpers.CheckDataExist(ctx, database.TableCollection, bson.M{"_id": order.Table_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate table", "details": err.Error()})
			return
		}
		if !tableExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid table ID"})
			return
		}

		order.Order_ID = primitive.NewObjectID().Hex()
		order.IsPaid = false
		order.Status = "001"
		order.Created_At = time.Now()
		order.Updated_At = time.Now()

		// Calculate total amount
		totalAmount := 0.0
		for _, item := range order.MenuItems {
			itemTotal := item.SubTotal
			totalAmount += itemTotal
		}
		order.TotalAmount = totalAmount

		result, err := OrderCollection.InsertOne(ctx, order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Order created successfully", "data": result})
	}
}

func GetAllOrders() gin.HandlerFunc {
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

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: filter}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "menus",
				"localField":   "menu_items.menu_id",
				"foreignField": "_id",
				"as":           "menu_details",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$menu_details", "preserveNullAndEmptyArrays": true}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "addons",
				"localField":   "menu_items.add_on_items.add_on_id",
				"foreignField": "_id",
				"as":           "add_on_details",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$add_on_details", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := OrderCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving orders", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var orders []bson.M
		if err := cursor.All(ctx, &orders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding orders", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Orders retrieved successfully", "data": orders})
	}
}

func GetOneOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderID := c.Param("order_id")

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"_id": orderID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "menus",
				"localField":   "menu_items.menu_id",
				"foreignField": "_id",
				"as":           "menu_details",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$menu_details", "preserveNullAndEmptyArrays": true}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "addons",
				"localField":   "menu_items.add_on_items.add_on_id",
				"foreignField": "_id",
				"as":           "add_on_details",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$add_on_details", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := OrderCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving order", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var orders []bson.M
		if err := cursor.All(ctx, &orders); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error decoding order", "details": err.Error()})
			return
		}

		if len(orders) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Order not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Order retrieved successfully", "data": orders[0]})
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderID := c.Param("order_id")

		var order models.Order
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}

		filter := bson.M{"_id": orderID}
		update := bson.M{
			"$set": bson.M{
				"status":     order.Status,
				"note":       order.Note,
				"is_paid":    order.IsPaid,
				"updated_at": time.Now(),
			},
		}

		result, err := OrderCollection.UpdateOne(ctx, filter, update)
		if err != nil || result.MatchedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error updating order", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Order updated successfully"})
	}
}

func DeleteOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderID := c.Param("order_id")

		_, err := OrderCollection.DeleteOne(ctx, bson.M{"_id": orderID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error deleting order", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Order deleted successfully"})
	}
}
