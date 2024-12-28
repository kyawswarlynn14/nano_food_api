package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	database "nano_food_api/database"
	helpers "nano_food_api/helpers"
	models "nano_food_api/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var MenuCollection *mongo.Collection = database.MenuCollection

/**

title: "Pizza"
short_title: "Pizza"
description: "Delicious pizza with toppings"
price: 10.5
category_id: "category_id_here"
branch_id: "branch_id_here"
menu_cover: <file>         # Single menu cover image
menu_images: <file[]>      # Multiple menu images

**/

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to initialize Firebase", "details": err.Error()})
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage", "details": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var menu models.Menu

		menu.Category_ID = c.PostForm("category_id")
		menu.Branch_ID = c.PostForm("branch_id")
		menu.Title = c.PostForm("title")
		menu.Short_Title = c.PostForm("short_title")
		menu.Description = c.PostForm("description")
		menu.Price = helpers.ParseFloat(c.PostForm("price"))
		menu.Discount = helpers.ParseFloat(c.PostForm("discount"))

		if menu.Category_ID == "" || menu.Branch_ID == "" || menu.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing required fields"})
			return
		}

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"_id": menu.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch", "details": err.Error()})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"_id": menu.Category_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category", "details": err.Error()})
			return
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
			return
		}

		menu.Menu_ID = primitive.NewObjectID().Hex()
		menu.Created_At = time.Now()
		menu.Updated_At = time.Now()
		menu.IsAvailable = true

		menuCoverFile, menuCoverHeader, err := c.Request.FormFile("menu_cover")
		if err == nil {
			defer menuCoverFile.Close()
			menuCoverURL, uploadErr := helpers.UploadFileToFirebase(client, menuCoverFile, fmt.Sprintf("menu_covers/%d_%s", time.Now().Unix(), menuCoverHeader.Filename))
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu cover image", "details": uploadErr.Error()})
				return
			}
			menu.Cover = menuCoverURL
		}

		// Handle multiple menu images
		menuImages, _ := c.MultipartForm()
		if menuImages != nil {
			var menuImageURLs []string
			files := menuImages.File["menu_images"]
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					continue
				}
				defer file.Close()

				imageURL, uploadErr := helpers.UploadFileToFirebase(client, file, fmt.Sprintf("menu_images/%d_%s", time.Now().Unix(), fileHeader.Filename))
				if uploadErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu image", "details": uploadErr.Error()})
					return
				}
				menuImageURLs = append(menuImageURLs, imageURL)
			}
			menu.Images = menuImageURLs
		}

		// Save menu to MongoDB
		_, err = MenuCollection.InsertOne(ctx, menu)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to save menu", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Menu created successfully",
			"menu":    menu,
		})
	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to initialize Firebase", "details": err.Error()})
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage", "details": err.Error()})
			return
		}

		menuID := c.Param("menu_id")

		// Find existing menu
		var existingMenu models.Menu
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = MenuCollection.FindOne(ctx, bson.M{"_id": menuID}).Decode(&existingMenu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found", "details": err.Error()})
			return
		}

		var menuUpdate models.Menu

		menuUpdate.Category_ID = c.PostForm("category_id")
		menuUpdate.Branch_ID = c.PostForm("branch_id")
		menuUpdate.Title = c.PostForm("title")
		menuUpdate.Short_Title = c.PostForm("short_title")
		menuUpdate.Description = c.PostForm("description")
		menuUpdate.Price = helpers.ParseFloat(c.PostForm("price"))
		menuUpdate.Discount = helpers.ParseFloat(c.PostForm("discount"))
		isAvaliable := true
		if c.PostForm("is_avaliable") == "false" {
			isAvaliable = false
		}
		menuUpdate.IsAvailable = isAvaliable

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"_id": menuUpdate.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch", "details": err.Error()})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"_id": menuUpdate.Category_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category", "details": err.Error()})
			return
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
			return
		}

		updateFields := bson.M{
			"updated_at":   time.Now(),
			"is_available": menuUpdate.IsAvailable,
		}
		if menuUpdate.Category_ID != "" {
			updateFields["category_id"] = menuUpdate.Category_ID
		}
		if menuUpdate.Title != "" {
			updateFields["title"] = menuUpdate.Title
		}
		if menuUpdate.Short_Title != "" {
			updateFields["short_title"] = menuUpdate.Short_Title
		}
		if menuUpdate.Description != "" {
			updateFields["description"] = menuUpdate.Description
		}
		if menuUpdate.Price > 0 {
			updateFields["price"] = menuUpdate.Price
		}
		if menuUpdate.Discount > 0 {
			updateFields["discount"] = menuUpdate.Discount
		}

		// Handle menu cover update
		menuCoverFile, menuCoverHeader, err := c.Request.FormFile("menu_cover")
		if err == nil {
			defer menuCoverFile.Close()

			// Delete old menu cover if it exists
			if existingMenu.Cover != "" {
				deleteErr := helpers.DeleteFileFromFirebase(client, existingMenu.Cover)
				if deleteErr != nil {
					fmt.Println("Failed to delete old menu cover image", deleteErr)
				}
			}

			// Upload new menu cover
			menuCoverURL, uploadErr := helpers.UploadFileToFirebase(client, menuCoverFile, fmt.Sprintf("menu_covers/%d_%s", time.Now().Unix(), menuCoverHeader.Filename))
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu cover image", "details": uploadErr.Error()})
				return
			}
			updateFields["cover"] = menuCoverURL
		}

		// Handle menu images update
		menuImages, _ := c.MultipartForm()
		if menuImages != nil {
			var menuImageURLs []string
			files := menuImages.File["menu_images"]

			// Delete old menu images
			for _, oldImage := range existingMenu.Images {
				deleteErr := helpers.DeleteFileFromFirebase(client, oldImage)
				if deleteErr != nil {
					fmt.Println("Failed to delete old menu images", deleteErr)
				}
			}

			// Upload new menu images
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					continue
				}
				defer file.Close()

				imageURL, uploadErr := helpers.UploadFileToFirebase(client, file, fmt.Sprintf("menu_images/%d_%s", time.Now().Unix(), fileHeader.Filename))
				if uploadErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu image", "details": uploadErr.Error()})
					return
				}
				menuImageURLs = append(menuImageURLs, imageURL)
			}
			updateFields["images"] = menuImageURLs
		}

		// Update menu in MongoDB
		_, err = MenuCollection.UpdateOne(ctx, bson.M{"_id": menuID}, bson.M{"$set": updateFields})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update menu", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Menu updated successfully",
		})
	}
}

func DeleteMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		menuID := c.Param("menu_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Find and delete the menu
		var menu models.Menu
		err := MenuCollection.FindOneAndDelete(ctx, bson.M{"_id": menuID}).Decode(&menu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found", "details": err.Error()})
			return
		}

		// Update add-ons to set menu_id to an empty string
		_, err = AddOnCollection.UpdateMany(ctx, bson.M{"menu_id": menuID}, bson.M{"$set": bson.M{"menu_id": ""}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update add-ons", "details": err.Error()})
			return
		}

		// Optionally delete associated images (menu cover, menu images, add-on images)
		app, _ := helpers.InitializeFirebaseApp()
		client, _ := app.Storage(context.Background())
		if menu.Cover != "" {
			helpers.DeleteFileFromFirebase(client, menu.Cover)
		}
		for _, image := range menu.Images {
			helpers.DeleteFileFromFirebase(client, image)
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Menu deleted successfully"})
	}
}

func GetAllMenusByBranchID() gin.HandlerFunc {
	return func(c *gin.Context) {
		branchID := c.Param("branch_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"_id": branchID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch", "details": err.Error()})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		cursor, err := MenuCollection.Aggregate(ctx, bson.A{
			bson.M{"$match": bson.M{"branch_id": branchID}},
			bson.M{
				"$lookup": bson.M{
					"from":         "branches",
					"localField":   "branch_id",
					"foreignField": "branch_id",
					"as":           "branch",
				},
			},
			bson.M{
				"$lookup": bson.M{
					"from":         "categories",
					"localField":   "category_id",
					"foreignField": "category_id",
					"as":           "category",
				},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menus", "details": err.Error()})
			return
		}

		var menus []bson.M
		if err = cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to parse menus", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "menus": menus})
	}
}

func GetAllMenusByCategoryID() gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Param("category_id")
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"_id": categoryID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category", "details": err.Error()})
			return
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
			return
		}

		cursor, err := MenuCollection.Aggregate(ctx, bson.A{
			bson.M{"$match": bson.M{"category_id": categoryID}},
			bson.M{
				"$lookup": bson.M{
					"from":         "branches",
					"localField":   "branch_id",
					"foreignField": "branch_id",
					"as":           "branch",
				},
			},
			bson.M{
				"$lookup": bson.M{
					"from":         "categories",
					"localField":   "category_id",
					"foreignField": "category_id",
					"as":           "category",
				},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menus", "details": err.Error()})
			return
		}

		var menus []bson.M
		if err = cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to parse menus", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "menus": menus})
	}
}

func GetOneMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		menuID := c.Param("menu_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Aggregate to populate branch and category
		cursor, err := MenuCollection.Aggregate(ctx, bson.A{
			bson.M{"$match": bson.M{"_id": menuID}},
			bson.M{
				"$lookup": bson.M{
					"from":         "branches",
					"localField":   "branch_id",
					"foreignField": "branch_id",
					"as":           "branch",
				},
			},
			bson.M{
				"$lookup": bson.M{
					"from":         "categories",
					"localField":   "category_id",
					"foreignField": "category_id",
					"as":           "category",
				},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menu", "details": err.Error()})
			return
		}

		var menu []bson.M
		if err = cursor.All(ctx, &menu); err != nil || len(menu) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "menu": menu[0]})
	}
}

func SearchMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse query parameters for search
		searchQuery := c.DefaultQuery("query", "")
		categoryID := c.DefaultQuery("category_id", "")
		minPrice := c.DefaultQuery("min_price", "")
		maxPrice := c.DefaultQuery("max_price", "")

		filter := bson.M{}

		// Add search query filter (title or description search)
		if searchQuery != "" {
			filter["$or"] = []bson.M{
				{"title": bson.M{"$regex": searchQuery, "$options": "i"}},
				{"description": bson.M{"$regex": searchQuery, "$options": "i"}},
			}
		}

		// Add category filter if provided
		if categoryID != "" {
			filter["category_id"] = categoryID
		}

		// Add price range filter if provided
		if minPrice != "" {
			minPriceVal, err := strconv.ParseFloat(minPrice, 64)
			if err == nil {
				filter["price"] = bson.M{"$gte": minPriceVal}
			}
		}

		if maxPrice != "" {
			if maxPriceVal, err := strconv.ParseFloat(maxPrice, 64); err == nil {
				if filter["price"] == nil {
					filter["price"] = bson.M{"$lte": maxPriceVal}
				} else {
					filter["price"] = bson.M{
						"$gte": filter["price"].(bson.M)["$gte"],
						"$lte": maxPriceVal,
					}
				}
			}
		}

		// Set up the context with timeout
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Aggregate to populate branch and category
		cursor, err := MenuCollection.Aggregate(ctx, bson.A{
			bson.M{"$match": filter},
			bson.M{
				"$lookup": bson.M{
					"from":         "branches",
					"localField":   "branch_id",
					"foreignField": "branch_id",
					"as":           "branch",
				},
			},
			bson.M{
				"$lookup": bson.M{
					"from":         "categories",
					"localField":   "category_id",
					"foreignField": "category_id",
					"as":           "category",
				},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menus", "details": err.Error()})
			return
		}

		// Collect the results
		var menus []models.Menu
		if err := cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to process search results", "details": err.Error()})
			return
		}

		// Return the search results
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Menus found successfully",
			"menus":   menus,
		})
	}
}
