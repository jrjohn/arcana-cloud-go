package response

import (
	"testing"
	"time"
)

func TestNewSuccess(t *testing.T) {
	data := map[string]string{"key": "value"}
	message := "Operation successful"

	resp := NewSuccess(data, message)

	if !resp.Success {
		t.Error("NewSuccess should set Success to true")
	}
	if resp.Message != message {
		t.Errorf("NewSuccess Message = %v, want %v", resp.Message, message)
	}
	if resp.Data == nil {
		t.Error("NewSuccess should set Data")
	}
	if resp.Timestamp.IsZero() {
		t.Error("NewSuccess should set Timestamp")
	}
}

func TestNewSuccessWithData(t *testing.T) {
	data := []int{1, 2, 3}

	resp := NewSuccessWithData(data)

	if !resp.Success {
		t.Error("NewSuccessWithData should set Success to true")
	}
	if resp.Message != "" {
		t.Errorf("NewSuccessWithData Message = %v, want empty", resp.Message)
	}
	if len(resp.Data) != 3 {
		t.Errorf("NewSuccessWithData Data length = %v, want 3", len(resp.Data))
	}
	if resp.Timestamp.IsZero() {
		t.Error("NewSuccessWithData should set Timestamp")
	}
}

func TestNewError(t *testing.T) {
	message := "An error occurred"

	resp := NewError[any](message)

	if resp.Success {
		t.Error("NewError should set Success to false")
	}
	if resp.Message != message {
		t.Errorf("NewError Message = %v, want %v", resp.Message, message)
	}
	if resp.Timestamp.IsZero() {
		t.Error("NewError should set Timestamp")
	}
}

func TestNewErrorWithDetails(t *testing.T) {
	message := "Validation failed"
	errors := map[string]string{
		"email": "invalid format",
		"name":  "required",
	}

	resp := NewErrorWithDetails[any](message, errors)

	if resp.Success {
		t.Error("NewErrorWithDetails should set Success to false")
	}
	if resp.Message != message {
		t.Errorf("NewErrorWithDetails Message = %v, want %v", resp.Message, message)
	}
	if resp.Errors == nil {
		t.Error("NewErrorWithDetails should set Errors")
	}
	if resp.Timestamp.IsZero() {
		t.Error("NewErrorWithDetails should set Timestamp")
	}
}

func TestApiResponse_GenericTypes(t *testing.T) {
	// Test with string data
	strResp := NewSuccess("hello", "string data")
	if strResp.Data != "hello" {
		t.Errorf("String data = %v, want hello", strResp.Data)
	}

	// Test with int data
	intResp := NewSuccess(42, "int data")
	if intResp.Data != 42 {
		t.Errorf("Int data = %v, want 42", intResp.Data)
	}

	// Test with struct data
	type User struct {
		ID   int
		Name string
	}
	user := User{ID: 1, Name: "Test"}
	structResp := NewSuccess(user, "struct data")
	if structResp.Data.ID != 1 {
		t.Errorf("Struct data ID = %v, want 1", structResp.Data.ID)
	}

	// Test with slice data
	sliceResp := NewSuccess([]string{"a", "b", "c"}, "slice data")
	if len(sliceResp.Data) != 3 {
		t.Errorf("Slice data length = %v, want 3", len(sliceResp.Data))
	}
}

func TestPageInfo_Struct(t *testing.T) {
	pageInfo := PageInfo{
		Page:       1,
		Size:       10,
		TotalItems: 100,
		TotalPages: 10,
		HasNext:    true,
		HasPrev:    false,
	}

	if pageInfo.Page != 1 {
		t.Errorf("PageInfo.Page = %v, want 1", pageInfo.Page)
	}
	if pageInfo.Size != 10 {
		t.Errorf("PageInfo.Size = %v, want 10", pageInfo.Size)
	}
	if pageInfo.TotalItems != 100 {
		t.Errorf("PageInfo.TotalItems = %v, want 100", pageInfo.TotalItems)
	}
	if pageInfo.TotalPages != 10 {
		t.Errorf("PageInfo.TotalPages = %v, want 10", pageInfo.TotalPages)
	}
	if !pageInfo.HasNext {
		t.Error("PageInfo.HasNext should be true")
	}
	if pageInfo.HasPrev {
		t.Error("PageInfo.HasPrev should be false")
	}
}

func TestNewPagedResponse(t *testing.T) {
	tests := []struct {
		name           string
		items          []string
		page           int
		size           int
		total          int64
		expectedPages  int
		expectedHasNext bool
		expectedHasPrev bool
	}{
		{
			name:           "first page",
			items:          []string{"a", "b", "c"},
			page:           1,
			size:           3,
			total:          10,
			expectedPages:  4,
			expectedHasNext: true,
			expectedHasPrev: false,
		},
		{
			name:           "middle page",
			items:          []string{"d", "e", "f"},
			page:           2,
			size:           3,
			total:          10,
			expectedPages:  4,
			expectedHasNext: true,
			expectedHasPrev: true,
		},
		{
			name:           "last page",
			items:          []string{"j"},
			page:           4,
			size:           3,
			total:          10,
			expectedPages:  4,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:           "single page",
			items:          []string{"a", "b"},
			page:           1,
			size:           10,
			total:          2,
			expectedPages:  1,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
		{
			name:           "empty result",
			items:          []string{},
			page:           1,
			size:           10,
			total:          0,
			expectedPages:  0,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
		{
			name:           "exact page boundary",
			items:          []string{"a", "b", "c", "d", "e"},
			page:           2,
			size:           5,
			total:          10,
			expectedPages:  2,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewPagedResponse(tt.items, tt.page, tt.size, tt.total)

			if len(resp.Items) != len(tt.items) {
				t.Errorf("Items length = %v, want %v", len(resp.Items), len(tt.items))
			}
			if resp.PageInfo.Page != tt.page {
				t.Errorf("Page = %v, want %v", resp.PageInfo.Page, tt.page)
			}
			if resp.PageInfo.Size != tt.size {
				t.Errorf("Size = %v, want %v", resp.PageInfo.Size, tt.size)
			}
			if resp.PageInfo.TotalItems != tt.total {
				t.Errorf("TotalItems = %v, want %v", resp.PageInfo.TotalItems, tt.total)
			}
			if resp.PageInfo.TotalPages != tt.expectedPages {
				t.Errorf("TotalPages = %v, want %v", resp.PageInfo.TotalPages, tt.expectedPages)
			}
			if resp.PageInfo.HasNext != tt.expectedHasNext {
				t.Errorf("HasNext = %v, want %v", resp.PageInfo.HasNext, tt.expectedHasNext)
			}
			if resp.PageInfo.HasPrev != tt.expectedHasPrev {
				t.Errorf("HasPrev = %v, want %v", resp.PageInfo.HasPrev, tt.expectedHasPrev)
			}
		})
	}
}

func TestPagedResponse_GenericTypes(t *testing.T) {
	// Test with struct
	type User struct {
		ID   int
		Name string
	}
	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	resp := NewPagedResponse(users, 1, 10, 2)

	if len(resp.Items) != 2 {
		t.Errorf("Items length = %v, want 2", len(resp.Items))
	}
	if resp.Items[0].Name != "Alice" {
		t.Errorf("First item name = %v, want Alice", resp.Items[0].Name)
	}

	// Test with int
	numbers := []int{1, 2, 3, 4, 5}
	intResp := NewPagedResponse(numbers, 1, 5, 5)
	if len(intResp.Items) != 5 {
		t.Errorf("Int items length = %v, want 5", len(intResp.Items))
	}
}

func TestApiResponse_Timestamp(t *testing.T) {
	before := time.Now()
	resp := NewSuccess("test", "message")
	after := time.Now()

	if resp.Timestamp.Before(before) || resp.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, should be between %v and %v", resp.Timestamp, before, after)
	}
}

func TestApiResponse_Struct(t *testing.T) {
	resp := ApiResponse[string]{
		Success:   true,
		Message:   "test message",
		Data:      "test data",
		Errors:    nil,
		Timestamp: time.Now(),
	}

	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Message != "test message" {
		t.Errorf("Message = %v, want test message", resp.Message)
	}
	if resp.Data != "test data" {
		t.Errorf("Data = %v, want test data", resp.Data)
	}
}

func TestPagedResponse_Struct(t *testing.T) {
	resp := PagedResponse[int]{
		Items: []int{1, 2, 3},
		PageInfo: PageInfo{
			Page:       1,
			Size:       10,
			TotalItems: 3,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	if len(resp.Items) != 3 {
		t.Errorf("Items length = %v, want 3", len(resp.Items))
	}
	if resp.PageInfo.TotalItems != 3 {
		t.Errorf("TotalItems = %v, want 3", resp.PageInfo.TotalItems)
	}
}

// Benchmarks
func BenchmarkNewSuccess(b *testing.B) {
	data := map[string]string{"key": "value"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewSuccess(data, "success")
	}
}

func BenchmarkNewError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewError[any]("error message")
	}
}

func BenchmarkNewPagedResponse(b *testing.B) {
	items := []string{"a", "b", "c", "d", "e"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPagedResponse(items, 1, 5, 100)
	}
}
