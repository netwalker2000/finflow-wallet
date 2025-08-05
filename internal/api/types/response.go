// internal/api/types/response.go
package types

// PaginatedResponse defines a generic structure for paginated API responses.
// T represents the type of data contained in the 'Data' slice.
type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	TotalCount int64 `json:"total_count"`
}
