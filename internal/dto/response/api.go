package response

import (
	"time"
)

// ApiResponse is a generic response wrapper for all API responses
type ApiResponse[T any] struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message,omitempty"`
	Data      T         `json:"data,omitempty"`
	Errors    any       `json:"errors,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewSuccess creates a successful API response
func NewSuccess[T any](data T, message string) ApiResponse[T] {
	return ApiResponse[T]{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewSuccessWithData creates a successful API response with just data
func NewSuccessWithData[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewError creates an error API response
func NewError[T any](message string) ApiResponse[T] {
	return ApiResponse[T]{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewErrorWithDetails creates an error API response with details
func NewErrorWithDetails[T any](message string, errors any) ApiResponse[T] {
	return ApiResponse[T]{
		Success:   false,
		Message:   message,
		Errors:    errors,
		Timestamp: time.Now(),
	}
}

// PageInfo contains pagination information
type PageInfo struct {
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// PagedResponse wraps a list response with pagination info
type PagedResponse[T any] struct {
	Items    []T      `json:"items"`
	PageInfo PageInfo `json:"page_info"`
}

// NewPagedResponse creates a new paged response
func NewPagedResponse[T any](items []T, page, size int, total int64) PagedResponse[T] {
	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return PagedResponse[T]{
		Items: items,
		PageInfo: PageInfo{
			Page:       page,
			Size:       size,
			TotalItems: total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}
}
