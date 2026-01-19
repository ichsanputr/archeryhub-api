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
			auth.GET("/me", middleware.AuthMiddleware(), handler.GetCurrentUser(db))
			
			// Google OAuth
			auth.GET("/google", handler.InitiateGoogleAuth(db))
			auth.GET("/google/callback", handler.GoogleCallback(db))
			auth.POST("/google/callback", handler.GoogleCallback(db))
		}

		// User route (alias for /auth/me, needed for frontend compatibility)
		api.GET("/user", middleware.AuthMiddleware(), handler.GetCurrentUser(db))

		// Tournament routes
		tournaments := api.Group("/tournaments")
		{
			// Public tournament routes
			tournaments.GET("", handler.GetTournaments(db))
			tournaments.GET("/:id", handler.GetTournamentByID(db))
			tournaments.GET("/:id/events", handler.GetTournamentEvents(db))
			tournaments.GET("/:id/participants", handler.GetTournamentParticipants(db))

			// Protected tournament routes (require authentication)
			protected := tournaments.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.POST("", handler.CreateTournament(db))
				protected.PUT("/:id", handler.UpdateTournament(db))
				protected.DELETE("/:id", handler.DeleteTournament(db))
				protected.POST("/:id/publish", handler.PublishTournament(db))
				protected.POST("/:id/participants", handler.RegisterParticipant(db))
			}
		}

		// Athlete routes
		athletes := api.Group("/athletes")
		{
			// Public athlete routes
			athletes.GET("", handler.GetAthletes(db))
			athletes.GET("/:id", handler.GetAthleteByID(db))

			// Protected athlete routes
			protected := athletes.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.POST("", handler.CreateAthlete(db))
				protected.PUT("/:id", handler.UpdateAthlete(db))
				protected.DELETE("/:id", handler.DeleteAthlete(db))
			}
		}

		// Division & Category routes (public, read-only)
		api.GET("/divisions", handler.GetDivisions(db))
		api.GET("/categories", handler.GetCategories(db))

		// Tournament Sessions
		api.GET("/tournaments/:id/sessions", handler.GetSessions(db))
		api.POST("/tournaments/:id/sessions", middleware.AuthMiddleware(), handler.CreateSession(db))
		api.PUT("/tournaments/:id/sessions/:sessionId", middleware.AuthMiddleware(), handler.UpdateSession(db))
		api.DELETE("/tournaments/:id/sessions/:sessionId", middleware.AuthMiddleware(), handler.DeleteSession(db))

		// Tournament Officials/Staff
		api.GET("/tournaments/:id/officials", handler.GetOfficials(db))
		api.POST("/tournaments/:id/officials", middleware.AuthMiddleware(), handler.CreateOfficial(db))
		api.PUT("/tournaments/:id/officials/:officialId", middleware.AuthMiddleware(), handler.UpdateOfficial(db))
		api.DELETE("/tournaments/:id/officials/:officialId", middleware.AuthMiddleware(), handler.DeleteOfficial(db))

		// Distances Configuration
		api.GET("/tournaments/:id/distances", handler.GetDistances(db))
		api.POST("/distances", middleware.AuthMiddleware(), handler.CreateDistance(db))
		api.PUT("/distances/:distanceId", middleware.AuthMiddleware(), handler.UpdateDistance(db))

		// Back Numbers & Target Assignments
		api.GET("/tournaments/:id/back-numbers", handler.GetBackNumbers(db))
		api.PUT("/participants/:participantId/assignment", middleware.AuthMiddleware(), handler.UpdateBackNumber(db))

		// Qualification Scoring
		qualification := api.Group("/qualification")
		{
			qualification.POST("/scores", middleware.AuthMiddleware(), handler.SubmitQualificationScore(db))
			qualification.GET("/participants/:participantId/scores", handler.GetQualificationScores(db))
			qualification.GET("/:tournamentId/rankings", handler.GetQualificationRankings(db))
		}

		// Elimination Matches
		elimination := api.Group("/elimination")
		{
			elimination.POST("/bracket", middleware.AuthMiddleware(), handler.CreateEliminationBracket(db))
			elimination.GET("/events/:eventId/bracket", handler.GetEliminationBracket(db))
			elimination.PUT("/matches/:matchId/score", middleware.AuthMiddleware(), handler.UpdateMatchScore(db))
		}

		// Team Management
		teams := api.Group("/teams")
		{
			teams.GET("/tournament/:tournamentId", handler.GetTeams(db))
			teams.GET("/:teamId", handler.GetTeam(db))
			teams.POST("/tournament/:tournamentId", middleware.AuthMiddleware(), handler.CreateTeam(db))
			teams.POST("/tournament/:tournamentId/generate", middleware.AuthMiddleware(), handler.MakeTeams(db))
			teams.POST("/scores", middleware.AuthMiddleware(), handler.SubmitTeamScore(db))
			teams.GET("/tournament/:tournamentId/rankings", handler.GetTeamRankings(db))
		}

		// Finals & Match Management
		finals := api.Group("/finals")
		{
			finals.GET("/events/:eventId/rankings", handler.GetFinalRankings(db))
			finals.POST("/events/:eventId/advance", middleware.AuthMiddleware(), handler.AdvanceToNextPhase(db))
			finals.GET("/matches/:matchId", handler.GetMatchDetails(db))
			finals.PUT("/matches/:matchId/schedule", middleware.AuthMiddleware(), handler.SetMatchSchedule(db))
			finals.POST("/matches/:matchId/start", middleware.AuthMiddleware(), handler.StartMatch(db))
			finals.POST("/matches/:matchId/complete", middleware.AuthMiddleware(), handler.CompleteMatch(db))
		}

		// Device Management (for mobile scoring apps)
		devices := api.Group("/devices")
		{
			devices.POST("/register", middleware.AuthMiddleware(), handler.RegisterDevice(db))
			devices.GET("/tournament/:tournamentId", middleware.AuthMiddleware(), handler.GetDevices(db))
			devices.GET("/:deviceCode/config", handler.GetDeviceConfig(db)) // Public for app login
			devices.POST("/:deviceCode/sync", handler.SyncDevice(db))
			devices.PUT("/:deviceCode/status", middleware.AuthMiddleware(), handler.UpdateDeviceStatus(db))
			devices.GET("/:deviceCode/qrcode", middleware.AuthMiddleware(), handler.GetDeviceQRCode(db))
		}

		// Payment & Registration routes
		payment := api.Group("/payment")
		{
			payment.GET("/channels", handler.GetPaymentChannels(db))
			payment.GET("/status/:reference", handler.GetPaymentStatus(db))
			payment.POST("/create", middleware.AuthMiddleware(), handler.CreatePayment(db))
			payment.POST("/tripay/callback", handler.PaymentCallback(db))
		}

		// Tournament registration
		api.POST("/tournaments/:id/register", middleware.AuthMiddleware(), handler.RegisterTournament(db))

		// Live Results & Rankings (public access)
		live := api.Group("/live")
		{
			live.GET("/:tournamentId/rankings", handler.GetQualificationRankings(db))
			live.GET("/events/:eventId/bracket", handler.GetEliminationBracket(db))
		}

		// Statistics (public access)
		stats := api.Group("/statistics")
		{
			stats.GET("/matches/:matchId", handler.GetMatchStatistics(db))
			stats.GET("/events/:eventId", handler.GetEventStatistics(db))
		}

		// Awards & Medals
		awards := api.Group("/awards")
		{
			awards.GET("/tournament/:tournamentId", handler.GetAwards(db))
			awards.GET("/tournament/:tournamentId/medals", handler.GetMedalTable(db))
			awards.POST("/tournament/:tournamentId", middleware.AuthMiddleware(), handler.CreateAward(db))
			awards.POST("/events/:eventId/auto", middleware.AuthMiddleware(), handler.AutoAwardMedals(db))
		}

		// Accreditation & Gate Control
		accreditation := api.Group("/accreditation")
		{
			accreditation.GET("/tournament/:tournamentId", middleware.AuthMiddleware(), handler.GetAccreditations(db))
			accreditation.POST("/tournament/:tournamentId", middleware.AuthMiddleware(), handler.CreateAccreditation(db))
			accreditation.POST("/tournament/:tournamentId/bulk", middleware.AuthMiddleware(), handler.BulkCreateAccreditations(db))
			accreditation.PUT("/:accredId/status", middleware.AuthMiddleware(), handler.UpdateAccreditationStatus(db))
			accreditation.POST("/gate-check", middleware.AuthMiddleware(), handler.GateCheck(db))
			accreditation.GET("/tournament/:tournamentId/gate-situation", handler.GetGateSituation(db))
		}

		// Print Outputs & Reports
		printouts := api.Group("/print")
		{
			printouts.POST("/generate", middleware.AuthMiddleware(), handler.GeneratePrintOutput(db))
			printouts.GET("/export/:type", handler.ExportCSV(db))
		}

		// Admin routes (require admin role)
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		// admin.Use(middleware.RequireRole("admin"))
		{
			// TODO: Implement admin handlers
			// admin.GET("/users", handler.GetAllUsers(db))
			// admin.PUT("/users/:id/role", handler.UpdateUserRole(db))
			// admin.GET("/activity-logs", handler.GetActivityLogs(db))
		}

		// Dashboard Stats
		api.GET("/stats/dashboard", handler.GetDashboardStats(db))
	}

	// WebSocket endpoint for real-time updates
	// TODO: Implement WebSocket handler
	r.GET("/ws/live/:tournamentId", handler.LiveUpdatesWebSocket(db))

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	logger.WithField("port", port).Info("Archery Hub API starting")
	logger.Fatal(r.Run(":" + port))
}
