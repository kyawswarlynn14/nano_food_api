package main

import (
	"log"
	"os"
	"time"

	"nano_food_api/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/limiter"
	"github.com/gin-contrib/limiter/store/memory"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Rate limit configuration
	rateStore := memory.NewStore()
	rateLimiter := limiter.RateLimiter(limiter.NewRateFromDuration(50, time.Minute), rateStore)

	router := gin.New()
	router.Use(gin.Logger())

	// CORS middleware
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

	router.Use(rateLimiter)

	// Routes
	routes.UserRoutes(router)
	routes.BranchRoutes(router)

	log.Fatal(router.Run(":" + port))
}
