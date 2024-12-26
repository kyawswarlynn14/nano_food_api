package routes

import (
	controllers "nano_food_api/controllers"
	middlewares "nano_food_api/middlewares"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	authRoutes := r.Group("/").Use(middlewares.Authentication())
	adminRoutes := r.Group("/").Use(middlewares.Authentication()).Use(middlewares.Authorization([]int{2, 3, 100}))

	publicRoutes.POST("/register", controllers.RegisterUser())
	publicRoutes.POST("/verify", controllers.VerifyUser())
	publicRoutes.POST("/login", controllers.LoginUser())

	authRoutes.GET("/me", controllers.GetCurrentUser())
	authRoutes.PUT("/update-user-info", controllers.UpdateUserInfo())
	authRoutes.PUT("/update-user-password", controllers.UpdateUserPassword())
	authRoutes.PUT("/upload-avatar", controllers.UploadAvatar())

	adminRoutes.GET("/get-restaurant-users/:restaurant_id", controllers.GetAllRestaurantUsers())
	adminRoutes.PUT("/update-user-role", controllers.UpdateUserRole())
	adminRoutes.DELETE("/delete-user", controllers.DeleteUser())

	authRoutes.Use(middlewares.Authorization([]int{3, 100})).POST("/create-user", controllers.CreateUser())
	authRoutes.Use(middlewares.Authorization([]int{100})).POST("/get-all-users", controllers.GetAllUsers())
}

func RestaurantRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	adminRoutes := r.Group("/").Use(middlewares.Authentication()).Use(middlewares.Authorization([]int{3, 100}))
	rootRoutes := r.Group("/").Use(middlewares.Authentication()).Use(middlewares.Authorization([]int{100}))

	publicRoutes.POST("/get-restaurant/:restaurant_id", controllers.GetOneRestaurant())
	adminRoutes.PUT("/update-restaurant/:restaurant_id", controllers.UpdateRestaurant())
	adminRoutes.PUT("/add-restaurant-member", controllers.AddRestaurantMember())
	adminRoutes.PUT("/remove-restaurant-member", controllers.RemoveRestaurantMember())

	rootRoutes.POST("/create-restaurant", controllers.CreateRestaurant())
	rootRoutes.GET("/get-restaurants", controllers.GetRestaurants())
	rootRoutes.DELETE("/delete-restaurant/:restaurant_id", controllers.DeleteRestaurant())
}
