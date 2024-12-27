package routes

import (
	controllers "nano_food_api/controllers"
	middlewares "nano_food_api/middlewares"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	authRoutes := r.Group("/").Use(middlewares.Authentication([]int{}))
	adminRoutes := r.Group("/").Use(middlewares.Authentication([]int{2, 3, 100}))

	publicRoutes.POST("/register", controllers.RegisterUser())
	publicRoutes.POST("/verify", controllers.VerifyUser())
	publicRoutes.POST("/login", controllers.LoginUser())

	authRoutes.GET("/me", controllers.GetCurrentUser())
	authRoutes.PUT("/update-user-info", controllers.UpdateUserInfo())
	authRoutes.PUT("/update-user-password", controllers.UpdateUserPassword())
	authRoutes.PUT("/upload-avatar", controllers.UploadAvatar())

	adminRoutes.GET("/get-branch-users/:branch_id", controllers.GetAllBranchUsers())
	adminRoutes.PUT("/update-user-role", controllers.UpdateUserRole())
	adminRoutes.DELETE("/delete-user/:user_id/:branch_id", controllers.DeleteUser())

	publicRoutes.Use(middlewares.Authentication([]int{3, 100})).POST("/create-user", controllers.CreateUser())
	publicRoutes.Use(middlewares.Authentication([]int{3, 100})).PUT("/update-user-branch", controllers.UpdateUserBranch())
	publicRoutes.Use(middlewares.Authentication([]int{100})).GET("/get-all-users", controllers.GetAllUsers())
}

func BranchRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	adminRoutes := r.Group("/").Use(middlewares.Authentication([]int{3, 100}))
	rootRoutes := r.Group("/").Use(middlewares.Authentication([]int{100}))

	publicRoutes.GET("/get-one-branch/:branch_id", controllers.GetOneBranch())

	adminRoutes.PUT("/update-branch/:branch_id", controllers.UpdateBranch())
	adminRoutes.POST("/create-branch", controllers.CreateBranch())
	adminRoutes.GET("/get-all-branches", controllers.GetBranches())

	rootRoutes.DELETE("/delete-branch/:branch_id", controllers.DeleteBranch())
}

func CategoryRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	adminRoutes := r.Group("/").Use(middlewares.Authentication([]int{2, 3, 100}))

	publicRoutes.GET("/get-all-categories/:branch_id", controllers.GetAllCategories())
	publicRoutes.GET("/get-one-category/:category_id", controllers.GetOneCategory())

	adminRoutes.PUT("/update-category/:category_id", controllers.UpdateCategory())
	adminRoutes.POST("/create-category", controllers.CreateCategory())
	adminRoutes.DELETE("/delete-category/:category_id", controllers.DeleteCategory())
}

func MenuRoutes(r *gin.Engine) {
	publicRoutes := r.Group("/")
	adminRoutes := r.Group("/").Use(middlewares.Authentication([]int{2, 3, 100}))

	publicRoutes.GET("/get-menus-by-branchID/:branch_id", controllers.GetAllMenusByBranchID())
	publicRoutes.GET("/get-menus-by-categoryID/:category_id", controllers.GetAllMenusByCategoryID())
	publicRoutes.GET("/get-one-menu/:menu_id", controllers.GetOneMenu())
	publicRoutes.GET("/search-menu", controllers.SearchMenu())

	adminRoutes.PUT("/update-menu/:menu_id", controllers.UpdateMenu())
	adminRoutes.PUT("/add-menu-addon/:menu_id", controllers.AddMenuAddOn())
	adminRoutes.PUT("/update-menu-addon/:menu_id/:add_on_id", controllers.UpdateMenuAddOn())
	adminRoutes.PUT("/remove-menu-addon/:menu_id/:add_on_id", controllers.RemoveMenuAddOn())
	adminRoutes.POST("/create-menu", controllers.CreateMenu())
	adminRoutes.DELETE("/delete-menu/:menu_id", controllers.DeleteMenu())
}
