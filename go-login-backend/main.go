package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv" // 用于读取 .env 文件
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
)

// 全局变量
var (
	db        *sql.DB
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
)

// 数据库配置结构体
type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

// 用户模型
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"` // 序列化时忽略密码字段
}

// 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"` // binding:"required" 确保字段不能为空
}

// 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 登录响应
type LoginResponse struct {
	Code    int    `json:"code"`    // 状态码，如200等
	Message string `json:"message"` //如success等
	Data    struct {
		Token string `json:"token"` // JWT令牌
		User  *User  `json:"user"`  // 用户信息
	} `json:"data"`
}

// 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// 成功响应
type SuccessResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 初始化环境变量
func initEnv() error {
	if err := godotenv.Load(); err != nil {
		log.Printf("警告: 未找到 .env 文件，将使用系统环境变量")
	}
	return nil
}

// 获取数据库配置
func getDBConfig() *DBConfig {
	return &DBConfig{
		User:     getEnv("DB_USER", "root"),
		Password: getEnv("DB_PASSWORD", ""),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "3306"),
		Name:     getEnv("DB_NAME", "test"),
	}
}

// 获取环境变量，支持默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 初始化数据库连接
func initDB() error {
	config := getDBConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User, config.Password, config.Host, config.Port, config.Name)
	//目的：构建一个 DSN（Data Source Name） 字符串，这是 Go 的 database/sql 驱动用于连接数据库的标准连接字符串。

	//创建与数据库链接的连接池
	var err error
	db, err = sql.Open("mysql", dsn)

	if err != nil {
		return fmt.Errorf("数据库连接失败: %v", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("数据库ping失败: %v", err)
	}
	//目的：真正地检查数据库网络连接是否可用。
	//详解：
	//db.Ping()：这个方法会从连接池中获取一个连接（或新建一个），并执行一条简单的查询（如 MySQL 的 SELECT 1）来验证与数据库的通信是否正常。

	db.SetMaxOpenConns(25)                 //设置连接池中最大打开的连接数。
	db.SetMaxIdleConns(25)                 //设置连接池中最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) //目的：设置连接的最大存活时间。详解：即使连接是空闲的，超过这个时间后它也会被关闭并重新建立。

	log.Println("✅ 数据库连接成功")
	return nil
}

// 根据用户名查询用户
func getUserByUsername(username string) (*User, error) {
	var user User
	query := "SELECT id, username, password FROM users WHERE username = ?"
	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}
	return &user, nil
}

// 生成 JWT Token
func generateJWT(username string) (string, error) {
	secret := getJWTSecret()
	if len(secret) == 0 {
		return "", fmt.Errorf("JWT 密钥未设置")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"iat": jwt.NewNumericDate(time.Now()),                     //签发时间
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), //过期时间
	})

	return token.SignedString(secret) //使用密钥对Token进行签名
}

// 获取 JWT 密钥
func getJWTSecret() []byte {
	if len(jwtSecret) == 0 {
		if secret := os.Getenv("JWT_SECRET"); secret != "" {
			jwtSecret = []byte(secret)
			return jwtSecret
		}
		log.Println("⚠️  警告: JWT_SECRET 未设置，使用默认密钥（仅用于开发）")
		return []byte("insecure-default-secret-key-for-development")
	}
	return jwtSecret
}

// JWT 认证中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:    401,
				Message: "请求头中未提供 Authorization 令牌",
			})
			c.Abort() //至关重要。它阻止这个请求链中后续的所有处理程序（包括最终的路由处理函数）被执行。
			return
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:    401,
				Message: "令牌格式错误，应为: Bearer <token>",
			})
			c.Abort()
			return
		}
		tokenString := authHeader[7:]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
			}
			return getJWTSecret(), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:    401,
				Message: "无效的令牌: " + err.Error(),
			})
			c.Abort()
			return
		}
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:    401,
				Message: "令牌已过期或无效",
			})
			c.Abort()
			return
		}

		c.Set("user", token)
		c.Next()
	}
}

// 登录处理函数
func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    400,
			Message: "无效的请求格式: " + err.Error(),
		})
		return
	}

	user, err := getUserByUsername(req.Username)
	if err != nil {
		log.Printf("数据库查询错误: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    500,
			Message: "服务器内部错误",
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	tokenString, err := generateJWT(user.Username)
	if err != nil {
		log.Printf("生成Token错误: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    500,
			Message: "生成令牌失败",
		})
		return
	}

	response := LoginResponse{
		Code:    200,
		Message: "登录成功",
	}
	response.Data.Token = tokenString
	response.Data.User = user

	c.JSON(http.StatusOK, response)
}

// 注册处理函数
func registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    400,
			Message: "无效的请求格式: " + err.Error(),
		})
		return
	}

	existingUser, err := getUserByUsername(req.Username)
	if err != nil {
		log.Printf("查询用户错误: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    500,
			Message: "服务器内部错误",
		})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Code:    409,
			Message: "用户名已存在",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码哈希错误: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    500,
			Message: "密码加密失败",
		})
		return
	}

	query := "INSERT INTO users (username, password) VALUES (?, ?)"
	result, err := db.Exec(query, req.Username, string(hashedPassword))
	if err != nil {
		log.Printf("插入用户错误: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    500,
			Message: "用户注册失败",
		})
		return
	}

	userID, _ := result.LastInsertId()

	c.JSON(http.StatusOK, SuccessResponse{
		Code:    200,
		Message: "注册成功",
		Data: gin.H{
			"id":       userID,
			"username": req.Username,
		},
	})
}

// 受保护的路由示例
func protectedHandler(c *gin.Context) {
	userToken, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    401,
			Message: "用户信息获取失败",
		})
		return
	}
	token := userToken.(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	username := claims["sub"].(string)

	c.JSON(http.StatusOK, SuccessResponse{
		Code:    200,
		Message: "访问受保护资源成功",
		Data: gin.H{
			"username": username,
			"message":  fmt.Sprintf("你好, %s！这是一个受保护的资源。", username),
		},
	})
}

// 健康检查端点
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, SuccessResponse{
		Code:    200,
		Message: "服务正常运行",
		Data: gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func main() {
	if err := initEnv(); err != nil {
		log.Printf("环境变量初始化警告: %v", err)
	}
	if err := initDB(); err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("关闭数据库连接时出错: %v", err)
		} else {
			log.Println("✅ 数据库连接已关闭")
		}
	}()

	router := gin.Default()
	router.Use(corsMiddleware())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 公共路由
	router.GET("/api/health", healthHandler)
	router.POST("/api/login", loginHandler)
	router.POST("/api/register", registerHandler)

	// 受保护的路由
	protected := router.Group("/api")
	protected.Use(authMiddleware())
	{
		protected.GET("/protected", protectedHandler)
		protected.GET("/user/profile", protectedHandler)
	}

	serverPort := getEnv("SERVER_PORT", "8080")
	log.Printf("🚀 服务器启动在 http://localhost:%s", serverPort)
	log.Printf("📊 健康检查: http://localhost:%s/api/health", serverPort)
	log.Printf("🔐 登录端点: http://localhost:%s/api/login", serverPort)
	log.Printf("📝 注册端点: http://localhost:%s/api/register", serverPort)

	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("❌ 服务器启动失败: %v", err)
	}
}
