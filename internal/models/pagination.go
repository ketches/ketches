package models

// PaginationRequest represents the standard pagination request parameters
type PaginationRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"page_size" json:"page_size"`
	Search   string `form:"search" json:"search"`
}

// PaginationResponse represents the standard pagination response
type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// DefaultPaginationRequest returns default pagination values
func DefaultPaginationRequest() PaginationRequest {
	return PaginationRequest{
		Page:     1,
		PageSize: 10,
		Search:   "",
	}
}

// Validate validates and normalizes pagination parameters
func (r *PaginationRequest) Validate() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 10
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
}

// GetOffset calculates the offset for database queries
func (r *PaginationRequest) GetOffset() int {
	return (r.Page - 1) * r.PageSize
}

// BuildPaginationResponse builds the standard pagination response
func BuildPaginationResponse(total int64, page, pageSize int) PaginationResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return PaginationResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
