package routes

import (
	controllers "github.com/kyawswarlynn14/nano_food_api/controllers"

	"github.com/gin-gonic/gin"
)

func UserRoutes(publicRoutes, authRoutes *gin.RouterGroup, adminRoutes *gin.RouterGroup) {
	publicRoutes.POST("/expense/register", controllers.RegisterUser())
	publicRoutes.POST("/expense/login", controllers.LoginUser())

	authRoutes.GET("/me", controllers.GetCurrentUser())
	authRoutes.PUT("/update-user-info", controllers.UpdateUserInfo())
	authRoutes.PUT("/update-user-password", controllers.UpdateUserPassword())

	adminRoutes.POST("/expense/login", controllers.LoginUser())
	adminRoutes.GET("/get-all-users", controllers.GetAllUsers())
	adminRoutes.PUT("/update-user-role", controllers.UpdateUserRole())
	adminRoutes.DELETE("/delete-user/:id", controllers.DeleteUser())
}
