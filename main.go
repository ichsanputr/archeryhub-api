package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"Archeris-api/database"
	"Archeris-api/handler"
	mobilehandler "Archeris-api/handler/mobile"
	"Archeris-api/middleware"
	"Archeris-api/utils"

	_ "Archeris-api/docs"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Archeris Mobile API
// @version 1.1
// @description Dedicated API for Archeris Mobile App
// @termsOfService http://archeris.net/terms/

// @contact.name Archeris Support
// @contact.url https://archeris.net
// @contact.email support@archeris.net

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host api.archeris.net
// @BasePath /
// @schemes https http

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

var logger *logrus.Logger

// fileOnlyHook is a logrus hook that writes specific log levels to a file
type fileOnlyHook struct {
	file      *os.File
	levels    []logrus.Level
	formatter logrus.Formatter
}

func (h *fileOnlyHook) Levels() []logrus.Level {
	return h.levels
}

func (h *fileOnlyHook) Fire(entry *logrus.Entry) error {
	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.file.Write(line)
	return err
}

// initLogger initializes the global Logrus logger
func initLogger() {
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// stdout: plain text, all levels
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Also configure the global logrus logger used by handlers
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// GIN output: stdout only (not the file)
	gin.DefaultWriter = os.Stdout
	gin.DefaultErrorWriter = os.Stdout

	// File: JSON format, only Error and Fatal
	if err := os.MkdirAll("logs", 0755); err != nil {
		logger.WithError(err).Error("Failed to create logs directory")
		return
	}

	logFile, err := os.OpenFile("logs/api.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.WithError(err).Error("Failed to open log file")
		return
	}

	fileFormatter := &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	}

	// Add hook to write only Error/Fatal to file
	hook := &fileOnlyHook{
		file:      logFile,
		levels:    []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel},
		formatter: fileFormatter,
	}
	logger.AddHook(hook)
	logrus.AddHook(hook)
}

func main() {
	// Initialize logger
	initLogger()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found")
	}

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Assign DB connection for token version checks in auth middleware
	middleware.DB = db

	// Initialize Gin router
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			// Allow localhost / 127.0.0.1 on any port (for Flutter web, Nuxt, etc.) or production domains
			if strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1") ||
				strings.HasPrefix(origin, "https://localhost") ||
				strings.HasPrefix(origin, "https://127.0.0.1") ||
				strings.HasPrefix(origin, "https://archeris.net") ||
				strings.HasPrefix(origin, "http://archeris.net") {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH, HEAD")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Global panic recovery and standardized error response
	r.Use(utils.GlobalRecoveryMiddleware())

	// Security headers middleware
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		if !strings.Contains(c.Request.URL.Path, "/printout") && !strings.Contains(c.Request.URL.Path, "/scoresheet") && !strings.Contains(c.Request.URL.Path, "/statistics-") {
			c.Header("X-Frame-Options", "SAMEORIGIN")
		}
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: blob: https: http:; style-src 'self' 'unsafe-inline' https: http:; script-src 'self' 'unsafe-inline' 'unsafe-eval' https: http:; img-src 'self' data: blob: https: http:; font-src 'self' data: https: http:; frame-ancestors 'self' http://localhost:* https://localhost:* http://127.0.0.1:* https://archeris.net https://*.archeris.net;")
		c.Next()
	})

	// Middleware: log all 5xx responses to file via logrus.Error
	r.Use(func(c *gin.Context) {
		c.Next()
		status := c.Writer.Status()
		if status >= 500 {
			logrus.WithFields(logrus.Fields{
				"status": status,
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
				"errors": c.Errors.String(),
			}).Errorf("[5xx] %s %s -> %d", c.Request.Method, c.Request.URL.Path, status)
		}
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Archeris.net API - TEST DEPLOY 2",
			"status":  "running",
		})
	})

	// Static uploads (media is handled dynamically via media.GET below)
	r.Static("/uploads", "./uploads")

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes (no /api/v1 prefix)
	api := r.Group("/")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"message": "Archeris API is Running",
				"version": "1.0.0",
			})
		})

		// Unified Public Pricing & Plans (SSOT)
		api.GET("/public/pricing/plans", handler.GetUnifiedPricingPlans(db))
		api.GET("/public/quota/plans", handler.GetUnifiedPricingPlans(db))
		api.GET("/subscription/comparison", handler.GetSubscriptionComparison())

		// Session endpoint (Global)
		api.GET("/chatbot/intents", handler.ChatbotIntents())
		api.POST("/chatbot/message", handler.ChatbotMessage())

		api.GET("/archer/me", middleware.AuthMiddleware(), handler.GetArcherProfile(db))
		api.GET("/organizer/me", middleware.AuthMiddleware(), handler.GetOrganizationProfile(db))

		// Project Task Management
		/*
			tasks := api.Group("/tasks")
			tasks.Use(middleware.AuthMiddleware())
			{
				tasks.GET("", handler.GetTasks(db))
				tasks.POST("", handler.CreateTask(db))
				tasks.PUT("/:uuid", handler.UpdateTask(db))
				tasks.PATCH("/:uuid/toggle", handler.ToggleTaskStatus(db))
				tasks.PATCH("/:uuid/status", handler.UpdateTaskStatus(db))
				tasks.DELETE("/:uuid", handler.DeleteTask(db))
			}
		*/

		// Authentication routes (public)
		auth := api.Group("/auth")
		{
			// Traditional auth & OTP registration
			auth.POST("/register", handler.Register(db))
			auth.POST("/register-email", middleware.RateLimit(10, 1*time.Minute), handler.RegisterWithEmail(db))
			auth.POST("/verify-register-otp", middleware.RateLimit(15, 1*time.Minute), handler.VerifyRegisterOTP(db))
			auth.POST("/resend-register-otp", middleware.RateLimit(5, 1*time.Minute), handler.ResendRegisterOTP(db))
			auth.POST("/login", middleware.RateLimit(10, 1*time.Minute), handler.Login(db))
			auth.POST("/refresh", handler.RefreshToken(db))
			auth.POST("/logout", handler.Logout())
			auth.GET("/check-name", handler.CheckNameExists(db))
			auth.GET("/check-username", handler.CheckUsernameExists(db))

			// Google OAuth
			auth.GET("/google", middleware.OptionalAuthMiddleware(), handler.InitiateGoogleAuth(db))
			auth.GET("/google/callback", middleware.OptionalAuthMiddleware(), handler.GoogleCallback(db))
			auth.POST("/google/callback", middleware.OptionalAuthMiddleware(), handler.GoogleCallback(db))

			auth.GET("/avatar/:identifier", handler.GetArcherProfileImage(db))

			// Forgot / Reset password (public — rate limited to prevent abuse)
			auth.POST("/forgot-password", middleware.RateLimit(5, 1*time.Minute), handler.ForgotPassword(db))
			auth.POST("/verify-reset-otp", middleware.RateLimit(10, 1*time.Minute), handler.VerifyResetOTP(db))
			auth.POST("/reset-password", middleware.RateLimit(5, 1*time.Minute), handler.ResetPassword(db))
			auth.POST("/change-password-otp", middleware.RateLimit(5, 1*time.Minute), handler.ChangePasswordWithOTP(db))

			// Alias for mobile login to satisfy public URL expectations
			auth.POST("/archer/login", middleware.RateLimit(10, 1*time.Minute), mobilehandler.MobileArcherLogin(db))
		}

		// Payment cleanup endpoint & background ticker
		api.POST("/payment/cleanup-expired", handler.CleanupExpiredPayments(db))
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			for range ticker.C {
				_, _ = handler.PerformPaymentCleanup(db)
			}
		}()

		// User routes
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{

			user.GET("/profile", handler.GetUserProfile(db))
			user.PUT("/profile", handler.UpdateUserProfile(db)) // Generic profile update handler
			user.PUT("/password", handler.UpdatePassword(db))
			user.POST("/request-email-change", handler.RequestEmailChange(db))
			user.POST("/verify-email-change", handler.VerifyEmailChange(db))
			user.GET("/settings", handler.GetUserSettings(db))
			user.PUT("/settings", handler.UpdateUserSettings(db))
			user.GET("/subscription", handler.GetMySubscription(db))
			user.GET("/subscription/export", handler.ExportInvoicesCSV(db))
		}

		// Event routes
		tournaments := api.Group("/tournaments")
		tournaments.Use(middleware.OptionalAuthMiddleware())
		{
			// External Tournaments (Ianseo scraper)
			tournaments.GET("/external", handler.GetExternalTournaments(db))
			tournaments.GET("/external/:slug", handler.GetExternalTournamentDetail(db))

			// Public Event routes
			tournaments.GET("", middleware.RateLimit(60, 1*time.Minute), handler.GetEvents(db))
			tournaments.GET("/:id", handler.GetEventByID(db))
			tournaments.GET("/:id/categories", handler.GetEventEvents(db))
			tournaments.GET("/:id/participants", handler.GetEventParticipants(db))
			tournaments.GET("/:id/participants/:participantId", handler.GetEventParticipant(db))
			tournaments.GET("/:id/participants/me", middleware.AuthMiddleware(), handler.GetMyEventRegistration(db))
			tournaments.DELETE("/:id/participants/me", middleware.AuthMiddleware(), handler.UnregisterFromEvent(db))
			tournaments.PUT("/:id/participants/:participantId", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateEventParticipant(db))
			tournaments.DELETE("/:id/participants/:participantId", middleware.AuthMiddleware(), handler.DeleteEventParticipant(db))
			tournaments.DELETE("/participants/:participantId", middleware.AuthMiddleware(), handler.CancelParticipantRegistration(db))
			tournaments.POST("/participants/:participantId/payment", middleware.AuthMiddleware(), handler.CreateParticipantPayment(db))
			tournaments.GET("/:id/teams", handler.GetEventTeams(db))
			tournaments.GET("/:id/my-team", middleware.AuthMiddleware(), handler.GetMyEventTeam(db))
			tournaments.GET("/:id/images", handler.GetEventImages(db))
			tournaments.GET("/:id/schedule", handler.GetEventSchedule(db))
			tournaments.GET("/:id/schedules", handler.GetEventSchedule(db))
			tournaments.GET("/:id/target-names", handler.GetTargetNames(db))
			tournaments.GET("/:id/payment-methods", handler.GetEventPaymentMethods(db))
			tournaments.GET("/:id/payments", handler.GetEventPayments(db))
			tournaments.POST("/participants/reregister", handler.ReregisterParticipant(db))
			tournaments.GET("/:id/participants/printout", handler.GetEventParticipantList(db))
			tournaments.GET("/:id/participants/statistics-classes", handler.GetEventStatisticsClasses(db))
			tournaments.GET("/:id/participants/statistics-clubs", handler.GetEventStatisticsClubs(db))
			tournaments.GET("/:id/qualification/start-list/printout", handler.GetQualificationStartListPrintout(db))
			tournaments.GET("/:id/qualification/results/printout", handler.GetQualificationResultsPrintout(db))
			tournaments.GET("/:id/results/medals/printout", handler.GetMedalStandingsPrintout(db))
			tournaments.GET("/:id/targets/labels/printout", handler.GetTargetLabelsPrintout(db))

			// Public Results endpoints
			tournaments.GET("/:id/results/qualification", handler.GetPublicQualificationResults(db))
			tournaments.GET("/:id/results/elimination", handler.GetPublicEliminationResults(db))

			// Protected Event routes (require authentication)
			protected := tournaments.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/my", handler.GetMyEvents(db))
				protected.POST("", middleware.RequireActivePlan(db), handler.CreateEvent(db))
				protected.PUT("/:id", middleware.RequireActivePlan(db), handler.UpdateEvent(db))
				protected.DELETE("/:id", middleware.RequireActivePlan(db), handler.DeleteEvent(db))
				protected.POST("/:id/publish", middleware.RequireActivePlan(db), handler.PublishEvent(db))
				protected.GET("/:id/participants/export", middleware.RequireActivePlan(db), handler.ExportParticipantsCSV(db))
				protected.POST("/:id/categories", middleware.RequireActivePlan(db), handler.CreateEventCategory(db))
				protected.POST("/:id/categories/batch", middleware.RequireActivePlan(db), handler.CreateEventCategories(db))
				protected.GET("/:id/categories/:categoryId", handler.GetEventCategoryDetails(db))
				protected.PUT("/:id/categories/:categoryId", middleware.RequireActivePlan(db), handler.UpdateEventCategory(db))
				protected.DELETE("/:id/categories/:categoryId", middleware.RequireActivePlan(db), handler.DeleteEventCategory(db))
				protected.POST("/:id/participants", handler.RegisterParticipant(db))
				protected.POST("/:id/participants/batch", middleware.RequireActivePlan(db), handler.BatchRegisterParticipants(db))
				protected.POST("/:id/participants/import-csv", middleware.RequireActivePlan(db), handler.ImportParticipantsCSV(db))
				
				// Multi-payment per participant
				protected.GET("/:id/participants/:participantId/payments", handler.GetParticipantPayments(db))
				protected.POST("/:id/participants/:participantId/payments", handler.AddParticipantPayment(db))
				protected.PATCH("/:id/participants/:participantId/approve-payment", handler.ApproveParticipantPayment(db))
				protected.POST("/:id/participants/:participantId/certificate", handler.UploadParticipantCertificate(db))
				
				// Certificate distribution & management
				protected.POST("/:id/certificates/upload-zip", handler.UploadCertificatesZIP(db))
				protected.GET("/:id/certificates/upload-batches", handler.GetCertificateUploadBatches(db))
				protected.GET("/:id/certificates/upload-batches/:batchId/progress", handler.GetBatchProgress(db))
				protected.POST("/:id/certificates/manual-assign", handler.ManualAssignCertificate(db))
				protected.GET("/:id/certificates", handler.GetEventCertificates(db))
				protected.DELETE("/:id/certificates/:certId", handler.DeleteArcherCertificate(db))
				protected.POST("/:id/certificates/generate-all", handler.GenerateAllCertificates(db))
				protected.DELETE("/:id/certificates/clear-all", handler.ClearAllCertificates(db))

				protected.PUT("/:id/images", middleware.RequireActivePlan(db), handler.UpdateEventImages(db))
				protected.PUT("/:id/schedule", middleware.RequireActivePlan(db), handler.UpdateEventSchedule(db))
				protected.PUT("/:id/schedules", middleware.RequireActivePlan(db), handler.UpdateEventSchedule(db))
				protected.POST("/:id/payment-methods", middleware.RequireActivePlan(db), handler.CreateEventPaymentMethod(db))
				protected.PUT("/:id/payment-methods/:methodId", middleware.RequireActivePlan(db), handler.UpdateEventPaymentMethod(db))
				protected.DELETE("/:id/payment-methods/:methodId", middleware.RequireActivePlan(db), handler.DeleteEventPaymentMethod(db))

				// Qualification target assignments - nested under tournaments/:id/qualification/sessions/:sessionId
				protected.POST("/:id/qualification/sessions/:sessionId/assignments", middleware.RequireActivePlan(db), handler.CreateBulkTargetAssignments(db))

				// Target management
				protected.POST("/:id/targets", middleware.RequireActivePlan(db), handler.CreateEventTarget(db))
				protected.PUT("/:id/targets/batch", middleware.RequireActivePlan(db), handler.BatchUpdateTargets(db))
				protected.PUT("/:id/targets/:target_id", middleware.RequireActivePlan(db), handler.UpdateEventTarget(db))
				protected.DELETE("/:id/targets/:target_id", middleware.RequireActivePlan(db), handler.DeleteEventTarget(db))
			}
		}

		// Qualification routes (event-level sessions)
		qualification := api.Group("/tournaments/:id/qualification")
		qualification.Use(middleware.OptionalAuthMiddleware())
		{
			qualification.GET("/sessions", handler.GetQualificationSessions(db))
			qualification.POST("/sessions", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.CreateQualificationSession(db))
			qualification.PATCH("/sessions/:sessionId", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateQualificationSession(db))
			qualification.POST("/sessions/:sessionId/lock", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.ToggleLockQualificationSession(db))
			qualification.GET("/leaderboard", handler.GetQualificationLeaderboard(db))
			qualification.GET("/sessions/:sessionCode/scoresheet", handler.GetQualificationScoresheet(db))
		}

		// Elimination routes (event-level brackets)
		elimination := api.Group("/tournaments/:id/elimination")
		elimination.Use(middleware.OptionalAuthMiddleware())
		{
			elimination.GET("/brackets", handler.GetBrackets(db))
			elimination.GET("/bracket-size", middleware.AuthMiddleware(), handler.GetBracketSizeRecommendation(db))
			elimination.POST("/brackets", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.CreateBracket(db))
			elimination.GET("/brackets/:bracketId", handler.GetBracket(db))
			elimination.PUT("/brackets/:bracketId", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateBracket(db))
			elimination.POST("/brackets/:bracketId/lock", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.ToggleLockEliminationBracket(db))
			elimination.POST("/brackets/:bracketId/generate", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.GenerateBracket(db))
			elimination.GET("/brackets/:bracketId/scores", handler.GetBracketScores(db))
			elimination.GET("/brackets/:bracketId/board-codes", handler.GetEliminationBoardCodes(db))
			elimination.GET("/brackets/:bracketId/scoresheet", handler.GetEliminationScoresheet(db))
			elimination.GET("/brackets/:bracketId/team-members", handler.GetBracketTeamMembers(db))
			elimination.PUT("/brackets/:bracketId/targets", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateMatchTargets(db))
			elimination.POST("/brackets/:bracketId/targets/auto-assign", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.AutoAssignMatchTargets(db))
			elimination.GET("/brackets/:bracketId/matches/:matchId", handler.GetMatch(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/score", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateMatchScore(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/finish", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.FinishMatch(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/end", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.EndMatch(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/reset", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.ResetMatch(db))
		}

		// Public flat match details
		api.GET("/match/:matchId", handler.GetMatch(db))

		qualSessions := api.Group("/qualification/sessions/:sessionId")
		qualSessions.Use(middleware.AuthMiddleware())
		{
			qualSessions.GET("/assignments", handler.GetSessionAssignments(db))
			qualSessions.GET("/board-codes", handler.GetBoardCodes(db))
			qualSessions.GET("/scores", handler.GetSessionScores(db))
			qualSessions.POST("/auto-assign", middleware.RequireActivePlan(db), handler.AutoAssignParticipants(db))
			qualSessions.POST("/reset-assignments", middleware.RequireActivePlan(db), handler.ResetSessionAssignments(db))
			qualSessions.POST("/swap-assignments", middleware.RequireActivePlan(db), handler.SwapTargetAssignments(db))
		}

		qualAssignments := api.Group("/qualification/assignments/:assignmentId")
		qualAssignments.Use(middleware.AuthMiddleware())
		{
			qualAssignments.GET("/scores", handler.GetQualificationAssignmentScores(db))
			qualAssignments.POST("/scores", middleware.RequireActivePlan(db), handler.UpdateQualificationScore(db))
			qualAssignments.DELETE("", middleware.RequireActivePlan(db), handler.DeleteQualificationAssignment(db))
		}

		// Target routes
		targets := api.Group("/targets")
		{
			targets.GET("", handler.GetTargets(db)) // ?phase=qualification&session_id=...
		}

		// Event Targets Data Master routes
		tournaments.GET("/:id/targets", handler.GetEventTargets(db))
		tournaments.GET("/:id/targets/options", handler.GetTargetOptions(db))
		tournaments.GET("/:id/targets/:target_id", handler.GetTargetDetails(db))

		// Archer personal target assignment (requires auth)
		tournaments.GET("/:id/my-target", middleware.AuthMiddleware(), handler.GetMyEventTarget(db))

		// Event category reference routes
		api.GET("/event-categories", handler.ListEventCategoryRefs(db))
		api.POST("/event-categories", handler.CreateEventCategoryRef(db))
		api.PUT("/event-categories/:id", handler.UpdateEventCategoryRef(db))

		// Archer routes
		archers := api.Group("/archers")
		{
			// Try to populate user context if token exists (for context-aware info in public endpoints)
			archers.Use(middleware.OptionalAuthMiddleware())

			// Public archer routes
			archers.GET("", middleware.RateLimit(60, 1*time.Minute), handler.GetArchers(db))
			archers.GET("/:id", handler.GetArcherByID(db))
			archers.GET("/:id/tournaments", handler.GetArcherEvents(db))
			archers.GET("/:id/events", handler.GetArcherEvents(db))
			archers.GET("/registration-profile/:uuid", handler.GetArcherRegistrationProfile(db))

			// Protected archer routes
			protected := archers.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/my/stats", handler.GetMyArcherStats(db))
				protected.GET("/me/stats", handler.GetMyArcherStats(db))
				protected.GET("/my/tournaments", handler.GetMyArcherEvents(db))
				protected.GET("/me/tournaments", handler.GetMyArcherEvents(db))
				protected.GET("/my/events", handler.GetMyArcherEvents(db))
				protected.GET("/me/events", handler.GetMyArcherEvents(db))
				protected.GET("/my/certificates", handler.GetArcherCertificates(db))
				protected.GET("/me/certificates", handler.GetArcherCertificates(db))
				protected.POST("", handler.CreateArcher(db))
				protected.PUT("/:id", handler.UpdateArcher(db))
				protected.DELETE("/:id", handler.DeleteArcher(db))
			}
		}

		// Certificate public & organizer routes
		api.GET("/certificates/:id/pdf", handler.GenerateCertificatePDF(db))
		api.GET("/certificates/verify/:cert_no", handler.VerifyCertificate(db))
		api.GET("/tournaments/:id/certificate-template", middleware.AuthMiddleware(), handler.GetCertificateTemplate(db))
		api.POST("/tournaments/:id/certificate-template", middleware.AuthMiddleware(), handler.SaveCertificateTemplate(db))

		// Reference data routes
		api.GET("/disciplines", handler.GetDisciplines(db))
		api.GET("/bow-types", handler.GetBowTypes(db))
		api.GET("/team-types", handler.GetEventTypes(db))
		api.GET("/gender-divisions", handler.GetGenderDivisions(db))
		api.GET("/age-groups", handler.GetAgeGroups(db))
		api.GET("/cities", handler.GetCities())
		api.POST("/contact", handler.SubmitContactMessage(db))

		// Blog & Newsletter routes
		api.POST("/newsletter/subscribe", handler.SubscribeNewsletter(db))
		
		blog := api.Group("/blog")
		{
			blog.GET("/articles", handler.ListBlogArticles(db))
			blog.GET("/articles/:slug", handler.GetBlogArticle(db))
			blog.GET("/articles/:slug/comments", handler.ListBlogComments(db))
			blog.POST("/articles/:slug/comments", middleware.OptionalAuthMiddleware(), handler.AddBlogComment(db))
			blog.GET("/:slug/comments", handler.ListBlogComments(db))
			blog.POST("/:slug/comments", middleware.OptionalAuthMiddleware(), handler.AddBlogComment(db))
			blog.POST("/:slug/views", handler.IncrementBlogArticleViews(db))
			blog.POST("/subscribe", handler.SubscribeNewsletter(db))
		}

		// Docs routes
		docs := api.Group("/docs")
		{
			docs.GET("", handler.ListDocs())
			// Use wildcard so docs can be nested by category, e.g. /docs/archer/archer-profile
			docs.GET("/*slug", handler.GetDocDetail())
		}

		docsComments := api.Group("/docs-comments")
		{
			docsComments.GET("/*slug", handler.ListDocsComments(db))
			docsComments.POST("/*slug", middleware.OptionalAuthMiddleware(), handler.AddDocsComment(db))
		}

		// Team Management
		teams := api.Group("/teams")
		{
			teams.GET("/event/:eventId", handler.GetTeams(db))
			teams.GET("/:teamId", handler.GetTeam(db))

			protectedTeams := teams.Group("")
			protectedTeams.Use(middleware.AuthMiddleware())
			{
				protectedTeams.GET("/my", handler.GetMyTeams(db))
				protectedTeams.POST("/event/:eventId", middleware.RequireActivePlan(db), handler.CreateTeam(db))
				protectedTeams.PUT("/:teamId", middleware.RequireActivePlan(db), handler.UpdateTeam(db))
				protectedTeams.DELETE("/:teamId", middleware.RequireActivePlan(db), handler.DeleteTeam(db))
				protectedTeams.POST("/event/:eventId/sync", middleware.RequireActivePlan(db), handler.SyncTeams(db))
			}

			teams.GET("/event/:eventId/rankings", handler.GetTeamRankings(db))
		}

		// Notification routes (protected)
		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware())
		{
			notifications.GET("", handler.GetNotifications(db))
			notifications.GET("/unread-count", handler.GetUnreadNotificationCount(db))
			notifications.PUT("/:id/read", handler.MarkNotificationAsRead(db))
			notifications.PUT("/read-all", handler.MarkAllNotificationsAsRead(db))
			notifications.DELETE("/:id", handler.DeleteNotification(db))
			notifications.DELETE("/clear-all", handler.DeleteAllNotifications(db))
			notifications.POST("", handler.CreateNotification(db))
		}

		// Payment & Registration routes
		payment := api.Group("/payment")
		{
			payment.GET("/channels", handler.GetPaymentChannels(db))
			payment.GET("/instruction", handler.GetPaymentInstruction(db))
			payment.GET("/status/:reference", handler.GetPaymentStatus(db))
			payment.GET("/invoice/:reference", handler.GenerateInvoicePDF(db))
			payment.POST("/create", middleware.AuthMiddleware(), handler.CreatePayment(db))
			payment.POST("/mayar/callback", handler.MayarWebhookCallback(db))
			payment.POST("/callback", handler.MayarWebhookCallback(db))
			payment.POST("/paypal/webhook", handler.PayPalWebhookCallback(db))
			payment.POST("/paypal/capture", handler.CapturePayPalPayment(db))
			payment.GET("/simulate-success/:reference", middleware.AuthMiddleware(), handler.SimulatePaymentSuccess(db))
			payment.GET("/my", middleware.AuthMiddleware(), handler.GetMyPayments(db))

			// Manual payment routes
			payment.POST("/manual/create", middleware.AuthMiddleware(), handler.CreateManualPayment(db))
			payment.POST("/manual/:reference/upload-proof", middleware.AuthMiddleware(), handler.UploadPaymentProof(db))
			payment.POST("/manual/:reference/verify", middleware.AuthMiddleware(), handler.VerifyManualPayment(db))
			payment.GET("/manual/pending", middleware.AuthMiddleware(), handler.GetPendingManualPayments(db))
		}

		// PayPal Dev Testing Routes
		dev := api.Group("/dev")
		{
			dev.GET("/paypal/check-token", handler.DevPayPalCheckToken)
			dev.POST("/paypal/create-order", handler.DevPayPalCreateOrder)
			dev.POST("/paypal/capture-order", handler.DevPayPalCaptureOrder)
			dev.GET("/paypal/order/:order_id", handler.DevPayPalGetOrder)
		}

		// Root/Admin routes
		root := api.Group("/root")
		{
			root.POST("/login", handler.RootLogin(db))

			protected := root.Group("/dashboard")
			protected.Use(middleware.AuthMiddleware(), middleware.RequireRole("root"))
			{
				// Account management
				protected.GET("/users", handler.GetAllUsers(db))
				protected.POST("/users", handler.RootCreateAccount(db))
				protected.PATCH("/users/:type/:uuid/terminate", handler.TerminateUser(db))

				// Subscription management
				protected.GET("/subscriptions", handler.GetAllSubscriptions(db))
				protected.PUT("/subscriptions/:type/:uuid", handler.UpdateUserSubscription(db))
				protected.POST("/subscriptions/:type/:uuid/addon", handler.AddSubscriptionAddon(db))

				// Plans
				protected.GET("/plans", handler.GetSubscriptionPlans(db))

				// Club management (data master)
				protected.GET("/clubs", handler.RootGetClubs(db))
				protected.POST("/clubs", handler.RootCreateClub(db))
				protected.PUT("/clubs/:id", handler.RootUpdateClub(db))
				protected.DELETE("/clubs/:id", handler.RootDeleteClub(db))
			}
		}

		// Mobile dedicated routes
		mobile := api.Group("/mobile")
		{
			mobile.GET("/hello", mobilehandler.MobileHello())

			// 1. Authentication (no auth required)
			auth := mobile.Group("/auth")
			{
				auth.POST("/scorekeeper/login", mobilehandler.MobileScorekeeperLogin(db))
				auth.POST("/scorekeeper/verify-code", mobilehandler.MobileVerifyScorekeeperCode(db))
				auth.POST("/archer/login", mobilehandler.MobileArcherLogin(db))
				auth.POST("/organizer/login", mobilehandler.MobileOrganizationLogin(db))
				auth.POST("/archer/register", mobilehandler.MobileArcherRegister(db))
				auth.POST("/google/login", mobilehandler.MobileGoogleLogin(db))
				auth.POST("/forgot-password", mobilehandler.MobileForgotPassword(db))
				auth.POST("/verify-otp", mobilehandler.MobileVerifyOTP(db))
				auth.POST("/reset-password", mobilehandler.MobileResetPassword(db))
				auth.POST("/logout", mobilehandler.MobileLogout(db))
				auth.POST("/google/bind", mobilehandler.MobileGoogleBind(db))
			}

			// 2. Events & Tournaments (public)
			mobile.GET("/tournaments", middleware.RateLimit(60, 1*time.Minute), mobilehandler.MobileListEvents(db))
			mobile.GET("/events", middleware.RateLimit(60, 1*time.Minute), mobilehandler.MobileListEvents(db))
			mobile.GET("/tournaments/history", middleware.RateLimit(60, 1*time.Minute), mobilehandler.MobileListEvents(db)) // Alias/Filter trigger
			mobile.GET("/events/history", middleware.RateLimit(60, 1*time.Minute), mobilehandler.MobileListEvents(db))
			mobile.GET("/tournaments/:slug", mobilehandler.MobileGetEventDetail(db))
			mobile.GET("/events/:slug", mobilehandler.MobileGetEventDetail(db))
			mobile.GET("/tournaments/:slug/participants", mobilehandler.MobileGetEventParticipants(db))
			mobile.GET("/events/:slug/participants", mobilehandler.MobileGetEventParticipants(db))
			mobile.GET("/tournaments/:slug/schedule", mobilehandler.MobileGetEventSchedule(db))
			mobile.GET("/events/:slug/schedule", mobilehandler.MobileGetEventSchedule(db))
			mobile.GET("/tournaments/:slug/categories", mobilehandler.MobileGetEventCategories(db))
			mobile.GET("/events/:slug/categories", mobilehandler.MobileGetEventCategories(db))
			mobile.GET("/tournaments/:slug/gallery", mobilehandler.MobileGetEventGallery(db))
			mobile.GET("/events/:slug/gallery", mobilehandler.MobileGetEventGallery(db))
			mobile.GET("/tournaments/:slug/faq", mobilehandler.MobileGetEventFAQ(db))
			mobile.GET("/events/:slug/faq", mobilehandler.MobileGetEventFAQ(db))
			mobile.GET("/tournaments/:slug/registration-fee", mobilehandler.MobileGetEventRegistrationFees(db))
			mobile.GET("/events/:slug/registration-fee", mobilehandler.MobileGetEventRegistrationFees(db))
			mobile.GET("/tournaments/:slug/rewards", mobilehandler.MobileGetEventRewards(db))
			mobile.GET("/events/:slug/rewards", mobilehandler.MobileGetEventRewards(db))
			mobile.GET("/tournaments/:slug/location", mobilehandler.MobileGetEventLocation(db))
			mobile.GET("/events/:slug/location", mobilehandler.MobileGetEventLocation(db))
			mobile.GET("/tournaments/:slug/results/qualification", handler.GetPublicQualificationResults(db))
			mobile.GET("/events/:slug/results/qualification", handler.GetPublicQualificationResults(db))
			mobile.GET("/tournaments/:slug/results/elimination", handler.GetPublicEliminationResults(db))
			mobile.GET("/events/:slug/results/elimination", handler.GetPublicEliminationResults(db))
			mobile.GET("/tournaments/:slug/results/files", handler.GetEventResultFiles(db))
			mobile.GET("/events/:slug/results/files", handler.GetEventResultFiles(db))

			mobile.GET("/clubs", mobilehandler.MobileListClubs(db))

			// 2d. Chatbot (public)
			chatbot := mobile.Group("/chatbot")
			{
				chatbot.GET("/intents", mobilehandler.MobileChatbotIntents())
				chatbot.POST("/message", mobilehandler.MobileChatbotMessage())
			}

			mobile.GET("/results/recent", mobilehandler.MobileRecentResults(db))
			mobile.GET("/payment/channels", handler.GetPaymentChannels(db))
			mobile.GET("/tournaments/:slug/payment-method", mobilehandler.MobileGetEventPaymentMethods(db))
			mobile.GET("/tournaments/payments/:reference/instructions", mobilehandler.MobileGetPaymentInstructions(db))

			// 3. Target scan by QR/barcode code (requires auth)
			mobileAuth := mobile.Group("")
			mobileAuth.Use(middleware.AuthMiddleware())
			{
				mobileAuth.GET("/scan", mobilehandler.MobileScanTarget(db))
				mobileAuth.GET("/sessions/boards", mobilehandler.MobileGetSessionBoards(db))
				mobileAuth.GET("/assignments/:assignmentId/detail", mobilehandler.MobileGetAssignmentScoreDetail(db))
			}

			// 4b. Archer account (requires auth)
			mobileArcher := mobile.Group("/archer")
			mobileArcher.Use(middleware.AuthMiddleware())
			{
				mobileArcher.GET("/me", mobilehandler.MobileGetArcherMe(db))
				mobileArcher.PUT("/me", mobilehandler.MobileUpdateArcherMe(db))
				mobileArcher.GET("/certificates", mobilehandler.MobileArcherGetCertificates(db))

				mobileArcher.GET("/payments/:reference", mobilehandler.MobileGetPaymentDetail(db))
				mobileArcher.GET("/tournaments", mobilehandler.MobileGetMyEvents(db))
				mobileArcher.GET("/events", mobilehandler.MobileGetMyEvents(db))
				mobileArcher.GET("/tournaments/payments", mobilehandler.MobileArcherGetEventPayments(db))
				mobileArcher.GET("/events/payments", mobilehandler.MobileArcherGetEventPayments(db))
				mobileArcher.GET("/tournaments/payments/:slug", mobilehandler.MobileArcherGetEventPaymentsByEvent(db))
				mobileArcher.GET("/events/payments/:slug", mobilehandler.MobileArcherGetEventPaymentsByEvent(db))
				mobileArcher.GET("/tournaments/:id/detail", mobilehandler.MobileArcherGetEventDetail(db))
				mobileArcher.GET("/events/:id/detail", mobilehandler.MobileArcherGetEventDetail(db))
				mobileArcher.GET("/tournaments/:id/performance", mobilehandler.MobileArcherGetEventPerformance(db))
				mobileArcher.GET("/events/:id/performance", mobilehandler.MobileArcherGetEventPerformance(db))
				mobileArcher.GET("/tournaments/:id/registration", mobilehandler.MobileGetMyRegistration(db))
				mobileArcher.GET("/events/:id/registration", mobilehandler.MobileGetMyRegistration(db))
				mobileArcher.GET("/tournaments/:id/qr", mobilehandler.MobileGetEventQRCode(db))
				mobileArcher.GET("/events/:id/qr", mobilehandler.MobileGetEventQRCode(db))
				mobileArcher.POST("/tournaments/register", mobilehandler.MobileRegisterEvent(db))
				mobileArcher.POST("/events/register", mobilehandler.MobileRegisterEvent(db))
				mobileArcher.DELETE("/tournaments/registrations/:registration_id", mobilehandler.MobileCancelRegistration(db))
				mobileArcher.DELETE("/events/registrations/:registration_id", mobilehandler.MobileCancelRegistration(db))
				mobileArcher.DELETE("/payments/:identifier/cancel", mobilehandler.MobileCancelPayment(db))
			}

			registerOrgRoutes := func(g *gin.RouterGroup) {
				g.GET("/me", mobilehandler.MobileGetOrganizationMe(db))
				g.PUT("/me", mobilehandler.MobileUpdateOrganizationMe(db))
				g.GET("/tournaments", mobilehandler.MobileGetOrganizationEvents(db))
				g.GET("/events", mobilehandler.MobileGetOrganizationEvents(db))
				g.GET("/tournaments/:id/participants", mobilehandler.MobileGetOrganizationEventParticipants(db))
				g.GET("/events/:id/participants", mobilehandler.MobileGetOrganizationEventParticipants(db))
				g.GET("/tournaments/:id/participants/:user_id", mobilehandler.MobileGetOrganizationParticipantDetail(db))
				g.GET("/events/:id/participants/:user_id", mobilehandler.MobileGetOrganizationParticipantDetail(db))
				g.DELETE("/tournaments/:id/participants/:user_id", mobilehandler.MobileOrganizationKickParticipant(db))
				g.DELETE("/events/:id/participants/:user_id", mobilehandler.MobileOrganizationKickParticipant(db))
				g.POST("/scan-registration", mobilehandler.MobileOrganizationScanRegistration(db))
				g.GET("/scan/history", mobilehandler.MobileGetScanHistory(db))
				g.GET("/dashboard", mobilehandler.MobileGetOrganizationDashboard(db))

				// Broadcasts
				g.GET("/tournaments/:id/broadcasts", mobilehandler.MobileGetEventBroadcasts(db))
				g.GET("/events/:id/broadcasts", mobilehandler.MobileGetEventBroadcasts(db))
				g.GET("/tournaments/:id/broadcasts/:broadcast_id", mobilehandler.MobileGetBroadcastDetail(db))
				g.GET("/events/:id/broadcasts/:broadcast_id", mobilehandler.MobileGetBroadcastDetail(db))
				g.POST("/tournaments/:id/broadcasts", mobilehandler.MobileCreateBroadcast(db))
				g.POST("/events/:id/broadcasts", mobilehandler.MobileCreateBroadcast(db))

				// Check-in & Search
				g.GET("/tournaments/:id/checkin-summary", mobilehandler.MobileGetCheckinSummary(db))
				g.GET("/events/:id/checkin-summary", mobilehandler.MobileGetCheckinSummary(db))
				g.POST("/tournaments/:id/participants/:participantId/manual-checkin", mobilehandler.MobileManualCheckin(db))
				g.POST("/events/:id/participants/:participantId/manual-checkin", mobilehandler.MobileManualCheckin(db))
				g.GET("/search-global", mobilehandler.MobileGlobalSearch(db))

				// Broadcast Extras
				g.POST("/tournaments/:id/broadcasts/reminder-unpaid", mobilehandler.MobileBroadcastReminderUnpaid(db))
				g.POST("/events/:id/broadcasts/reminder-unpaid", mobilehandler.MobileBroadcastReminderUnpaid(db))

				// Payments & Invoices
				g.GET("/tournaments/:id/payments", mobilehandler.MobileGetEventPayments(db))
				g.GET("/events/:id/payments", mobilehandler.MobileGetEventPayments(db))
				g.GET("/payments/:transactionId/invoice", mobilehandler.MobileGetInvoiceDetail(db))
				g.POST("/payments/:transactionId/manual-approve", mobilehandler.MobileManualApprovePayment(db))
				g.POST("/payments/:transactionId/refund", mobilehandler.MobileRefundPayment(db))

				// Notifications
				g.GET("/notifications", mobilehandler.MobileGetOrganizerNotifications(db))
				g.PUT("/notifications/mark-read", mobilehandler.MobileMarkAllOrganizerNotificationsRead(db))

				// Finance
				g.GET("/finance/earnings", mobilehandler.MobileGetOrganizationEarnings(db))
				g.GET("/finance/balance", mobilehandler.MobileGetOrganizationWallet(db))
				g.GET("/finance/bank-accounts", mobilehandler.MobileGetOrganizationBankAccounts(db))
				g.POST("/finance/bank-accounts", mobilehandler.MobileAddOrganizationBankAccount(db))
				g.PUT("/finance/bank-accounts/:id", mobilehandler.MobileUpdateOrganizationBankAccount(db))
				g.DELETE("/finance/bank-accounts/:id", mobilehandler.MobileDeleteOrganizationBankAccount(db))
				g.POST("/finance/withdraw", mobilehandler.MobileCreateWithdrawal(db))
				g.GET("/finance/withdrawals", handler.GetWithdrawals(db))
			}

			mobileOrganizer := mobile.Group("/organizer")
			mobileOrganizer.Use(middleware.AuthMiddleware())
			registerOrgRoutes(mobileOrganizer)

			mobileOrganization := mobile.Group("/organization")
			mobileOrganization.Use(middleware.AuthMiddleware())
			registerOrgRoutes(mobileOrganization)

			// 4. Qualification Scoring
			qual := mobile.Group("/qualification")
			qual.Use(middleware.AuthMiddleware())
			{
				qual.GET("/scoring/cards", handler.GetScoringCards(db))
				qual.GET("/scoring/targets", handler.GetScoringTargets(db))
				qual.POST("/scoring/scores/:assignmentId", handler.UpdateQualificationScore(db))
				qual.PUT("/scoring/arrow", mobilehandler.MobileEditArrowScoreAudit(db))
				qual.POST("/scoring/scores/:assignmentId/submit-final", mobilehandler.MobileSubmitFinalScoresheet(db))
			}

			// 5. Elimination Scoring
			elim := mobile.Group("/elimination")
			elim.Use(middleware.AuthMiddleware())
			{
				elim.GET("/scoring/cards", handler.GetScoringCards(db))
				elim.POST("/scoring/matches/:matchId/score", handler.UpdateMatchScore(db))
				elim.POST("/scoring/matches/:matchId/finish", handler.FinishMatch(db))
				elim.POST("/scoring/matches/:matchId/end", handler.EndMatch(db))
				elim.POST("/scoring/matches/:matchId/reset", handler.ResetMatch(db))
			}

			// 6. Scorekeeper dedicated endpoints
			sk := mobile.Group("/scorekeeper")
			sk.Use(middleware.AuthMiddleware())
			{
				sk.GET("/me", mobilehandler.MobileGetScorekeeperMe(db))
				sk.GET("/tournaments", mobilehandler.MobileGetScorekeeperEvents(db))
				sk.GET("/events", mobilehandler.MobileGetScorekeeperEvents(db))
				sk.POST("/verify-code", mobilehandler.MobileVerifyScorekeeperCode(db))
				sk.GET("/recent-scans", mobilehandler.MobileGetScorekeeperRecentScans(db))
				sk.GET("/history", mobilehandler.MobileGetScorekeeperHistory(db))
			}

			// 7. Options (Public)
			options := mobile.Group("/options")
			{
				options.GET("/clubs", mobilehandler.GetClubOptions(db))
				options.GET("/organizers", mobilehandler.GetOrganizationOptions(db))
				options.GET("/disciplines", mobilehandler.GetDisciplineOptions(db))
				options.GET("/bow-types", mobilehandler.GetBowTypeOptions(db))
				options.GET("/age-groups", mobilehandler.GetAgeGroupOptions(db))
				options.GET("/gender-divisions", mobilehandler.GetGenderDivisionOptions(db))
				options.GET("/cities", mobilehandler.GetCityOptions())
				options.GET("/event-types", mobilehandler.GetEventTypeOptions(db))
				options.GET("/banks", mobilehandler.MobileGetBankOptions(db))
			}

			// 8. Media (requires auth)
			mobileMedia := mobile.Group("/media")
			mobileMedia.Use(middleware.AuthMiddleware())
			{
				mobileMedia.POST("/upload", mobilehandler.MobileUploadMedia(db))
				mobileMedia.GET("", handler.ListMedia(db))
				mobileMedia.DELETE("/:id", handler.DeleteMedia(db))
			}

			// 9. Notifications (requires auth)
			mobileNotification := mobile.Group("/notifications")
			mobileNotification.Use(middleware.AuthMiddleware())
			{
				mobileNotification.GET("", mobilehandler.MobileGetNotifications(db))
				mobileNotification.PUT("/:id/read", mobilehandler.MobileMarkNotificationRead(db))
				mobileNotification.PUT("/read-all", mobilehandler.MobileMarkAllNotificationsRead(db))
				mobileNotification.GET("/unread-count", mobilehandler.MobileGetUnreadCount(db))
			}

		}

		// Organizer routes
		orgs := api.Group("/organizers")
		{
			// Public organizer routes
			orgs.GET("", handler.GetOrganizations(db))
			orgs.GET("/:slug", handler.GetOrganizationBySlug(db))

			// Protected organizer routes
			protectedOrgs := orgs.Group("")
			protectedOrgs.Use(middleware.AuthMiddleware())
			{
				protectedOrgs.GET("/me", handler.GetOrganizationProfile(db))
				protectedOrgs.PUT("/me", handler.UpdateOrganizationProfile(db))
				protectedOrgs.GET("/stats", handler.GetOrganizationDashboardStats(db))

				// Quota management
				protectedOrgs.GET("/me/quota", handler.GetMyQuota(db))
				protectedOrgs.GET("/me/quota/history", handler.GetQuotaHistory(db))
				protectedOrgs.POST("/me/quota/purchase", handler.PurchaseQuota(db))

				// Reports
				reports := protectedOrgs.Group("/reports")
				{
					reports.GET("/participants", handler.GetOrganizationParticipantsReport(db))
					reports.GET("/finance", handler.GetOrganizationFinanceReport(db))
					reports.GET("/performance", handler.GetOrganizationPerformanceReport(db))
					reports.GET("/attendance", handler.GetOrganizationAttendanceReport(db))
				}

				// Scorekeeper management
				scorekeepers := protectedOrgs.Group("/scorekeepers")
				{
					scorekeepers.GET("", handler.GetOrganizationScorekeepers(db))
					scorekeepers.POST("", middleware.RequireActivePlan(db), handler.CreateScorekeeper(db))
					scorekeepers.PUT("/:id", middleware.RequireActivePlan(db), handler.UpdateScorekeeper(db))
					scorekeepers.DELETE("/:id", middleware.RequireActivePlan(db), handler.DeleteScorekeeper(db))
					scorekeepers.POST("/:id/regenerate", middleware.RequireActivePlan(db), handler.RegenerateScorekeeperCode(db))
				}

				// Bank accounts
				bankAccounts := protectedOrgs.Group("/bank-accounts")
				{
					bankAccounts.GET("", handler.GetBankAccounts(db))
					bankAccounts.POST("", middleware.RequireActivePlan(db), handler.CreateBankAccount(db))
					bankAccounts.PUT("", middleware.RequireActivePlan(db), handler.SyncBankAccounts(db))
					bankAccounts.PUT("/:id", middleware.RequireActivePlan(db), handler.UpdateBankAccount(db))
					bankAccounts.DELETE("/:id", middleware.RequireActivePlan(db), handler.DeleteBankAccount(db))
				}

				// Payment methods
				paymentMethods := protectedOrgs.Group("/payment-methods")
				{
					paymentMethods.GET("", handler.GetOrganizationPaymentMethods(db))
					paymentMethods.POST("", middleware.RequireActivePlan(db), handler.CreateOrganizationPaymentMethod(db))
					paymentMethods.PUT("", middleware.RequireActivePlan(db), handler.SyncOrganizationPaymentMethods(db))
					paymentMethods.PUT("/:id", middleware.RequireActivePlan(db), handler.UpdateOrganizationPaymentMethod(db))
					paymentMethods.DELETE("/:id", middleware.RequireActivePlan(db), handler.DeleteOrganizationPaymentMethod(db))
				}

				// Wallet & Withdrawals
				wallet := protectedOrgs.Group("/wallet")
				{
					wallet.GET("", handler.GetMyWallet(db))
					wallet.GET("/withdrawals", handler.GetWithdrawals(db))
					wallet.POST("/withdrawals", middleware.RequireActivePlan(db), handler.CreateWithdrawal(db))
					wallet.GET("/mutations", handler.GetWalletMutations(db))
				}

				// Earnings
				protectedOrgs.GET("/earnings", handler.GetOrganizationEarningsSummary(db))
				protectedOrgs.GET("/earnings/:id", handler.GetOrganizationEarningsDetail(db))
			}
		}

		// Club routes (Data Master)
		cbr := api.Group("/clubs")
		{
			cbr.GET("", handler.GetClubs(db))
			cbr.GET("/:id", handler.GetClubByID(db))
		}

		// Media routes
		media := api.Group("/media")
		{
			// Public media access
			media.GET("/:filename", handler.GetMedia(db))
			r.GET("/api/v1/media/:filename", handler.GetMedia(db))

			// Protected media routes
			protectedMedia := media.Group("")
			protectedMedia.Use(middleware.AuthMiddleware())
			{
				protectedMedia.POST("/upload", handler.UploadMedia(db))
				protectedMedia.GET("", handler.ListMedia(db))
				protectedMedia.DELETE("/:id", handler.DeleteMedia(db))
			}
		}

		// Download endpoint moved outside `/media/*filepath` wildcard.
		api.GET("/media-download/:filename", handler.DownloadMedia())

		// Discovery routes
		discovery := api.Group("/discovery")
		{
			discovery.GET("/sitemap", handler.GetSitemapData(db))
		}

		// Event registration is handled via POST /tournaments/:id/participants

		// Get host and port from environment
		host := os.Getenv("HOST")
		if host == "" {
			host = "0.0.0.0"
		}
		port := os.Getenv("PORT")
		if port == "" {
			port = "8001"
		}

		logger.Infof("Server listening on %s:%s", host, port)
		logger.Fatal(r.Run(host + ":" + port))
	}
}
