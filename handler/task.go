package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TaskDeveloper struct {
	UUID        string     `json:"uuid" db:"uuid"`
	Title       string     `json:"title" db:"title" binding:"required"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status"`
	Priority    string     `json:"priority" db:"priority"`
	DueDate     *time.Time `json:"due_date" db:"due_date"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

func GetTasks(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tasks []TaskDeveloper
		err := db.Select(&tasks, "SELECT * FROM task_developer ORDER BY created_at DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

func CreateTask(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title       string `json:"title" binding:"required"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		taskUUID := uuid.New().String()
		now := time.Now()

		_, err := db.Exec(`
			INSERT INTO task_developer (uuid, title, description, status, priority, created_at, updated_at)
			VALUES (?, ?, ?, 'pending', 'medium', ?, ?)
		`, taskUUID, req.Title, req.Description, now, now)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Task created", "uuid": taskUUID})
	}
}

func UpdateTask(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskUUID := c.Param("uuid")
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE task_developer 
			SET title = ?, description = ?, updated_at = NOW()
			WHERE uuid = ?
		`, req.Title, req.Description, taskUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Task updated"})
	}
}

func DeleteTask(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskUUID := c.Param("uuid")
		_, err := db.Exec("DELETE FROM task_developer WHERE uuid = ?", taskUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
	}
}
