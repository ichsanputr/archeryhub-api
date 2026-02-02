package main

import (
	"io"
	"net/http"
	"os"

	"archeryhub-api/database"
	"archeryhub-api/handler"
	"archeryhub-api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// initLogger initializes the global Logrus logger
func initLogger() {
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "function",
		},
	})

	if err := os.MkdirAll("logs", 0755); err != nil {
		logger.WithError(err).Error("Failed to create logs directory")
		return
	}

	logFile, err := os.OpenFile("logs/api.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.WithError(err).Error("Failed to open log file")
		return
	}

	logger.SetOutput(io.MultiWriter(os.Stdout, logFile))
	logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))
	gin.DefaultWriter = io.MultiWriter(os.Stdout, logFile)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
			logrus.FieldKeyFunc:  "function",
		},
	})

	logger.Info("Global logger initialized successfully")
}

func main() {
	// Initialize logger
	initLogger()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found")
	}

	// Initialize database
	logger.Info("Initializing database connection")
	db, err := database.InitDB()
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()
	logger.Info("Database connection established successfully")

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowedOrigins := []string{
			"https://archeryhub.id",
			"http://localhost:9000",
			"http://localhost:3000",
			"http://127.0.0.1:9000",
			"http://127.0.0.1:3000",
		}

		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Archeryhub.id API",
			"status":  "running",
		})
	})

	// Serve static media files
	r.Static("/media", "./media")

	// API routes
	api := r.Group("/api/v1")
	{
		// Health check
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"message": "Archery Hub API is Running",
				"version": "1.0.0",
			})
		})

		// Authentication routes (public)
		auth := api.Group("/auth")
		{
			// Traditional auth
			auth.POST("/register", handler.Register(db))
			auth.POST("/login", handler.Login(db))
			auth.POST("/logout", handler.Logout())
			auth.GET("/check-name", handler.CheckNameExists(db))
			auth.GET("/me", middleware.AuthMiddleware(), handler.GetCurrentUser(db))

			// Google OAuth
			auth.GET("/google", handler.InitiateGoogleAuth(db))
			auth.GET("/google/callback", handler.GoogleCallback(db))
			auth.POST("/google/callback", handler.GoogleCallback(db))
			auth.GET("/sample-user", handler.GetSampleUser(db))
			auth.GET("/avatar/:identifier", handler.GetArcherProfileImage(db))
		}

		// User routes
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{
			user.GET("", handler.GetCurrentUser(db))
			user.GET("/profile", handler.GetUserProfile(db))
			user.PUT("/profile", handler.UpdateUserProfile(db)) // Generic profile update handler
			user.PUT("/password", handler.UpdatePassword(db))
		}

		// Event routes
		events := api.Group("/events")
		events.Use(middleware.OptionalAuthMiddleware())
		{
			// Public Event routes
			events.GET("", handler.GetEvents(db))
			events.GET("/:id", handler.GetEventByID(db))
			events.GET("/:id/categories", handler.GetEventEvents(db))
			events.GET("/:id/participants", handler.GetEventParticipants(db))
			events.GET("/:id/participants/:participantId", handler.GetEventParticipant(db))
			events.PUT("/:id/participants/:participantId", middleware.AuthMiddleware(), handler.UpdateEventParticipant(db))
			events.DELETE("/:id/participants/:participantId", middleware.AuthMiddleware(), handler.DeleteEventParticipant(db))
			events.DELETE("/participants/:participantId", middleware.AuthMiddleware(), handler.CancelParticipantRegistration(db))
			events.GET("/:id/teams", handler.GetEventTeams(db))
			events.GET("/:id/images", handler.GetEventImages(db))
			events.GET("/:id/schedule", handler.GetEventSchedule(db))
			events.GET("/:id/target-names", handler.GetTargetNames(db))
			events.GET("/:id/payment-methods", handler.GetEventPaymentMethods(db))

			// Protected Event routes (require authentication)
			protected := events.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/my", handler.GetMyEvents(db))
				protected.POST("", handler.CreateEvent(db))
				protected.PUT("/:id", handler.UpdateEvent(db))
				protected.DELETE("/:id", handler.DeleteEvent(db))
				protected.POST("/:id/publish", handler.PublishEvent(db))
				protected.POST("/:id/categories", handler.CreateEventCategory(db))
				protected.POST("/:id/categories/batch", handler.CreateEventCategories(db))
				protected.PUT("/:id/categories/:categoryId", handler.UpdateEventCategory(db))
				protected.DELETE("/:id/categories/:categoryId", handler.DeleteEventCategory(db))
				protected.POST("/:id/participants", handler.RegisterParticipant(db))
				protected.PUT("/:id/images", handler.UpdateEventImages(db))
				protected.PUT("/:id/schedule", handler.UpdateEventSchedule(db))
				protected.POST("/:id/payment-methods", handler.CreateEventPaymentMethod(db))
				protected.PUT("/:id/payment-methods/:methodId", handler.UpdateEventPaymentMethod(db))
				protected.DELETE("/:id/payment-methods/:methodId", handler.DeleteEventPaymentMethod(db))

			}
		}

		// Event category reference routes
		api.GET("/event-categories", handler.ListEventCategoryRefs(db))
		api.POST("/event-categories", handler.CreateEventCategoryRef(db))
		api.PUT("/event-categories/:id", handler.UpdateEventCategoryRef(db))

		// Archer routes
		archers := api.Group("/archers")
		{
			// Public archer routes
			archers.GET("", handler.GetArchers(db))
			archers.GET("/:id", handler.GetArcherByID(db))
			archers.GET("/:id/events", handler.GetArcherEvents(db))

			// Protected archer routes
			protected := archers.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/me", handler.GetArcherProfile(db))
				protected.POST("", handler.CreateArcher(db))
				protected.PUT("/:id", handler.UpdateArcher(db))
				protected.DELETE("/:id", handler.DeleteArcher(db))
			}
		}

		// Reference data routes
		api.GET("/disciplines", handler.GetDisciplines(db))
		api.GET("/bow-types", handler.GetBowTypes(db))
		api.GET("/team-types", handler.GetEventTypes(db))
		api.GET("/gender-divisions", handler.GetGenderDivisions(db))
		api.GET("/age-groups", handler.GetAgeGroups(db))
		api.GET("/cities", handler.GetCities())

		// News routes
		news := api.Group("/news")
		{
			// Public news routes
			news.GET("", handler.GetNewsPublic(db))
			news.GET("/:id", handler.GetNewsByID(db))

			// Protected news routes
			protectedNews := news.Group("")
			protectedNews.Use(middleware.AuthMiddleware())
			{
				protectedNews.GET("/my", handler.GetNews(db))
				protectedNews.POST("", handler.CreateNews(db))
				protectedNews.PUT("/:id", handler.UpdateNews(db))
				protectedNews.DELETE("/:id", handler.DeleteNews(db))
			}
		}

		// Back Numbers & Target Assignments
		api.GET("/events/:id/back-numbers", handler.GetBackNumbers(db))
		api.PUT("/participants/:participantId/assignment", middleware.AuthMiddleware(), handler.UpdateBackNumber(db))

		// Team Management
		teams := api.Group("/teams")
		{
			teams.GET("/event/:eventId", handler.GetTeams(db))
			teams.GET("/:teamId", handler.GetTeam(db))

			protectedTeams := teams.Group("")
			protectedTeams.Use(middleware.AuthMiddleware())
			{
				protectedTeams.GET("/my", handler.GetMyTeams(db))
				protectedTeams.POST("/event/:eventId", handler.CreateTeam(db))
			}

			teams.GET("/event/:eventId/rankings", handler.GetTeamRankings(db))
		}

		// Payment & Registration routes
		payment := api.Group("/payment")
		{
			payment.GET("/channels", handler.GetPaymentChannels(db))
			payment.GET("/status/:reference", handler.GetPaymentStatus(db))
			payment.POST("/create", middleware.AuthMiddleware(), handler.CreatePayment(db))
			payment.POST("/tripay/callback", handler.PaymentCallback(db))
		}

		// Event registration is handled via POST /events/:id/participants

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	logger.WithField("port", port).Info("Archery Hub API starting")
	logger.Fatal(r.Run(":" + port))
}
}
