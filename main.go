package main

import (
	"net/http"
	"os"

	"archeryhub-api/database"
	_ "archeryhub-api/docs"
	"archeryhub-api/handler"
	"archeryhub-api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title ArcheryHub Mobile API
// @version 1.0
// @description This documentation is exclusively for the ArcheryHub Mobile Application.
// @description NOTE: This Swagger only contains endpoints relevant to mobile app workflows (Scoring, Events, and Auth).
// @host localhost:8001
// @BasePath /api/v1

var logger *logrus.Logger

// fileOnlyHook is a logrus hook that writes specific log levels to a file
type fileOnlyHook struct {
	file   *os.File
	levels []logrus.Level
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
	// Set Gin to release mode to disable debug logging
	gin.SetMode(gin.ReleaseMode)

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

	// Initialize Gin router
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

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
			"message": "Welcome to Archeryhub.id API",
			"status":  "running",
		})
	})

	r.GET("/preview/scoresheet", handler.PreviewScoresheet())

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

		// Session endpoint (Global)

		api.GET("/archer/me", middleware.AuthMiddleware(), handler.GetArcherProfile(db))
		api.GET("/organization/me", middleware.AuthMiddleware(), handler.GetOrganizationProfile(db))
		api.GET("/club/me", middleware.AuthMiddleware(), handler.GetClubMe(db))
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

			// Google OAuth
			auth.GET("/google", middleware.OptionalAuthMiddleware(), handler.InitiateGoogleAuth(db))
			auth.GET("/google/callback", middleware.OptionalAuthMiddleware(), handler.GoogleCallback(db))
			auth.POST("/google/callback", middleware.OptionalAuthMiddleware(), handler.GoogleCallback(db))

			auth.GET("/avatar/:identifier", handler.GetArcherProfileImage(db))
		}

		// User routes
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{

			user.GET("/profile", handler.GetUserProfile(db))
			user.PUT("/profile", handler.UpdateUserProfile(db)) // Generic profile update handler
			user.PUT("/password", handler.UpdatePassword(db))
			user.GET("/settings", handler.GetUserSettings(db))
			user.PUT("/settings", handler.UpdateUserSettings(db))
			user.GET("/subscription", handler.GetMySubscription(db))
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
			events.POST("/participants/reregister", handler.ReregisterParticipant(db))

			// Public Results endpoints
			events.GET("/:id/results/qualification", handler.GetPublicQualificationResults(db))
			events.GET("/:id/results/elimination", handler.GetPublicEliminationResults(db))

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
				protected.GET("/:id/categories/:categoryId", handler.GetEventCategoryDetails(db))
				protected.PUT("/:id/categories/:categoryId", handler.UpdateEventCategory(db))
				protected.DELETE("/:id/categories/:categoryId", handler.DeleteEventCategory(db))
				protected.POST("/:id/participants", handler.RegisterParticipant(db))
				protected.POST("/:id/participants/batch", handler.BatchRegisterParticipants(db))
				protected.PUT("/:id/images", handler.UpdateEventImages(db))
				protected.PUT("/:id/schedule", handler.UpdateEventSchedule(db))
				protected.POST("/:id/payment-methods", handler.CreateEventPaymentMethod(db))
				protected.PUT("/:id/payment-methods/:methodId", handler.UpdateEventPaymentMethod(db))
				protected.DELETE("/:id/payment-methods/:methodId", handler.DeleteEventPaymentMethod(db))

				// Qualification target assignments - nested under events/:id/qualification/sessions/:sessionId
				protected.POST("/:id/qualification/sessions/:sessionId/assignments", handler.CreateBulkTargetAssignments(db))

				// Target management
				protected.POST("/:id/targets", handler.CreateEventTarget(db))
				protected.PUT("/:id/targets/batch", handler.BatchUpdateTargets(db))
				protected.PUT("/:id/targets/:target_id", handler.UpdateEventTarget(db))
				protected.DELETE("/:id/targets/:target_id", handler.DeleteEventTarget(db))
			}
		}

		// Qualification routes (event-level sessions)
		qualification := api.Group("/events/:id/qualification")
		qualification.Use(middleware.AuthMiddleware())
		{
			qualification.GET("/sessions", handler.GetQualificationSessions(db))
			qualification.POST("/sessions", handler.CreateQualificationSession(db))
			qualification.PATCH("/sessions/:sessionId", handler.UpdateQualificationSession(db))
			qualification.DELETE("/sessions/:sessionId", handler.DeleteQualificationSession(db))
			qualification.GET("/leaderboard", handler.GetQualificationLeaderboard(db))
			qualification.GET("/sessions/:sessionCode/scoresheet", handler.GetQualificationScoresheet(db))
		}

		// Elimination routes (event-level brackets)
		elimination := api.Group("/events/:id/elimination")
		elimination.Use(middleware.OptionalAuthMiddleware())
		{
			elimination.GET("/brackets", handler.GetBrackets(db))
			elimination.POST("/brackets", middleware.AuthMiddleware(), handler.CreateBracket(db))
			elimination.GET("/brackets/:bracketId", handler.GetBracket(db))
			elimination.PUT("/brackets/:bracketId", middleware.AuthMiddleware(), handler.UpdateBracket(db))
			elimination.DELETE("/brackets/:bracketId", middleware.AuthMiddleware(), handler.DeleteBracket(db))
			elimination.POST("/brackets/:bracketId/generate", middleware.AuthMiddleware(), handler.GenerateBracket(db))
			elimination.GET("/brackets/:bracketId/scores", handler.GetBracketScores(db))
			elimination.GET("/brackets/:bracketId/board-codes", handler.GetEliminationBoardCodes(db))
			elimination.GET("/brackets/:bracketId/scoresheet", handler.GetEliminationScoresheet(db))
			elimination.PUT("/brackets/:bracketId/targets", middleware.AuthMiddleware(), handler.UpdateMatchTargets(db))
			elimination.POST("/brackets/:bracketId/targets/auto-assign", middleware.AuthMiddleware(), handler.AutoAssignMatchTargets(db))
			elimination.GET("/brackets/:bracketId/matches/:matchId", handler.GetMatch(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/score", middleware.AuthMiddleware(), handler.UpdateMatchScore(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/finish", middleware.AuthMiddleware(), handler.FinishMatch(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/end", middleware.AuthMiddleware(), handler.EndMatch(db))
			elimination.POST("/brackets/:bracketId/matches/:matchId/reset", middleware.AuthMiddleware(), handler.ResetMatch(db))
		}

		// Public flat match details
		api.GET("/match/:matchId", handler.GetMatch(db))

		qualSessions := api.Group("/qualification/sessions/:sessionId")
		qualSessions.Use(middleware.AuthMiddleware())
		{
			qualSessions.GET("/assignments", handler.GetSessionAssignments(db))
			qualSessions.GET("/board-codes", handler.GetBoardCodes(db))
			qualSessions.GET("/scores", handler.GetSessionScores(db))
			qualSessions.POST("/auto-assign", handler.AutoAssignParticipants(db))
			qualSessions.POST("/reset-assignments", handler.ResetSessionAssignments(db))
			qualSessions.POST("/swap-assignments", handler.SwapTargetAssignments(db))
		}

		qualAssignments := api.Group("/qualification/assignments/:assignmentId")
		qualAssignments.Use(middleware.AuthMiddleware())
		{
			qualAssignments.GET("/scores", handler.GetQualificationAssignmentScores(db))
			qualAssignments.POST("/scores", handler.UpdateQualificationScore(db))
			qualAssignments.DELETE("", handler.DeleteQualificationAssignment(db))
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
			// Public archer routes
			archers.GET("", handler.GetArchers(db))
			archers.GET("/:id", handler.GetArcherByID(db))
			archers.GET("/:id/events", handler.GetArcherEvents(db))
			archers.GET("/registration-profile/:uuid", handler.GetArcherRegistrationProfile(db))

			// Protected archer routes
			protected := archers.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
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
				protectedTeams.PUT("/:teamId", handler.UpdateTeam(db))
				protectedTeams.DELETE("/:teamId", handler.DeleteTeam(db))
				protectedTeams.POST("/event/:eventId/sync", handler.SyncTeams(db))
			}

			teams.GET("/event/:eventId/rankings", handler.GetTeamRankings(db))
		}

		// Payment & Registration routes
		payment := api.Group("/payment")
		{
			payment.GET("/channels", handler.GetPaymentChannels(db))
			payment.GET("/status/:reference", handler.GetPaymentStatus(db))
			payment.GET("/invoice/:reference", handler.GenerateInvoicePDF(db))
			payment.POST("/create", middleware.AuthMiddleware(), handler.CreatePayment(db))
			payment.POST("/tripay/callback", handler.PaymentCallback(db))
		}

		// Root/Admin routes
		root := api.Group("/root")
		{
			root.POST("/login", handler.RootLogin(db))

			protected := root.Group("/dashboard")
			protected.Use(middleware.AuthMiddleware(), middleware.RequireRole("root"))
			{
				protected.GET("/users", handler.GetAllUsers(db))
			}
		}

		// Mobile dedicated routes
		mobile := api.Group("/mobile")
		{
			mobile.GET("/hello", handler.MobileHello())

			// 1. Authentication (no auth required)
			auth := mobile.Group("/auth")
			{
				auth.POST("/scorekeeper/login", handler.MobileScorekeeperLogin(db))
			}

			// 2. Events (public)
			mobile.GET("/events", handler.MobileListEvents(db))

			// 3. Target scan by QR/barcode code (requires auth)
			mobileAuth := mobile.Group("")
			mobileAuth.Use(middleware.AuthMiddleware())
			{
				mobileAuth.GET("/scan", handler.MobileScanTarget(db))
				mobileAuth.GET("/sessions/boards", handler.MobileGetSessionBoards(db))
				mobileAuth.GET("/assignments/:assignmentId/detail", handler.MobileGetAssignmentScoreDetail(db))
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
				sk.GET("/me", handler.MobileGetScorekeeperMe(db))
				sk.GET("/events", handler.MobileGetScorekeeperEvents(db))
			}

			// Swagger UI
			mobile.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
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

				protectedOrgs.PUT("/me", handler.UpdateOrganizationProfile(db))

				// Scorekeeper management
				scorekeepers := protectedOrgs.Group("/scorekeepers")
				{
					scorekeepers.GET("", handler.GetOrganizationScorekeepers(db))
					scorekeepers.POST("", handler.CreateScorekeeper(db))
					scorekeepers.PUT("/:id", handler.UpdateScorekeeper(db))
					scorekeepers.DELETE("/:id", handler.DeleteScorekeeper(db))
				}
			}
		}

		// Club routes
		clubs := api.Group("/clubs")
		{
			// Public club routes (specific paths MUST come before wildcard)
			clubs.GET("", handler.GetClubs(db))
			clubs.GET("/availability", handler.CheckSlugAvailability(db))
			clubs.GET("/profile/:slug", handler.GetClubProfile(db))

			// Protected club routes (prefixed to avoid wildcard collision)
			protectedClubs := clubs.Group("")
			protectedClubs.Use(middleware.AuthMiddleware())
			{
				protectedClubs.PUT("/me", handler.UpdateClubMe(db))
				protectedClubs.GET("/dashboard/stats", handler.GetClubDashboardStats(db))
				protectedClubs.PUT("/profile", handler.UpdateMyClubProfile(db))

				// Membership & Admin routes
				protectedClubs.POST("/join/:clubId", handler.JoinClub(db))
				protectedClubs.GET("/my/membership", handler.GetMyClubMembership(db))
				protectedClubs.GET("/members/:clubId", handler.GetClubMembers(db))
				protectedClubs.POST("/approve/:memberId", handler.ApproveClubMember(db))
				protectedClubs.POST("/leave", handler.LeaveClub(db))
				protectedClubs.POST("/invite", handler.InviteToClub(db))
				protectedClubs.DELETE("/members/:archerId", handler.KickClubMember(db))
				protectedClubs.PATCH("/members/:archerId/notes", handler.UpdateMemberNotes(db))

				// Registration Form Builder routes
				protectedClubs.GET("/forms", handler.GetRegistrationForm(db))
				protectedClubs.POST("/forms", handler.CreateRegistrationForm(db))
				protectedClubs.GET("/forms/:formId", handler.GetRegistrationForm(db))
				protectedClubs.PUT("/forms/:formId", handler.UpdateRegistrationForm(db))
				protectedClubs.DELETE("/forms/:formId", handler.DeleteRegistrationForm(db))
				protectedClubs.POST("/forms/:formId/publish", handler.PublishRegistrationForm(db))
				protectedClubs.PUT("/forms/:formId/reorder", handler.ReorderFormItems(db))

				// Section routes
				protectedClubs.POST("/forms/:formId/sections", handler.CreateFormSection(db))
				protectedClubs.PUT("/forms/:formId/sections/:sectionId", handler.UpdateFormSection(db))
				protectedClubs.DELETE("/forms/:formId/sections/:sectionId", handler.DeleteFormSection(db))

				// Field routes
				protectedClubs.POST("/forms/:formId/sections/:sectionId/fields", handler.CreateFormField(db))
				protectedClubs.PUT("/forms/:formId/fields/:fieldId", handler.UpdateFormField(db))
				protectedClubs.DELETE("/forms/:formId/fields/:fieldId", handler.DeleteFormField(db))

				// Membership Management routes
				membership := protectedClubs.Group("/membership")
				{
					// Stats
					membership.GET("/stats", handler.GetMembershipStats(db))

					// Packages (paket buatan club)
					membership.GET("/packages", handler.GetMembershipPackages(db))
					membership.POST("/packages", handler.CreateMembershipPackage(db))
					membership.PUT("/packages/:packageId", handler.UpdateMembershipPackage(db))
					membership.DELETE("/packages/:packageId", handler.DeleteMembershipPackage(db))

					// Subscriptions
					membership.GET("/subscriptions", handler.GetMembershipSubscriptions(db))
					membership.POST("/subscriptions", handler.AssignMembershipPackage(db))
					membership.POST("/subscriptions/:subscriptionId/pay", handler.RecordMembershipPayment(db))
					membership.GET("/subscriptions/archer/:archerId", handler.GetArcherSubscriptionHistory(db))
				}
			}

			// Public endpoints — wildcard MUST be last to avoid shadowing named routes above
			clubs.GET("/:slug/registration-form", handler.GetPublicRegistrationForm(db))
			clubs.GET("/:slug", handler.GetClubBySlug(db))
		}

		// Seller routes
		sellers := api.Group("/sellers")
		{
			protectedSellers := sellers.Group("")
			protectedSellers.Use(middleware.AuthMiddleware())
			{

				protectedSellers.PUT("/me", handler.UpdateSellerProfile(db))
			}
		}

		// Media routes
		media := api.Group("/media")
		{
			// Public media access
			media.GET("/download/:filename", handler.DownloadMedia())
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
