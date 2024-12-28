package routes

import (
	controllers "nano_food_api/controllers"

	"github.com/gin-gonic/gin"
)

type RouteGroups struct {
	Public    *gin.RouterGroup
	Auth      gin.IRoutes
	Assistant gin.IRoutes
	Manager   gin.IRoutes
	Admin     gin.IRoutes
	Root      gin.IRoutes
}

func UserRoutes(r *RouteGroups) {
	r.Public.POST("/register", controllers.RegisterUser())
	r.Public.POST("/verify", controllers.VerifyUser())
	r.Public.POST("/login", controllers.LoginUser())

	r.Auth.GET("/me", controllers.GetCurrentUser())
	r.Auth.PUT("/update-user-info", controllers.UpdateUserInfo())
	r.Auth.PUT("/update-user-password", controllers.UpdateUserPassword())
	r.Auth.PUT("/upload-avatar", controllers.UploadAvatar())

	r.Manager.GET("/get-branch-users/:branch_id", controllers.GetAllBranchUsers())
	r.Manager.PUT("/update-user-role", controllers.UpdateUserRole())

	r.Admin.DELETE("/delete-user/:user_id/:branch_id", controllers.DeleteUser())
	r.Admin.POST("/create-user", controllers.CreateUser())
	r.Admin.PUT("/update-user-branch", controllers.UpdateUserBranch())
	r.Root.GET("/get-all-users", controllers.GetAllUsers())
}

func BranchRoutes(r *RouteGroups) {
	r.Public.GET("/get-one-branch/:branch_id", controllers.GetOneBranch())

	r.Admin.PUT("/update-branch/:branch_id", controllers.UpdateBranch())
	r.Admin.POST("/create-branch", controllers.CreateBranch())
	r.Admin.GET("/get-all-branches", controllers.GetBranches())

	r.Root.DELETE("/delete-branch/:branch_id", controllers.DeleteBranch())
}

func CategoryRoutes(r *RouteGroups) {
	r.Public.GET("/get-all-categories/:branch_id", controllers.GetAllCategories())
	r.Public.GET("/get-one-category/:category_id", controllers.GetOneCategory())

	r.Manager.PUT("/update-category/:category_id", controllers.UpdateCategory())
	r.Manager.POST("/create-category", controllers.CreateCategory())
	r.Admin.DELETE("/delete-category/:category_id", controllers.DeleteCategory())
}

func TableRoutes(r *RouteGroups) {
	r.Public.GET("/get-all-tables/:branch_id", controllers.GetAllTables())
	r.Public.GET("/get-one-table/:table_id", controllers.GetOneTable())

	r.Manager.PUT("/update-table/:table_id", controllers.UpdateTable())
	r.Manager.POST("/create-table", controllers.CreateTable())
	r.Admin.DELETE("/delete-table/:table_id", controllers.DeleteTable())
}

func MenuRoutes(r *RouteGroups) {
	r.Public.GET("/get-menus-by-branchID/:branch_id", controllers.GetAllMenusByBranchID())
	r.Public.GET("/get-menus-by-categoryID/:category_id", controllers.GetAllMenusByCategoryID())
	r.Public.GET("/get-one-menu/:menu_id", controllers.GetOneMenu())
	r.Public.GET("/search-menu", controllers.SearchMenu())

	r.Manager.PUT("/update-menu/:menu_id", controllers.UpdateMenu())
	r.Manager.POST("/create-menu", controllers.CreateMenu())
	r.Admin.DELETE("/delete-menu/:menu_id", controllers.DeleteMenu())
}

func AddOnRoutes(r *RouteGroups) {
	r.Public.GET("/get-all-addons", controllers.GetAllAddOns())
	r.Public.GET("/get-one-addon/:add_on_id", controllers.GetOneAddOn())

	r.Manager.POST("/create-addon", controllers.AddMenuAddOn())
	r.Manager.PUT("/update-addon/:add_on_id", controllers.UpdateMenuAddOn())
	r.Manager.DELETE("/delete-addon/:add_on_id", controllers.RemoveMenuAddOn())
}

func OrderRoutes(r *RouteGroups) {
	r.Public.GET("/get-all-orders", controllers.GetAllOrders())
	r.Public.GET("/get-one-order/:order_id", controllers.GetOneOrder())
	r.Public.POST("/create-order", controllers.CreateOrder())

	r.Manager.PUT("/update-order/:order_id", controllers.UpdateOrder())
	r.Admin.DELETE("/delete-order/:order_id", controllers.DeleteOrder())
}

func SaleRoutes(r *RouteGroups) {
	r.Manager.GET("/get-all-sales", controllers.GetAllSales())
	r.Public.GET("/get-one-sale/:sale_id", controllers.GetOneSale())
	r.Public.POST("/create-sale", controllers.CreateSale())

	r.Admin.DELETE("/delete-sale/:sale_id", controllers.DeleteSale())
}
