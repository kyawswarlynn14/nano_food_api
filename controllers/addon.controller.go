package controllers

import (
	"context"
	"fmt"
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

var AddOnCollection *mongo.Collection = database.AddOnCollection

func AddMenuAddOn() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Parse add-on data from request
		menuID := c.PostForm("menu_id")
		addOnTitle := c.PostForm("title")
		addOnDescription := c.PostForm("description")
		addOnPrice := helpers.ParseFloat(c.PostForm("price"))

		// Find existing menu
		var existingMenu models.Menu
		err := MenuCollection.FindOne(ctx, bson.M{"_id": menuID}).Decode(&existingMenu)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"error":   "Menu not found",
					"details": err.Error(),
				},
			)
			return
		}

		// Upload add-on image
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to initialize Firebase",
					"details": err.Error(),
				},
			)
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to connect to Firebase Storage",
					"details": err.Error(),
				},
			)
			return
		}

		var addOnImageURL string
		addOnImageFile, addOnImageHeader, err := c.Request.FormFile("cover")
		if err == nil {
			defer addOnImageFile.Close()
			addOnImageURL, err = helpers.UploadFileToFirebase(client, addOnImageFile, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnImageHeader.Filename))
			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"error":   "Failed to upload add-on image",
						"details": err.Error(),
					},
				)
				return
			}
		}

		// Create add-on object
		addOn := models.AddOn{
			AddOn_ID:    primitive.NewObjectID().Hex(),
			Menu_ID:     menuID,
			Title:       addOnTitle,
			Description: addOnDescription,
			Price:       addOnPrice,
			Cover:       addOnImageURL,
			IsAvailable: true,
		}

		_, err = AddOnCollection.InsertOne(ctx, addOn)
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false, "error": "Failed to save addon",
					"details": err.Error(),
				},
			)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Add-on added successfully",
			"add_on":  addOn,
		})
	}
}

func UpdateMenuAddOn() gin.HandlerFunc {
	return func(c *gin.Context) {
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to initialize Firebase",
					"details": err.Error(),
				},
			)
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to connect to Firebase Storage",
					"details": err.Error(),
				},
			)
			return
		}

		addOnID := c.Param("add_on_id")

		// Find existing addon
		var existingAddOn models.AddOn
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = AddOnCollection.FindOne(ctx, bson.M{"_id": addOnID}).Decode(&existingAddOn)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"error":   "Add On not found",
					"details": err.Error(),
				},
			)
			return
		}

		var addOnUpdate models.AddOn

		addOnUpdate.Menu_ID = c.PostForm("menu_id")
		addOnUpdate.Title = c.PostForm("title")
		addOnUpdate.Description = c.PostForm("description")
		addOnUpdate.Price = helpers.ParseFloat(c.PostForm("price"))
		isAvaliable := true
		if c.PostForm("is_avaliable") == "false" {
			isAvaliable = false
		}
		addOnUpdate.IsAvailable = isAvaliable

		menuExists, err := helpers.CheckDataExist(ctx, database.MenuCollection, bson.M{"_id": addOnUpdate.Menu_ID})
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to validate menu",
					"details": err.Error(),
				},
			)
			return
		}
		if !menuExists {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"error":   "Invalid menu ID",
				},
			)
			return
		}

		updateFields := bson.M{
			"updated_at":   time.Now(),
			"is_available": addOnUpdate.IsAvailable,
		}
		if addOnUpdate.Menu_ID != "" {
			updateFields["category_id"] = addOnUpdate.Menu_ID
		}
		if addOnUpdate.Title != "" {
			updateFields["title"] = addOnUpdate.Title
		}
		if addOnUpdate.Description != "" {
			updateFields["description"] = addOnUpdate.Description
		}
		if addOnUpdate.Price > 0 {
			updateFields["price"] = addOnUpdate.Price
		}

		// Handle add on cover update
		addOnCoverFile, addOnCoverHeader, err := c.Request.FormFile("cover")
		if err == nil {
			defer addOnCoverFile.Close()

			// Delete old add on cover if it exists
			if existingAddOn.Cover != "" {
				deleteErr := helpers.DeleteFileFromFirebase(client, existingAddOn.Cover)
				if deleteErr != nil {
					fmt.Println("Failed to delete old add on cover image", deleteErr)
				}
			}

			// Upload new menu cover
			addOnCoverURL, uploadErr := helpers.UploadFileToFirebase(client, addOnCoverFile, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnCoverHeader.Filename))
			if uploadErr != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"error":   "Failed to upload add on cover image",
						"details": uploadErr.Error(),
					},
				)
				return
			}
			updateFields["cover"] = addOnCoverURL
		}

		// Update add on in MongoDB
		_, err = AddOnCollection.UpdateOne(ctx, bson.M{"_id": addOnID}, bson.M{"$set": updateFields})
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false, "error": "Failed to update add on",
					"details": err.Error(),
				},
			)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Add on updated successfully",
		})
	}
}

func RemoveMenuAddOn() gin.HandlerFunc {
	return func(c *gin.Context) {
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to initialize Firebase",
					"details": err.Error(),
				},
			)
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Failed to connect to Firebase Storage",
					"details": err.Error(),
				},
			)
			return
		}

		addOnID := c.Param("add_on_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var addOn models.AddOn
		err = AddOnCollection.FindOne(ctx, bson.M{"_id": addOnID}).Decode(&addOn)
		if err != nil {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"error":   "AddOn item not found",
					"details": err.Error(),
				},
			)
			return
		}

		if addOn.Cover != "" {
			helpers.DeleteFileFromFirebase(client, addOn.Cover)
		}

		_, err = AddOnCollection.DeleteOne(ctx, bson.M{"_id": addOnID})
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Error deleting add on item",
					"details": err.Error(),
				},
			)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Add-on removed successfully",
		})
	}
}

func GetAllAddOns() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		menuID := c.Query("menu_id")

		filter := bson.M{}
		if menuID != "" {
			filter["menu_id"] = menuID
		}

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: filter}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "menus",
				"localField":   "menu_id",
				"foreignField": "_id",
				"as":           "menu",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$menu", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := AddOnCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Error retrieving addons",
					"details": err.Error(),
				},
			)
			return
		}
		defer cursor.Close(ctx)

		var addOns []bson.M
		if err := cursor.All(ctx, &addOns); err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Error decoding addOns",
					"details": err.Error(),
				},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "AddOns retrieved successfully",
				"data":    addOns,
			},
		)
	}
}

func GetOneAddOn() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		addOnID := c.Param("add_on_id")

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"_id": addOnID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "menus",
				"localField":   "menu_id",
				"foreignField": "_id",
				"as":           "menu",
			}}},
			{{Key: "$unwind", Value: bson.M{"path": "$menu", "preserveNullAndEmptyArrays": true}}},
		}

		cursor, err := AddOnCollection.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Error retrieving addons",
					"details": err.Error(),
				},
			)
			return
		}
		defer cursor.Close(ctx)

		var addOns []bson.M
		if err := cursor.All(ctx, &addOns); err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"error":   "Error decoding addons",
					"details": err.Error(),
				},
			)
			return
		}

		if len(addOns) == 0 {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"error":   "AddOn not found",
				},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "AddOn retrieved successfully",
				"data":    addOns[0],
			},
		)
	}
}
