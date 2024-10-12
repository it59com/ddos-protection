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
	router.GET("/report/blocks", AuthMiddleware(), services.BlockReportHandler)

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

	router.GET("/docs/report", AuthMiddleware(), services.IPWeightReportHandler)

	//IPWeightReportHandler
}
