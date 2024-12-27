package controllers

import (
	"context"
	"fmt"
	"mime/multipart"
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
add_on_name: ["Cheese", "Pepperoni"] # Add-on names
add_on_note: ["note 1", "note 2"]    # Add-on notes
add_on_price: [2.0, 3.0]             # Add-on prices
add_on_images: <file[]>              # Add-on images

**/

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		app, err := helpers.InitializeFirebaseApp()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to initialize Firebase"})
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage"})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var menu models.Menu
		if err := c.ShouldBind(&menu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid input data"})
			return
		}

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"branch_id": menu.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch"})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"category_id": menu.Category_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category"})
			return
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
			return
		}

		menu.Menu_ID = primitive.NewObjectID().String()
		menu.Created_At = time.Now()
		menu.Updated_At = time.Now()

		menuCoverFile, menuCoverHeader, err := c.Request.FormFile("menu_cover")
		if err == nil {
			defer menuCoverFile.Close()
			menuCoverURL, uploadErr := helpers.UploadFileToFirebase(client, menuCoverFile, fmt.Sprintf("menu_covers/%d_%s", time.Now().Unix(), menuCoverHeader.Filename))
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu cover image"})
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
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu image"})
					return
				}
				menuImageURLs = append(menuImageURLs, imageURL)
			}
			menu.Images = menuImageURLs
		}

		// Handle add-on data and images
		addOns := []models.AddOn{}
		addOnTitles := c.PostFormArray("add_on_title")
		addOnNotes := c.PostFormArray("add_on_note")
		addOnPrices := c.PostFormArray("add_on_price")

		addOnFiles, ok := menuImages.File["add_on_images"]
		if !ok {
			addOnFiles = []*multipart.FileHeader{}
		}

		for i, addOnTitle := range addOnTitles {
			price := 0.0
			if i < len(addOnPrices) {
				price = helpers.ParseFloat(addOnPrices[i])
			}

			note := ""
			if i < len(addOnNotes) {
				note = addOnNotes[i]
			}

			var addOnImageURL string
			if i < len(addOnFiles) {
				file, err := addOnFiles[i].Open()
				if err == nil {
					defer file.Close()

					addOnImageURL, err = helpers.UploadFileToFirebase(client, file, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnFiles[i].Filename))
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload add-on image"})
						return
					}
				}
			}

			addOn := models.AddOn{
				AddOn_ID:    primitive.NewObjectID().String(),
				Title:       addOnTitle,
				Price:       price,
				Cover:       addOnImageURL,
				Note:        note,
				IsAvailable: true,
			}
			addOns = append(addOns, addOn)
		}
		menu.AddOns = addOns

		// Save menu to MongoDB
		_, err = MenuCollection.InsertOne(ctx, menu)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to save menu"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to initialize Firebase"})
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage"})
			return
		}

		menuID := c.Param("menu_id")

		// Find existing menu
		var existingMenu models.Menu
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = MenuCollection.FindOne(ctx, bson.M{"menu_id": menuID}).Decode(&existingMenu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found"})
			return
		}

		var menuUpdate models.Menu
		if err := c.ShouldBind(&menuUpdate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid input data"})
			return
		}

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"branch_id": menuUpdate.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch"})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"category_id": menuUpdate.Category_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category"})
			return
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
			return
		}

		updateFields := bson.M{
			"updated_at": time.Now(),
		}

		// Handle menu cover update
		menuCoverFile, menuCoverHeader, err := c.Request.FormFile("menu_cover")
		if err == nil {
			defer menuCoverFile.Close()

			// Delete old menu cover if it exists
			if existingMenu.Cover != "" {
				deleteErr := helpers.DeleteFileFromFirebase(client, existingMenu.Cover)
				if deleteErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete old menu cover image"})
					return
				}
			}

			// Upload new menu cover
			menuCoverURL, uploadErr := helpers.UploadFileToFirebase(client, menuCoverFile, fmt.Sprintf("menu_covers/%d_%s", time.Now().Unix(), menuCoverHeader.Filename))
			if uploadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu cover image"})
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
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete old menu image"})
					return
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
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload menu image"})
					return
				}
				menuImageURLs = append(menuImageURLs, imageURL)
			}
			updateFields["images"] = menuImageURLs
		}

		// Handle add-ons update
		addOns := []models.AddOn{}
		addOnTitles := c.PostFormArray("add_on_title")
		addOnNotes := c.PostFormArray("add_on_note")
		addOnAvailables := c.PostFormArray("add_on_available")
		addOnPrices := c.PostFormArray("add_on_price")

		addOnFiles, ok := menuImages.File["add_on_images"]
		if !ok {
			addOnFiles = []*multipart.FileHeader{}
		}

		// Delete old add-on images
		for _, oldAddOn := range existingMenu.AddOns {
			if oldAddOn.Cover != "" {
				deleteErr := helpers.DeleteFileFromFirebase(client, oldAddOn.Cover)
				if deleteErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete old add-on image"})
					return
				}
			}
		}

		// Upload new add-ons and images
		for i, addOnTitle := range addOnTitles {
			price := 0.0
			if i < len(addOnPrices) {
				price = helpers.ParseFloat(addOnPrices[i])
			}

			note := ""
			if i < len(addOnNotes) {
				note = addOnNotes[i]
			}

			avaliable := true
			if i < len(addOnAvailables) {
				avaliable, _ = strconv.ParseBool(addOnAvailables[i])
			}

			var addOnImageURL string
			if i < len(addOnFiles) {
				file, err := addOnFiles[i].Open()
				if err == nil {
					defer file.Close()

					addOnImageURL, err = helpers.UploadFileToFirebase(client, file, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnFiles[i].Filename))
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload add-on image"})
						return
					}
				}
			}

			addOn := models.AddOn{
				AddOn_ID:    primitive.NewObjectID().String(),
				Title:       addOnTitle,
				Price:       price,
				Cover:       addOnImageURL,
				Note:        note,
				IsAvailable: avaliable,
			}
			addOns = append(addOns, addOn)
		}
		updateFields["add_ons"] = addOns

		// Update menu in MongoDB
		_, err = MenuCollection.UpdateOne(ctx, bson.M{"menu_id": menuID}, bson.M{"$set": updateFields})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update menu"})
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
		err := MenuCollection.FindOneAndDelete(ctx, bson.M{"menu_id": menuID}).Decode(&menu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found"})
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
		for _, addOn := range menu.AddOns {
			if addOn.Cover != "" {
				helpers.DeleteFileFromFirebase(client, addOn.Cover)
			}
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Menu deleted successfully"})
	}
}

func GetAllMenusByBranchID() gin.HandlerFunc {
	return func(c *gin.Context) {
		branchID := c.Param("branch_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"branch_id": branchID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menus"})
			return
		}

		var menus []bson.M
		if err = cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to parse menus"})
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

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"category_id": categoryID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menus"})
			return
		}

		var menus []bson.M
		if err = cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to parse menus"})
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
			bson.M{"$match": bson.M{"menu_id": menuID}},
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menu"})
			return
		}

		var menu []bson.M
		if err = cursor.All(ctx, &menu); err != nil || len(menu) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found"})
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
				filter["price"] = bson.M{"$gte": minPriceVal} // Filter for prices greater than or equal to minPrice
			}
		}

		if maxPrice != "" {
			if maxPriceVal, err := strconv.ParseFloat(maxPrice, 64); err == nil {
				if filter["price"] == nil {
					filter["price"] = bson.M{"$lte": maxPriceVal} // Filter for prices less than or equal to maxPrice
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to retrieve menus"})
			return
		}

		// Collect the results
		var menus []models.Menu
		if err := cursor.All(ctx, &menus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to process search results"})
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
