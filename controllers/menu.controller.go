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
add_on_title: ["Cheese", "Pepperoni"] # Add-on titles
add_on_description: ["note 1", "note 2"]    # Add-on descriptions
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

		menu.Category_ID = c.PostForm("category_id")
		menu.Branch_ID = c.PostForm("branch_id")
		menu.Title = c.PostForm("title")
		menu.Short_Title = c.PostForm("short_title")
		menu.Description = c.PostForm("description")
		menu.Price = helpers.ParseFloat(c.PostForm("price"))

		if menu.Category_ID == "" || menu.Branch_ID == "" || menu.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing required fields"})
			return
		}

		branchExists, err := helpers.CheckDataExist(ctx, database.BranchCollection, bson.M{"_id": menu.Branch_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate branch"})
			return
		}
		if !branchExists {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid branch ID"})
			return
		}

		categoryExists, err := helpers.CheckDataExist(ctx, database.CategoryCollection, bson.M{"_id": menu.Category_ID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to validate category"})
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

		// Handle add-on data and images
		addOns := []models.AddOn{}
		addOnTitles := c.PostFormArray("add_on_title")
		addOnDescriptions := c.PostFormArray("add_on_description")
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

			description := ""
			if i < len(addOnDescriptions) {
				description = addOnDescriptions[i]
			}

			var addOnImageURL string
			if i < len(addOnFiles) {
				file, err := addOnFiles[i].Open()
				if err == nil {
					defer file.Close()

					addOnImageURL, err = helpers.UploadFileToFirebase(client, file, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnFiles[i].Filename))
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload add-on image", "details": err.Error()})
						return
					}
				}
			}

			addOn := models.AddOn{
				AddOn_ID:    primitive.NewObjectID().Hex(),
				Title:       addOnTitle,
				Price:       price,
				Cover:       addOnImageURL,
				Description: description,
				IsAvailable: true,
			}
			addOns = append(addOns, addOn)
		}
		menu.AddOns = addOns

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

func AddMenuAddOn() gin.HandlerFunc {
	return func(c *gin.Context) {
		menuID := c.Param("menu_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Find existing menu
		var existingMenu models.Menu
		err := MenuCollection.FindOne(ctx, bson.M{"_id": menuID}).Decode(&existingMenu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found", "details": err.Error()})
			return
		}

		// Parse add-on data from request
		addOnTitle := c.PostForm("add_on_title")
		addOnDescription := c.PostForm("add_on_description")
		addOnPrice := helpers.ParseFloat(c.PostForm("add_on_price"))

		// Upload add-on image
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

		var addOnImageURL string
		addOnImageFile, addOnImageHeader, err := c.Request.FormFile("add_on_image")
		if err == nil {
			defer addOnImageFile.Close()
			addOnImageURL, err = helpers.UploadFileToFirebase(client, addOnImageFile, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnImageHeader.Filename))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload add-on image", "details": err.Error()})
				return
			}
		}

		// Create add-on object
		addOn := models.AddOn{
			AddOn_ID:    primitive.NewObjectID().Hex(),
			Title:       addOnTitle,
			Description: addOnDescription,
			Price:       addOnPrice,
			Cover:       addOnImageURL,
			IsAvailable: true,
		}

		// Update menu with the new add-on
		_, err = MenuCollection.UpdateOne(ctx, bson.M{"_id": menuID}, bson.M{"$push": bson.M{"add_ons": addOn}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to add add-on", "details": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to initialize Firebase"})
			return
		}

		client, err := app.Storage(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to connect to Firebase Storage"})
			return
		}

		menuID := c.Param("menu_id")
		addOnID := c.Param("add_on_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Find the menu containing the add-on
		var menu models.Menu
		err = MenuCollection.FindOne(ctx, bson.M{"_id": menuID}).Decode(&menu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found"})
			return
		}

		// Parse the updated add-on data from the request body
		var updatedAddOn models.AddOn
		updatedAddOn.Title = c.PostForm("add_on_title")
		updatedAddOn.Description = c.PostForm("add_on_description")
		updatedAddOn.Price = helpers.ParseFloat(c.PostForm("add_on_price"))
		isAvailable := true
		if c.PostForm("is_available") == "false" {
			isAvailable = false
		}
		updatedAddOn.IsAvailable = isAvailable

		// Locate the add-on to update
		var found bool
		for i, addOn := range menu.AddOns {
			if addOn.AddOn_ID == addOnID {
				addOnFile, addOnHeader, err := c.Request.FormFile("add_on_image")
				if err == nil {
					defer addOnFile.Close()

					if addOn.Cover != "" {
						deleteErr := helpers.DeleteFileFromFirebase(client, addOn.Cover)
						if deleteErr != nil {
							fmt.Println("Failed to delete old add on image", deleteErr)
						}
					}

					// Upload new add on cover
					addOnImageURL, uploadErr := helpers.UploadFileToFirebase(client, addOnFile, fmt.Sprintf("addons/%d_%s", time.Now().Unix(), addOnHeader.Filename))
					if uploadErr != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to upload add on image", "details": uploadErr.Error()})
						return
					}
					addOn.Cover = addOnImageURL
				}

				// Update other add-on fields
				addOn.Title = updatedAddOn.Title
				addOn.Price = updatedAddOn.Price
				addOn.Description = updatedAddOn.Description
				addOn.IsAvailable = updatedAddOn.IsAvailable
				menu.AddOns[i] = addOn
				found = true
				break
			}
		}

		if !found {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Add-on not found"})
			return
		}

		// Save the updated menu back to MongoDB
		_, err = MenuCollection.UpdateOne(ctx, bson.M{"_id": menuID}, bson.M{"$set": bson.M{"add_ons": menu.AddOns}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update menu", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Add-on updated successfully",
		})
	}
}

func RemoveMenuAddOn() gin.HandlerFunc {
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
		addOnID := c.Param("add_on_id")

		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var menu models.Menu
		err = MenuCollection.FindOne(ctx, bson.M{"_id": menuID}).Decode(&menu)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Menu not found"})
			return
		}

		var updatedAddOns []models.AddOn
		var addOnToDelete *models.AddOn
		for _, addOn := range menu.AddOns {
			if addOn.AddOn_ID == addOnID {
				addOnToDelete = &addOn
			} else {
				updatedAddOns = append(updatedAddOns, addOn)
			}
		}

		if addOnToDelete == nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Add-on not found"})
			return
		}

		if addOnToDelete.Cover != "" {
			deleteErr := helpers.DeleteFileFromFirebase(client, addOnToDelete.Cover)
			if deleteErr != nil {
				fmt.Println("Failed to delete add-on image from Firebase:", deleteErr)
			}
		}

		_, err = MenuCollection.UpdateOne(ctx, bson.M{"_id": menuID}, bson.M{"$set": bson.M{"add_ons": updatedAddOns}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update menu", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Add-on removed successfully",
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
