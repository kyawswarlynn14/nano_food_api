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

	adminRoutes.GET("/get-branch-users/:branch_id", controllers.GetAllBranchUsers())
	adminRoutes.PUT("/update-user-role", controllers.UpdateUserRole())
	adminRoutes.DELETE("/delete-user", controllers.DeleteUser())

	authRoutes.Use(middlewares.Authorization([]int{3, 100})).POST("/create-user", controllers.CreateUser())
	authRoutes.Use(middlewares.Authorization([]int{3, 100})).POST("/update-user-branch", controllers.UpdateUserBranch())
	authRoutes.Use(middlewares.Authorization([]int{100})).POST("/get-all-users", controllers.GetAllUsers())
}

func BranchRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	adminRoutes := r.Group("/").Use(middlewares.Authentication()).Use(middlewares.Authorization([]int{3, 100}))
	rootRoutes := r.Group("/").Use(middlewares.Authentication()).Use(middlewares.Authorization([]int{100}))

	publicRoutes.POST("/get-branch/:branch_id", controllers.GetOneBranch())
	adminRoutes.PUT("/update-branch/:branch_id", controllers.UpdateBranch())
	adminRoutes.POST("/create-branch", controllers.CreateBranch())
	adminRoutes.GET("/get-branches", controllers.GetBranches())
	rootRoutes.DELETE("/delete-branch/:branch_id", controllers.DeleteBranch())
}
