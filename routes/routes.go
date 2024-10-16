package routes

import (
	"ddos-protection-api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Инициализация маршрутов
func InitRoutes(router *gin.Engine) {
	router.POST("/register", services.RegisterHandler)
	router.POST("/login", services.LoginHandler)
	router.POST("/block/:ip", AuthMiddleware(), services.BlockIPHandler)

	// Новый маршрут для получения отчета
	router.GET("/report/blocks", AuthMiddleware(), services.IPWeightReportHandler)
	router.GET("/docs/report", services.BlockReportHandler)
	// Документация
	router.LoadHTMLGlob("templates/*")

	router.GET("/docs/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})

	router.GET("/docs/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	router.GET("/docs/block", func(c *gin.Context) {
		c.HTML(http.StatusOK, "block.html", nil)
	})

	router.GET("/ws", services.WebSocketHandler) // новый маршрут для WebSocket
	router.GET("/active_sessions", services.GetActiveSessionsHandler)
	router.DELETE("/user/delete", AuthMiddleware(), services.DeleteUserHandler)

}
