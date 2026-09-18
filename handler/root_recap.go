package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// BusinessRecapResponse represents the full executive overview for root/business owner
type BusinessRecapResponse struct {
	Users struct {
		Total        int `json:"total"`
		Archers      int `json:"archers"`
		Organizers   int `json:"organizers"`
		Clubs        int `json:"clubs"`
		Scorekeepers int `json:"scorekeepers"`
		NewThisMonth int `json:"new_this_month"`
	} `json:"users"`

	Tournaments struct {
		Total        int                    `json:"total"`
		Active       int                    `json:"active"`
		Completed    int                    `json:"completed"`
		Draft        int                    `json:"draft"`
		Participants int                    `json:"total_participants"`
		TopEvents    []RecapTopTournament   `json:"top_events"`
	} `json:"tournaments"`

	Finance struct {
		TotalGMV            float64            `json:"total_gmv"`
		TotalTransactions   int                `json:"total_transactions"`
		PaidTransactions    int                `json:"paid_transactions"`
		PendingTransactions int                `json:"pending_transactions"`
		ThisMonthRevenue    float64            `json:"this_month_revenue"`
		RecentTransactions  []RecapTransaction `json:"recent_transactions"`
	} `json:"finance"`

	Content struct {
		TotalArticles int               `json:"total_articles"`
		TotalViews    int               `json:"total_views"`
		TopArticles   []RecapTopArticle `json:"top_articles"`
	} `json:"content"`
}

type RecapTopTournament struct {
	UUID              string  `db:"uuid" json:"id"`
	Slug              *string `db:"slug" json:"slug"`
	Name              string  `db:"name" json:"name"`
	Venue             *string `db:"venue" json:"venue"`
	City              *string `db:"city" json:"city"`
	Status            string  `db:"status" json:"status"`
	StartDate         *string `db:"start_date" json:"start_date"`
	ParticipantsCount int     `db:"participants_count" json:"participants_count"`
}

type RecapTransaction struct {
	Reference      string     `db:"reference" json:"reference"`
	ArcherName     string     `db:"archer_name" json:"archer_name"`
	ArcherEmail    string     `db:"archer_email" json:"archer_email"`
	TournamentName string     `db:"tournament_name" json:"tournament_name"`
	PaymentMethod  *string    `db:"payment_method" json:"payment_method"`
	Amount         float64    `db:"amount" json:"amount"`
	Status         string     `db:"status" json:"status"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

type RecapTopArticle struct {
	UUID        string  `db:"uuid" json:"id"`
	Slug        string  `db:"slug" json:"slug"`
	Title       string  `db:"title" json:"title"`
	Category    string  `db:"category" json:"category"`
	Views       int     `db:"views" json:"views"`
	ImageURL    string  `db:"image_url" json:"image"`
	PublishedAt *string `db:"published_at" json:"published_at"`
}

// RootGetBusinessRecap gathers all business metrics for the owner dashboard
func RootGetBusinessRecap(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var recap BusinessRecapResponse

		// 1. User metrics
		_ = db.Get(&recap.Users.Archers, "SELECT COUNT(*) FROM archers")
		_ = db.Get(&recap.Users.Organizers, "SELECT COUNT(*) FROM organizers")
		_ = db.Get(&recap.Users.Clubs, "SELECT COUNT(*) FROM clubs")
		_ = db.Get(&recap.Users.Scorekeepers, "SELECT COUNT(*) FROM scorekeepers")
		recap.Users.Total = recap.Users.Archers + recap.Users.Organizers + recap.Users.Clubs + recap.Users.Scorekeepers

		_ = db.Get(&recap.Users.NewThisMonth, `
			SELECT 
				(SELECT COUNT(*) FROM archers WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)) +
				(SELECT COUNT(*) FROM organizers WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)) +
				(SELECT COUNT(*) FROM clubs WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY))
		`)

		// 2. Tournament metrics
		_ = db.Get(&recap.Tournaments.Total, "SELECT COUNT(*) FROM tournaments")
		_ = db.Get(&recap.Tournaments.Active, "SELECT COUNT(*) FROM tournaments WHERE status IN ('active', 'published') AND (end_date IS NULL OR end_date >= NOW())")
		_ = db.Get(&recap.Tournaments.Completed, "SELECT COUNT(*) FROM tournaments WHERE end_date < NOW()")
		_ = db.Get(&recap.Tournaments.Draft, "SELECT COUNT(*) FROM tournaments WHERE status = 'draft'")
		_ = db.Get(&recap.Tournaments.Participants, "SELECT COUNT(*) FROM tournament_participants")

		// Top tournaments by participation
		topTournamentsQuery := `
			SELECT 
				t.uuid, t.slug, t.name, t.venue, t.city, t.status,
				DATE_FORMAT(t.start_date, '%Y-%m-%d') as start_date,
				COUNT(tp.id) as participants_count
			FROM tournaments t
			LEFT JOIN tournament_participants tp ON t.uuid = tp.tournament_id
			GROUP BY t.uuid, t.slug, t.name, t.venue, t.city, t.status, t.start_date
			ORDER BY participants_count DESC, t.created_at DESC
			LIMIT 5
		`
		var topEvents []RecapTopTournament
		if err := db.Select(&topEvents, topTournamentsQuery); err == nil {
			recap.Tournaments.TopEvents = topEvents
		} else {
			recap.Tournaments.TopEvents = []RecapTopTournament{}
		}

		// 3. Financial & revenue metrics
		_ = db.Get(&recap.Finance.TotalGMV, "SELECT COALESCE(SUM(amount), 0) FROM payment_transactions WHERE status = 'paid'")
		_ = db.Get(&recap.Finance.TotalTransactions, "SELECT COUNT(*) FROM payment_transactions")
		_ = db.Get(&recap.Finance.PaidTransactions, "SELECT COUNT(*) FROM payment_transactions WHERE status = 'paid'")
		_ = db.Get(&recap.Finance.PendingTransactions, "SELECT COUNT(*) FROM payment_transactions WHERE status = 'pending'")
		_ = db.Get(&recap.Finance.ThisMonthRevenue, "SELECT COALESCE(SUM(amount), 0) FROM payment_transactions WHERE status = 'paid' AND created_at >= DATE_FORMAT(NOW(), '%Y-%m-01')")

		// Recent transactions ledger
		recentTxQuery := `
			SELECT 
				pt.reference,
				COALESCE(a.full_name, 'Archer Member') as archer_name,
				COALESCE(a.email, '-') as archer_email,
				COALESCE(t.name, 'Archery Tournament') as tournament_name,
				pt.payment_method,
				pt.amount,
				pt.status,
				pt.created_at
			FROM payment_transactions pt
			LEFT JOIN archers a ON pt.user_id = a.uuid
			LEFT JOIN tournaments t ON pt.tournament_id = t.uuid
			ORDER BY pt.created_at DESC
			LIMIT 8
		`
		var recentTx []RecapTransaction
		if err := db.Select(&recentTx, recentTxQuery); err == nil {
			recap.Finance.RecentTransactions = recentTx
		} else {
			recap.Finance.RecentTransactions = []RecapTransaction{}
		}

		// 4. Content & article metrics
		_ = db.Get(&recap.Content.TotalArticles, "SELECT COUNT(*) FROM blog_articles")
		_ = db.Get(&recap.Content.TotalViews, "SELECT COALESCE(SUM(views), 0) FROM blog_articles")

		topArticlesQuery := `
			SELECT uuid, slug, title, category, views, image_url, DATE_FORMAT(published_at, '%Y-%m-%d') as published_at
			FROM blog_articles
			ORDER BY views DESC, published_at DESC
			LIMIT 4
		`
		var topArticles []RecapTopArticle
		if err := db.Select(&topArticles, topArticlesQuery); err == nil {
			recap.Content.TopArticles = topArticles
		} else {
			recap.Content.TopArticles = []RecapTopArticle{}
		}

		c.JSON(http.StatusOK, recap)
	}
}
