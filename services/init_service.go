package services

import (
	"ddos-protection-api/auth"
	"ddos-protection-api/config"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Инициализация Redis
func InitRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.Redis.Address,
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Ошибка при подключении к Redis: %v", err)
	}
	fmt.Println("Подключение к Redis установлено успешно.")
}

// Функция для регистрации нового пользователя
func RegisterHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат данных"})
		return
	}

	if err := auth.RegisterUser(req.Email, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "пользователь успешно зарегистрирован"})
}
