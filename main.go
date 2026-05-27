package main

import (
	"net/http"
	"os"

	"Archeris-api/database"
	"Archeris-api/handler"
	mobilehandler "Archeris-api/handler/mobile"
	"Archeris-api/middleware"

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
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowedOrigins := []string{
			"https://archeris.net",
			"http://localhost:9000",
			"http://localhost:3003",
			"http://localhost:3000",
			"http://127.0.0.1:9000",
			"http://127.0.0.1:3003",
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
			"message": "Welcome to Archeris.net API - TEST DEPLOY 1",
			"status":  "running",
		})
	})

	// Media is served via handlers under /media (see routes below).

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

		// Session endpoint (Global)
		api.GET("/chatbot/intents", handler.ChatbotIntents())
		api.POST("/chatbot/message", handler.ChatbotMessage())

		api.GET("/archer/me", middleware.AuthMiddleware(), handler.GetArcherProfile(db))
		api.GET("/organization/me", middleware.AuthMiddleware(), handler.GetOrganizationProfile(db))
		api.GET("/seller/me", middleware.AuthMiddleware(), handler.GetSellerProfile(db))

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
			// Traditional auth
			auth.POST("/register", handler.Register(db))
			auth.POST("/login", handler.Login(db))
			auth.POST("/logout", handler.Logout())
			auth.GET("/check-name", handler.CheckNameExists(db))
			auth.GET("/check-username", handler.CheckUsernameExists(db))

			// Google OAuth
			auth.GET("/google", middleware.OptionalAuthMiddleware(), handler.InitiateGoogleAuth(db))
			auth.GET("/google/callback", middleware.OptionalAuthMiddleware(), handler.GoogleCallback(db))
			auth.POST("/google/callback", middleware.OptionalAuthMiddleware(), handler.GoogleCallback(db))

			auth.GET("/avatar/:identifier", handler.GetArcherProfileImage(db))

			// Forgot / Reset password (public Ã¢â‚¬â€ no auth required)
			auth.POST("/forgot-password", handler.ForgotPassword(db))
			auth.POST("/verify-reset-otp", handler.VerifyResetOTP(db))
			auth.POST("/reset-password", handler.ResetPassword(db))
			auth.POST("/change-password-otp", handler.ChangePasswordWithOTP(db))

			// Alias for mobile login to satisfy public URL expectations
			auth.POST("/archer/login", mobilehandler.MobileArcherLogin(db))
		}

		// Public Subscription routes
		api.GET("/public/subscription/comparison", handler.GetSubscriptionComparison())

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
		events := api.Group("/events")
		events.Use(middleware.OptionalAuthMiddleware())
		{
			// Public Event routes
			events.GET("", handler.GetEvents(db))
			events.GET("/:id", handler.GetEventByID(db))
			events.GET("/:id/categories", handler.GetEventEvents(db))
			events.GET("/:id/participants", handler.GetEventParticipants(db))
			events.GET("/:id/participants/:participantId", handler.GetEventParticipant(db))
			events.GET("/:id/participants/me", middleware.AuthMiddleware(), handler.GetMyEventRegistration(db))
			events.DELETE("/:id/participants/me", middleware.AuthMiddleware(), handler.UnregisterFromEvent(db))
			events.PUT("/:id/participants/:participantId", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateEventParticipant(db))
			events.DELETE("/:id/participants/:participantId", middleware.AuthMiddleware(), handler.DeleteEventParticipant(db))
			events.DELETE("/participants/:participantId", middleware.AuthMiddleware(), handler.CancelParticipantRegistration(db))
			events.POST("/participants/:participantId/payment", middleware.AuthMiddleware(), handler.CreateParticipantPayment(db))
			events.GET("/:id/teams", handler.GetEventTeams(db))
			events.GET("/:id/images", handler.GetEventImages(db))
			events.GET("/:id/schedule", handler.GetEventSchedule(db))
			events.GET("/:id/target-names", handler.GetTargetNames(db))
			events.GET("/:id/payment-methods", handler.GetEventPaymentMethods(db))
			events.GET("/:id/payments", handler.GetEventPayments(db))
			events.POST("/participants/reregister", handler.ReregisterParticipant(db))
			events.GET("/:id/participants/printout", handler.GetEventParticipantList(db))

			// Public Results endpoints
			events.GET("/:id/results/qualification", handler.GetPublicQualificationResults(db))
			events.GET("/:id/results/elimination", handler.GetPublicEliminationResults(db))

			// Protected Event routes (require authentication)
			protected := events.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/my", handler.GetMyEvents(db))
				protected.POST("", middleware.RequireActivePlan(db), handler.CreateEvent(db))
				protected.PUT("/:id", middleware.RequireActivePlan(db), handler.UpdateEvent(db))
				protected.DELETE("/:id", middleware.RequireActivePlan(db), handler.DeleteEvent(db))
				protected.POST("/:id/publish", middleware.RequireActivePlan(db), handler.PublishEvent(db))
				protected.POST("/:id/reset", middleware.RequireActivePlan(db), handler.ResetEventData(db))
				protected.POST("/:id/reset/request-code", middleware.RequireActivePlan(db), handler.RequestResetCode(db))
				protected.GET("/:id/participants/export", middleware.RequireActivePlan(db), handler.ExportParticipantsCSV(db))
				protected.POST("/:id/categories", middleware.RequireActivePlan(db), handler.CreateEventCategory(db))
				protected.POST("/:id/categories/batch", middleware.RequireActivePlan(db), handler.CreateEventCategories(db))
				protected.GET("/:id/categories/:categoryId", handler.GetEventCategoryDetails(db))
				protected.PUT("/:id/categories/:categoryId", middleware.RequireActivePlan(db), handler.UpdateEventCategory(db))
				protected.DELETE("/:id/categories/:categoryId", middleware.RequireActivePlan(db), handler.DeleteEventCategory(db))
				protected.POST("/:id/participants", middleware.RequireActivePlan(db), handler.RegisterParticipant(db))
				protected.POST("/:id/participants/batch", middleware.RequireActivePlan(db), handler.BatchRegisterParticipants(db))
				protected.PUT("/:id/images", middleware.RequireActivePlan(db), handler.UpdateEventImages(db))
				protected.PUT("/:id/schedule", middleware.RequireActivePlan(db), handler.UpdateEventSchedule(db))
				protected.POST("/:id/payment-methods", middleware.RequireActivePlan(db), handler.CreateEventPaymentMethod(db))
				protected.PUT("/:id/payment-methods/:methodId", middleware.RequireActivePlan(db), handler.UpdateEventPaymentMethod(db))
				protected.DELETE("/:id/payment-methods/:methodId", middleware.RequireActivePlan(db), handler.DeleteEventPaymentMethod(db))

				// Qualification target assignments - nested under events/:id/qualification/sessions/:sessionId
				protected.POST("/:id/qualification/sessions/:sessionId/assignments", middleware.RequireActivePlan(db), handler.CreateBulkTargetAssignments(db))

				// Target management
				protected.POST("/:id/targets", middleware.RequireActivePlan(db), handler.CreateEventTarget(db))
				protected.PUT("/:id/targets/batch", middleware.RequireActivePlan(db), handler.BatchUpdateTargets(db))
				protected.PUT("/:id/targets/:target_id", middleware.RequireActivePlan(db), handler.UpdateEventTarget(db))
				protected.DELETE("/:id/targets/:target_id", middleware.RequireActivePlan(db), handler.DeleteEventTarget(db))
			}
		}

		// Qualification routes (event-level sessions)
		qualification := api.Group("/events/:id/qualification")
		qualification.Use(middleware.AuthMiddleware())
		{
			qualification.GET("/sessions", handler.GetQualificationSessions(db))
			qualification.POST("/sessions", middleware.RequireActivePlan(db), handler.CreateQualificationSession(db))
			qualification.PATCH("/sessions/:sessionId", middleware.RequireActivePlan(db), handler.UpdateQualificationSession(db))
			qualification.DELETE("/sessions/:sessionId", middleware.RequireActivePlan(db), handler.DeleteQualificationSession(db))
			qualification.GET("/leaderboard", handler.GetQualificationLeaderboard(db))
			qualification.GET("/sessions/:sessionCode/scoresheet", handler.GetQualificationScoresheet(db))
		}

		// Elimination routes (event-level brackets)
		elimination := api.Group("/events/:id/elimination")
		elimination.Use(middleware.OptionalAuthMiddleware())
		{
			elimination.GET("/brackets", handler.GetBrackets(db))
			elimination.GET("/bracket-size", middleware.AuthMiddleware(), handler.GetBracketSizeRecommendation(db))
			elimination.POST("/brackets", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.CreateBracket(db))
			elimination.GET("/brackets/:bracketId", handler.GetBracket(db))
			elimination.PUT("/brackets/:bracketId", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.UpdateBracket(db))
			elimination.DELETE("/brackets/:bracketId", middleware.AuthMiddleware(), middleware.RequireActivePlan(db), handler.DeleteBracket(db))
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
		events.GET("/:id/targets", handler.GetEventTargets(db))
		events.GET("/:id/targets/options", handler.GetTargetOptions(db))
		events.GET("/:id/targets/:target_id", handler.GetTargetDetails(db))

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
			archers.GET("", handler.GetArchers(db))
			archers.GET("/:id", handler.GetArcherByID(db))
			archers.GET("/:id/events", handler.GetArcherEvents(db))
			archers.GET("/registration-profile/:uuid", handler.GetArcherRegistrationProfile(db))

			// Protected archer routes
			protected := archers.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/me/stats", handler.GetMyArcherStats(db))
				protected.GET("/my/events", handler.GetMyArcherEvents(db))
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
		api.POST("/contact", handler.SubmitContactMessage(db))

		// News routes
		news := api.Group("/news")
		{
			// Public news routes
			news.GET("", handler.GetNewsPublic(db))
			news.GET("/:id", handler.GetNewsByID(db))
			news.POST("/:id/views", handler.IncrementNewsViews(db))
			news.POST("/subscribe", handler.SubscribeNews(db))
			news.GET("/:id/comments", mobilehandler.MobileListNewsComments(db))
			news.POST("/:id/comments", middleware.OptionalAuthMiddleware(), mobilehandler.MobileAddNewsComment(db))
			news.GET("/:id/related", mobilehandler.MobileListRelatedNews(db))

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

		// Blog routes
		blog := api.Group("/blog")
		{
			blog.GET("/:slug/comments", handler.ListBlogComments(db))
			blog.POST("/:slug/comments", middleware.OptionalAuthMiddleware(), handler.AddBlogComment(db))
			blog.POST("/:slug/views", handler.IncrementBlogArticleViews(db))
		}

		// Docs routes
		docs := api.Group("/docs")
		{
			docs.GET("/:slug/comments", handler.ListDocsComments(db))
			docs.POST("/:slug/comments", middleware.OptionalAuthMiddleware(), handler.AddDocsComment(db))
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

		// Chat routes (archer Ã¢â€ â€ seller)
		chat := api.Group("/chat")
		chat.Use(middleware.AuthMiddleware())
		{
			chat.POST("/conversations", handler.StartOrGetConversation(db))
			chat.GET("/conversations", handler.ListConversations(db))
			chat.GET("/conversations/:id/messages", handler.GetConversationMessages(db))
			chat.POST("/conversations/:id/messages", handler.SendMessage(db))
			chat.GET("/unread", handler.GetChatUnreadCount(db))
		}

		// Payment & Registration routes
		payment := api.Group("/payment")
		{
			payment.GET("/channels", handler.GetPaymentChannels(db))
			payment.GET("/instruction", handler.GetPaymentInstruction(db))
			payment.GET("/status/:reference", handler.GetPaymentStatus(db))
			payment.GET("/invoice/:reference", handler.GenerateInvoicePDF(db))
			payment.POST("/create", middleware.AuthMiddleware(), handler.CreatePayment(db))
			payment.POST("/tripay/callback", handler.PaymentCallback(db))
			payment.POST("/paddle/initiate", middleware.AuthMiddleware(), handler.InitiatePaddlePayment(db))
			payment.POST("/paddle/callback", handler.PaddleWebhookCallback(db))
			payment.GET("/simulate-success/:reference", handler.SimulatePaymentSuccess(db))
			payment.GET("/my", middleware.AuthMiddleware(), handler.GetMyPayments(db))

			// Manual payment routes
			payment.POST("/manual/create", middleware.AuthMiddleware(), handler.CreateManualPayment(db))
			payment.POST("/manual/:reference/upload-proof", middleware.AuthMiddleware(), handler.UploadPaymentProof(db))
			payment.POST("/manual/:reference/verify", middleware.AuthMiddleware(), handler.VerifyManualPayment(db))
			payment.GET("/manual/pending", middleware.AuthMiddleware(), handler.GetPendingManualPayments(db))
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
				auth.POST("/archer/login", mobilehandler.MobileArcherLogin(db))
				auth.POST("/organization/login", mobilehandler.MobileOrganizationLogin(db))
				auth.POST("/seller/login", mobilehandler.MobileSellerLogin(db))
				auth.POST("/archer/register", mobilehandler.MobileArcherRegister(db))
				auth.POST("/seller/register", mobilehandler.MobileSellerRegister(db))
				auth.POST("/google/login", mobilehandler.MobileGoogleLogin(db))
				auth.POST("/forgot-password", mobilehandler.MobileForgotPassword(db))
				auth.POST("/verify-otp", mobilehandler.MobileVerifyOTP(db))
				auth.POST("/reset-password", mobilehandler.MobileResetPassword(db))
				auth.POST("/logout", mobilehandler.MobileLogout(db))
				auth.POST("/google/bind", mobilehandler.MobileGoogleBind(db))
			}

			// 2. Events (public)
			mobile.GET("/events", mobilehandler.MobileListEvents(db))
			mobile.GET("/events/history", mobilehandler.MobileListEvents(db)) // Alias/Filter trigger
			mobile.GET("/events/:slug", mobilehandler.MobileGetEventDetail(db))
			mobile.GET("/events/:slug/participants", mobilehandler.MobileGetEventParticipants(db))
			mobile.GET("/events/:slug/schedule", mobilehandler.MobileGetEventSchedule(db))
			mobile.GET("/events/:slug/categories", mobilehandler.MobileGetEventCategories(db))
			mobile.GET("/events/:slug/gallery", mobilehandler.MobileGetEventGallery(db))
			mobile.GET("/events/:slug/faq", mobilehandler.MobileGetEventFAQ(db))
			mobile.GET("/events/:slug/registration-fee", mobilehandler.MobileGetEventRegistrationFees(db))
			mobile.GET("/events/:slug/rewards", mobilehandler.MobileGetEventRewards(db))
			mobile.GET("/events/:slug/location", mobilehandler.MobileGetEventLocation(db))
			mobile.GET("/events/:slug/results/qualification", handler.GetPublicQualificationResults(db))
			mobile.GET("/events/:slug/results/elimination", handler.GetPublicEliminationResults(db))
			mobile.GET("/events/:slug/results/files", handler.GetEventResultFiles(db))

			// 2b. News (public read-only)
			mobile.GET("/news", mobilehandler.MobileListNews(db))
			mobile.GET("/news/:id", mobilehandler.MobileGetNewsDetail(db))
			mobile.GET("/news/:id/comments", mobilehandler.MobileListNewsComments(db))
			mobile.POST("/news/:id/comments", middleware.OptionalAuthMiddleware(), mobilehandler.MobileAddNewsComment(db))
			mobile.GET("/news/:id/related", mobilehandler.MobileListRelatedNews(db))

			// 2d. Chatbot (public)
			chatbot := mobile.Group("/chatbot")
			{
				chatbot.GET("/intents", mobilehandler.MobileChatbotIntents())
				chatbot.POST("/message", mobilehandler.MobileChatbotMessage())
			}

			// 2c. Marketplace (public read-only)
			mobile.GET("/marketplace/products", mobilehandler.MobileMarketplaceListProducts(db))
			mobile.GET("/marketplace/products/:id", mobilehandler.MobileMarketplaceGetProductDetail(db))
			mobile.GET("/payment/channels", handler.GetPaymentChannels(db))
			mobile.GET("/events/:slug/payment-method", mobilehandler.MobileGetEventPaymentMethods(db))
			mobile.GET("/events/payments/:reference/instructions", mobilehandler.MobileGetPaymentInstructions(db))

			// 3. Target scan by QR/barcode code (requires auth)
			mobileAuth := mobile.Group("")
			mobileAuth.Use(middleware.AuthMiddleware())
			{
				mobileAuth.GET("/scan", mobilehandler.MobileScanTarget(db))
				mobileAuth.GET("/sessions/boards", mobilehandler.MobileGetSessionBoards(db))
				mobileAuth.GET("/assignments/:assignmentId/detail", mobilehandler.MobileGetAssignmentScoreDetail(db))
			}

			// 3b. Chat archer <-> seller (requires auth)
			mobileChat := mobile.Group("/chat")
			mobileChat.Use(middleware.AuthMiddleware())
			{
				mobileChat.POST("/conversations", mobilehandler.MobileStartOrGetConversation(db))
				mobileChat.GET("/conversations", mobilehandler.MobileListConversations(db))
				mobileChat.GET("/conversations/:id/messages", mobilehandler.MobileGetConversationMessages(db))
				mobileChat.POST("/conversations/:id/messages", mobilehandler.MobileSendMessage(db))
				mobileChat.GET("/unread", mobilehandler.MobileGetChatUnreadCount(db))
				mobileChat.GET("/last-active", mobilehandler.MobileGetPeerLastActive(db))
			}

			// 4b. Archer account (requires auth)
			mobileArcher := mobile.Group("/archer")
			mobileArcher.Use(middleware.AuthMiddleware())
			{
				mobileArcher.GET("/me", mobilehandler.MobileGetArcherMe(db))
				mobileArcher.PUT("/me", mobilehandler.MobileUpdateArcherMe(db))
				mobileArcher.GET("/cart", mobilehandler.MobileArcherGetCart(db))
				mobileArcher.POST("/cart", mobilehandler.MobileArcherAddToCart(db))
				mobileArcher.PUT("/cart/:id", mobilehandler.MobileArcherUpdateCartItem(db))
				mobileArcher.DELETE("/cart/:id", mobilehandler.MobileArcherRemoveFromCart(db))
				mobileArcher.DELETE("/cart", mobilehandler.MobileArcherClearCart(db))
				mobileArcher.POST("/cart/checkout", mobilehandler.MobileArcherCheckoutCart(db))
				mobileArcher.GET("/orders", mobilehandler.MobileArcherGetOrderHistory(db))

				mobileArcher.GET("/payments/:reference", mobilehandler.MobileGetPaymentDetail(db))
				mobileArcher.GET("/events", mobilehandler.MobileGetMyEvents(db))
				mobileArcher.GET("/events/payments", mobilehandler.MobileArcherGetEventPayments(db))
				mobileArcher.GET("/events/payments/:slug", mobilehandler.MobileArcherGetEventPaymentsByEvent(db))
				mobileArcher.GET("/events/:id/detail", mobilehandler.MobileArcherGetEventDetail(db))
				mobileArcher.GET("/events/:id/registration", mobilehandler.MobileGetMyRegistration(db))
				mobileArcher.GET("/events/:id/qr", mobilehandler.MobileGetEventQRCode(db))
				mobileArcher.POST("/events/register", mobilehandler.MobileRegisterEvent(db))
			}

			mobileOrganization := mobile.Group("/organization")
			mobileOrganization.Use(middleware.AuthMiddleware())
			{
				mobileOrganization.GET("/me", mobilehandler.MobileGetOrganizationMe(db))
				mobileOrganization.PUT("/me", mobilehandler.MobileUpdateOrganizationMe(db))
				mobileOrganization.GET("/events", mobilehandler.MobileGetOrganizationEvents(db))
				mobileOrganization.GET("/events/:id/participants", mobilehandler.MobileGetOrganizationEventParticipants(db))
				mobileOrganization.GET("/events/:id/participants/:user_id", mobilehandler.MobileGetOrganizationParticipantDetail(db))
				mobileOrganization.DELETE("/events/:id/participants/:user_id", mobilehandler.MobileOrganizationKickParticipant(db))
				mobileOrganization.POST("/scan-registration", mobilehandler.MobileOrganizationScanRegistration(db))
				mobileOrganization.GET("/dashboard", mobilehandler.MobileGetOrganizationDashboard(db))

				// Finance
				mobileOrganization.GET("/finance/earnings", mobilehandler.MobileGetOrganizationEarnings(db))
				mobileOrganization.GET("/finance/balance", mobilehandler.MobileGetOrganizationWallet(db))
				mobileOrganization.GET("/finance/bank-accounts", mobilehandler.MobileGetOrganizationBankAccounts(db))
				mobileOrganization.POST("/finance/bank-accounts", mobilehandler.MobileAddOrganizationBankAccount(db))
				mobileOrganization.PUT("/finance/bank-accounts/:id", mobilehandler.MobileUpdateOrganizationBankAccount(db))
				mobileOrganization.DELETE("/finance/bank-accounts/:id", mobilehandler.MobileDeleteOrganizationBankAccount(db))
			}

			mobileSeller := mobile.Group("/seller")
			mobileSeller.Use(middleware.AuthMiddleware())
			{
				mobileSeller.GET("/me", mobilehandler.MobileGetSellerMe(db))
				mobileSeller.PUT("/me", mobilehandler.MobileUpdateSellerMe(db))
				mobileSeller.PUT("/me/page", mobilehandler.MobileUpdateSellerPage(db))
				mobileSeller.GET("/products", mobilehandler.MobileGetSellerProducts(db))
				mobileSeller.POST("/products", mobilehandler.MobileCreateProduct(db))
				mobileSeller.PUT("/products/:id", mobilehandler.MobileUpdateProduct(db))
				mobileSeller.DELETE("/products/:id", mobilehandler.MobileDeleteProduct(db))
				mobileSeller.GET("/dashboard", mobilehandler.MobileGetSellerDashboard(db))
			}

			// 4. Qualification Scoring
			qual := mobile.Group("/qualification")
			qual.Use(middleware.AuthMiddleware())
			{
				qual.GET("/scoring/cards", handler.GetScoringCards(db))
				qual.GET("/scoring/targets", handler.GetScoringTargets(db))
				qual.POST("/scoring/scores/:assignmentId", handler.UpdateQualificationScore(db))
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
				sk.GET("/events", mobilehandler.MobileGetScorekeeperEvents(db))
			}

			// 7. Options (Public)
			options := mobile.Group("/options")
			{
				options.GET("/clubs", mobilehandler.GetClubOptions(db))
				options.GET("/organizations", mobilehandler.GetOrganizationOptions(db))
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

		}

		// Organization routes
		orgs := api.Group("/organizations")
		{
			// Public organization routes
			orgs.GET("", handler.GetOrganizations(db))
			orgs.GET("/:slug", handler.GetOrganizationBySlug(db))

			// Protected organization routes
			protectedOrgs := orgs.Group("")
			protectedOrgs.Use(middleware.AuthMiddleware())
			{
				protectedOrgs.GET("/me", handler.GetOrganizationProfile(db))
				protectedOrgs.PUT("/me", handler.UpdateOrganizationProfile(db))
				protectedOrgs.GET("/stats", handler.GetOrganizationDashboardStats(db))

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

		// Product routes Ã¢â‚¬â€ /my must be registered BEFORE /:id to avoid wildcard conflict
		products := api.Group("/products")
		{
			products.GET("", handler.GetProducts(db))

			// Protected product routes (specific paths before wildcard)
			protectedProducts := products.Group("")
			protectedProducts.Use(middleware.AuthMiddleware())
			{
				protectedProducts.GET("/my", handler.GetMyProducts(db))
				protectedProducts.POST("", handler.CreateProduct(db))
			}

			// Wildcard routes - must come after static ones
			products.GET("/:id", handler.GetProductByID(db))
			products.POST("/:id/views", handler.IncrementProductViews(db))
			products.PUT("/:id", middleware.AuthMiddleware(), handler.UpdateProduct(db))
			products.DELETE("/:id", middleware.AuthMiddleware(), handler.DeleteProduct(db))
		}

		// Cart routes
		cart := api.Group("/cart")
		cart.Use(middleware.AuthMiddleware()) // Archers only
		{
			cart.GET("", handler.GetCart(db))
			cart.POST("", handler.AddToCart(db))
			cart.PUT("/:id", handler.UpdateCartItem(db))
			cart.DELETE("/:id", handler.DeleteCartItem(db))
			cart.POST("/checkout", handler.CheckoutCart(db))
		}

		// Seller routes (protected)
		sellersProtected := api.Group("/sellers")
		sellersProtected.Use(middleware.AuthMiddleware())
		{
			sellersProtected.GET("/me", handler.GetSellerProfile(db))
			sellersProtected.GET("/profile", handler.GetSellerProfile(db))
			sellersProtected.PUT("/me", handler.UpdateSellerProfileBasic(db))
			sellersProtected.PUT("/profile", handler.UpdateSellerProfileBasic(db))
			sellersProtected.PUT("/me/page", handler.UpdateSellerProfile(db))

			// Seller finance
			sellersProtected.GET("/bank-accounts", handler.GetBankAccounts(db))
			sellersProtected.POST("/bank-accounts", handler.CreateBankAccount(db))
			sellersProtected.PUT("/bank-accounts/:id", handler.UpdateBankAccount(db))
			sellersProtected.DELETE("/bank-accounts/:id", handler.DeleteBankAccount(db))

			sellersProtected.GET("/wallet", handler.GetMyWallet(db))
			sellersProtected.GET("/wallet/withdrawals", handler.GetWithdrawals(db))
			sellersProtected.POST("/wallet/withdrawals", handler.CreateWithdrawal(db))
		}

		// Order routes (seller) Ã¢â‚¬â€ also accessible as /api/v1/orders
		ordersGroup := api.Group("/orders")
		ordersGroup.Use(middleware.AuthMiddleware())
		{
			ordersGroup.GET("", handler.GetSellerOrders(db))
			ordersGroup.GET("/:id", handler.GetSellerOrderByID(db))
			ordersGroup.GET("/export", handler.ExportSellerOrders(db))
			ordersGroup.GET("/stats", handler.GetSellerStats(db))
			ordersGroup.PUT("/:id/status", handler.UpdateOrderStatus(db))
		}

		// Media routes
		media := api.Group("/media")
		{
			// Public media access
			media.GET("/:filename", handler.GetMedia())

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

		// Event registration is handled via POST /events/:id/participants

		// Get port from environment
		port := os.Getenv("PORT")
		if port == "" {
			port = "8001"
		}

		logger.Fatal(r.Run(":" + port))
	}
}
