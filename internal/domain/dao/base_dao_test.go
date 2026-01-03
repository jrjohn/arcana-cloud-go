package dao

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPageResult(t *testing.T) {
	items := []*string{ptr("a"), ptr("b"), ptr("c")}
	result := NewPageResult(items, 10, 1, 3)

	assert.Equal(t, items, result.Items)
	assert.Equal(t, int64(10), result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 3, result.Size)
}

func TestPageResult_TotalPages(t *testing.T) {
	tests := []struct {
		name       string
		totalCount int64
		size       int
		expected   int
	}{
		{"exact division", 20, 10, 2},
		{"with remainder", 25, 10, 3},
		{"single page", 5, 10, 1},
		{"empty", 0, 10, 0},
		{"zero size", 10, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &PageResult[string]{
				TotalCount: tt.totalCount,
				Size:       tt.size,
			}
			assert.Equal(t, tt.expected, result.TotalPages())
		})
	}
}

func TestPageResult_HasNext(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		totalCount int64
		size       int
		expected   bool
	}{
		{"has next page", 1, 20, 10, true},
		{"last page", 2, 20, 10, false},
		{"only page", 1, 5, 10, false},
		{"middle page", 2, 30, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &PageResult[string]{
				Page:       tt.page,
				TotalCount: tt.totalCount,
				Size:       tt.size,
			}
			assert.Equal(t, tt.expected, result.HasNext())
		})
	}
}

func TestPageResult_HasPrev(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		expected bool
	}{
		{"first page", 1, false},
		{"second page", 2, true},
		{"third page", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &PageResult[string]{
				Page: tt.page,
			}
			assert.Equal(t, tt.expected, result.HasPrev())
		})
	}
}

func TestQueryOption(t *testing.T) {
	opt := QueryOption{
		OrderBy:    "created_at",
		Descending: true,
		Preloads:   []string{"User", "Plugin"},
		Conditions: map[string]any{"status": "active"},
	}

	assert.Equal(t, "created_at", opt.OrderBy)
	assert.True(t, opt.Descending)
	assert.Len(t, opt.Preloads, 2)
	assert.Equal(t, "active", opt.Conditions["status"])
}

func ptr(s string) *string {
	return &s
}
