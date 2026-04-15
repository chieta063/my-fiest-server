package main

import (
	"cmp"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Message はGORMのモデル。タグでテーブルのカラムと対応させる
type Message struct {
	gorm.Model        // ID, CreatedAt, UpdatedAt, DeletedAt を自動で追加してくれる
	Body       string `json:"body" gorm:"not null"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	gin.SetMode(gin.ReleaseMode)

	// GORMでDB接続
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// マイグレーション（Messageモデルに対応するテーブルを自動で作成・更新する）
	db.AutoMigrate(&Message{})

	r := gin.New()
	r.Use(sloggin.New(logger))
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello world!",
		})
	})

	r.GET("/hello/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(200, gin.H{
			"message": "Hello, " + name + "!",
		})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/db-check", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(500, gin.H{"error": "DB connection failed"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{"error": "DB connection failed"})
			return
		}
		c.JSON(200, gin.H{"message": "DB connected!"})
	})

	// メッセージを保存するエンドポイント
	r.POST("/messages", func(c *gin.Context) {
		var input struct {
			Body string `json:"body"`
		}
		// リクエストのJSONを受け取る
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		// GORMでINSERT
		message := Message{Body: input.Body}
		result := db.Create(&message)
		if result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to save message"})
			return
		}
		c.JSON(201, message)
	})

	// メッセージ一覧を取得するエンドポイント
	r.GET("/messages", func(c *gin.Context) {
		var messages []Message
		// GORMでSELECT
		result := db.Order("created_at desc").Find(&messages)
		if result.Error != nil {
			c.JSON(500, gin.H{"error": "Failed to get messages"})
			return
		}
		c.JSON(200, messages)
	})

	port := cmp.Or(os.Getenv("PORT"), "8080")
	logger.Info("Server starting", slog.String("port", port))
	r.Run((":" + port))
}
