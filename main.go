package main

import (
	"log"
	"os"
	"time"

	"nano_food_api/middlewares"
	"nano_food_api/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
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

	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  20,
	}
	store := memory.NewStore()

	instance := limiter.New(store, rate)
	rateLimiter := ginlimiter.NewMiddleware(instance)

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(rateLimiter)

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

	routeGroups := &routes.RouteGroups{
		Public:    router.Group("/"),
		Auth:      router.Group("/").Use(middlewares.Authentication([]int{})),
		Assistant: router.Group("/").Use(middlewares.Authentication([]int{1, 2, 3, 100})),
		Manager:   router.Group("/").Use(middlewares.Authentication([]int{2, 3, 100})),
		Admin:     router.Group("/").Use(middlewares.Authentication([]int{3, 100})),
		Root:      router.Group("/").Use(middlewares.Authentication([]int{100})),
	}

	// Routes
	routes.UserRoutes(routeGroups)
	routes.BranchRoutes(routeGroups)
	routes.CategoryRoutes(routeGroups)
	routes.TableRoutes(routeGroups)
	routes.MenuRoutes(routeGroups)
	routes.AddOnRoutes(routeGroups)
	routes.OrderRoutes(routeGroups)
	routes.SaleRoutes(routeGroups)

	log.Fatal(router.Run(":" + port))
}
