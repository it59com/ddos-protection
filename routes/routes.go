package routes

import (
	"context"
	"ddos-protection-api/auth"
	"ddos-protection-api/db"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

const (
	requestLimit = 10               // Лимит запросов от одного IP
	timeWindow   = 60 * time.Second // Временное окно в секундах
)

// Инициализация Redis
func InitRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Замените на адрес вашего Redis-сервера
	})
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

// Функция для аутентификации пользователя
func LoginHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат данных"})
		return
	}

	token, err := auth.LoginUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Функция для блокировки IP-адреса
// Обработчик блокировки IP-адреса
func BlockIPHandler(c *gin.Context) {
	ip := c.Param("ip")
	host := c.ClientIP()
	firewall := c.DefaultQuery("firewall", "unknown")

	// Проверка и учет количества запросов от IP
	blocked, err := trackIPRequests(ip, host, firewall)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if blocked {
		c.JSON(http.StatusOK, gin.H{"message": "IP заблокирован"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Запрос зафиксирован"})
	}
}

func trackIPRequests(ip string, host string, firewall string) (bool, error) {
	// SQL-запрос для вставки или обновления записи в таблице requests
	query := `INSERT INTO requests (ip, host, request_count, last_request) 
			  VALUES (?, ?, 1, CURRENT_TIMESTAMP)
			  ON CONFLICT(ip, host) DO UPDATE SET 
			  request_count = request_count + 1,
			  last_request = CURRENT_TIMESTAMP;`

	_, err := db.DB.Exec(query, ip, host)
	if err != nil {
		return false, fmt.Errorf("ошибка обновления данных о запросе: %w", err)
	}

	// Использование Redis для учета лимита запросов
	key := fmt.Sprintf("req_count:%s", ip)
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("ошибка инкремента счетчика в Redis: %w", err)
	}

	// Устанавливаем TTL для ключа, если это первый запрос
	if count == 1 {
		rdb.Expire(ctx, key, timeWindow)
	}

	// Проверка, если запросы превышают лимит, блокируем IP
	if count > requestLimit {
		err := db.AddToDatabase(ip, firewall, int(count))
		if err != nil {
			return false, fmt.Errorf("ошибка добавления IP в базу данных: %w", err)
		}
		return true, nil
	}
	return false, nil
}

// Новый маршрут для формирования отчета по блокировкам
func BlockReportHandler(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT ip, host, request_count, last_request 
		FROM requests 
		WHERE request_count >= ? 
		ORDER BY last_request DESC
	`, requestLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка получения отчета"})
		return
	}
	defer rows.Close()

	var report []map[string]interface{}
	for rows.Next() {
		var ip, host string
		var requestCount int
		var lastRequest time.Time

		if err := rows.Scan(&ip, &host, &requestCount, &lastRequest); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при сканировании данных отчета"})
			return
		}

		report = append(report, map[string]interface{}{
			"ip":            ip,
			"host":          host,
			"request_count": requestCount,
			"last_request":  lastRequest,
		})
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}

// Инициализация маршрутов
func InitRoutes(router *gin.Engine) {
	router.POST("/register", RegisterHandler)
	router.POST("/login", LoginHandler)
	router.POST("/block/:ip", AuthMiddleware(), BlockIPHandler)

	// Новый маршрут для получения отчета
	router.GET("/report/blocks", AuthMiddleware(), BlockReportHandler)

	// Документация
	router.GET("/docs/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})
	router.GET("/docs/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	router.GET("/docs/block", func(c *gin.Context) {
		c.HTML(http.StatusOK, "block.html", nil)
	})
	router.GET("/docs/report", func(c *gin.Context) {
		c.HTML(http.StatusOK, "report.html", nil)
	})
}
