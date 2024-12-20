package main

import (
	"log"
	"os"

	"github.com/kyawswarlynn14/nano_food_api/middlewares"
	"github.com/kyawswarlynn14/nano_food_api/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	router := gin.New()
	router.Use(gin.Logger())

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"https://nano-food.vercel.app",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	publicRoutes := router.Group("/")
	authRoutes := router.Group("/")
	authRoutes.Use(middlewares.Authentication())

	adminRoutes := router.Group("/")
	adminRoutes.Use(middlewares.Authentication()).Use(middlewares.Authorization([]int{1, 2, 100}))

	routes.UserRoutes(publicRoutes, authRoutes, adminRoutes)

	log.Fatal(router.Run(":" + port))
}
