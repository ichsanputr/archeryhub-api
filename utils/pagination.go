package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	TotalCount  int `json:"total_count"`
	Limit       int `json:"limit"`
	Offset      int `json:"offset"`
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
}

// GetPaginationParams extracts limit, offset and page from query parameters
func GetPaginationParams(c *gin.Context) (limit, offset, page int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	// Use offset if provided, otherwise calculate from page
	offsetStr := c.Query("offset")
	if offsetStr != "" {
		offset, _ = strconv.Atoi(offsetStr)
	} else {
		offset = (page - 1) * limit
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset, page
}

// CalculatePagination creates the metadata for pagination response
func CalculatePagination(totalCount, limit, offset, page int) PaginationMeta {
	lastPage := 1
	if limit > 0 {
		lastPage = (totalCount + limit - 1) / limit
	}
	if lastPage < 1 {
		lastPage = 1
	}

	return PaginationMeta{
		TotalCount:  totalCount,
		Limit:       limit,
		Offset:      offset,
		CurrentPage: page,
		LastPage:    lastPage,
	}
}
