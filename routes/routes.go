package routes

import (
	"main/controller"
	middleware "main/middlewares"
	"main/ws"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(router *gin.RouterGroup, uc *controller.UserController, ws *ws.WsController) {

	UserRouter := router.Group("/users")
	{
		UserRouter.POST("/signup", uc.SignUp)
		UserRouter.POST("/login", uc.Login)

		authRoutes := UserRouter.Group("/")
		authRoutes.Use(middleware.AuthMiddleware())
		authRoutes.POST("/logout", uc.Logout)
		authRoutes.GET("/:id", uc.GetUser)
		authRoutes.GET("/", uc.GetUsers)
		authRoutes.PUT("/change-password", uc.ChangePassword)
		authRoutes.PUT("/:id", uc.UpdateUser)
		authRoutes.DELETE("/:id", uc.DeleteUser)
	}

	wsRouter := router.Group("/ws")
	{
		authRoutes := wsRouter.Group("/").Use(middleware.AuthMiddleware())
		authRoutes.POST("/create-room", ws.CreateRoom)
		authRoutes.GET("/join-room/:roomId", ws.JoinRoom)
		authRoutes.GET("/getRooms", ws.GetRooms)
		authRoutes.GET("/getClients/:roomId", ws.GetClients)
	}
}
