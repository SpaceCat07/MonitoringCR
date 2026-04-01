package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaginationRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type PaginationMeta struct {
	Offset      int   `json:"offset"`
	Limit       int   `json:"limit"`
	Count       int   `json:"count"`
	Total       int64 `json:"total"`
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	HasPrev     bool  `json:"has_prev"`
	HasNext     bool  `json:"has_next"`
	PrevOffset  int   `json:"prev_offset"`
	NextOffset  int   `json:"next_offset"`
}

func ParsePagination(c *gin.Context, defaultLimit, maxLimit int) PaginationRequest {
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return PaginationRequest{
		Offset: offset,
		Limit:  limit,
	}
}

func BuildPaginationMeta(offset, limit, count int, total int64) PaginationMeta {
	hasPrev := offset > 0
	hasNext := int64(offset+limit) < total

	prevOffset := offset - limit
	if prevOffset < 0 {
		prevOffset = 0
	}

	nextOffset := offset + limit
	if int64(nextOffset) > total {
		nextOffset = offset
	}

	currentPage := 1
	if limit > 0 {
		currentPage = (offset / limit) + 1
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return PaginationMeta{
		Offset:      offset,
		Limit:       limit,
		Count:       count,
		Total:       total,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		HasPrev:     hasPrev,
		HasNext:     hasNext,
		PrevOffset:  prevOffset,
		NextOffset:  nextOffset,
	}
}