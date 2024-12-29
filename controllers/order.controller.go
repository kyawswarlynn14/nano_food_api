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

var OrderCollection *mongo.Collection = database.OrderCollection

func orderPipeline(filter bson.M) mongo.Pipeline {
	var pipeline = mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "menus",
			"localField":   "menu_items.menu_id",
			"foreignField": "_id",
			"as":           "menu_details",
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "add_ons",
			"localField":   "menu_items.add_on_items.add_on_id",
			"foreignField": "_id",
			"as":           "add_on_details",
		}}},
		{{Key: "$addFields", Value: bson.M{
			"menu_items": bson.M{
				"$map": bson.M{
					"input": "$menu_items",
					"as":    "menu_item",
					"in": bson.M{
						"menu_id":  "$$menu_item.menu_id",
						"note":     "$$menu_item.note",
						"quantity": "$$menu_item.quantity",
						"menu_details": bson.M{
							"$filter": bson.M{
								"input": "$menu_details",
								"as":    "menu_detail",
								"cond":  bson.M{"$eq": []interface{}{"$$menu_detail._id", "$$menu_item.menu_id"}},
							},
						},
						"add_on_items": bson.M{
							"$map": bson.M{
								"input": "$$menu_item.add_on_items",
								"as":    "add_on_item",
								"in": bson.M{
									"add_on_id": "$$add_on_item.add_on_id",
									"note":      "$$add_on_item.note",
									"quantity":  "$$add_on_item.quantity",
									"add_on_details": bson.M{
										"$filter": bson.M{
											"input": "$add_on_details",
											"as":    "add_on_detail",
											"cond":  bson.M{"$eq": []interface{}{"$$add_on_detail._id", "$$add_on_item.add_on_id"}},
										},
									},
								},
							},
						},
					},
				},
			},
		}}},
		{{Key: "$project", Value: bson.M{
			"menu_details":   0,
			"add_on_details": 0,
		}}},
	}

	return pipeline
}

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
		for _, menuItem := range order.MenuItems {
			var menu models.Menu
			err := MenuCollection.FindOne(ctx, bson.M{"_id": menuItem.Menu_ID}).Decode(&menu)
			if err != nil {
				log.Printf("Error retrieving menu: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving menu", "details": err.Error()})
				return
			}
			menuPrice := (menu.Price - menu.Discount) * float64(menuItem.Quantity)

			addOnSubTotal := 0.0
			for _, addOnItem := range menuItem.AddOnItems {
				var addOn models.AddOn
				err := AddOnCollection.FindOne(ctx, bson.M{"_id": addOnItem.AddOnID}).Decode(&addOn)
				if err != nil {
					log.Printf("Error retrieving addon: %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Error retrieving addon", "details": err.Error()})
					return
				}
				addOnPrice := addOn.Price * float64(addOnItem.Quantity)
				addOnSubTotal += addOnPrice
			}

			menuSubtotal := menuPrice + addOnSubTotal
			totalAmount += menuSubtotal
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

		pipeline := orderPipeline(filter)

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
		filter := bson.M{"_id": orderID}

		pipeline := orderPipeline(filter)

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
