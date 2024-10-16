package routes

import (
	"ddos-protection-api/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// // Middleware для защиты маршрутов с использованием Bearer токена
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "требуется авторизация"})
			c.Abort()
			return
		}

		// Проверка формата заголовка Authorization
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "неправильный формат токена"})
			c.Abort()
			return
		}

		// Проверка токена
		token := parts[1]
		
		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "недействительный токен"})
			c.Abort()
			return
		}

		// Передаем email из токена в контекст запроса
		c.Set("email", claims.Email)
		c.Next()
	}
}
