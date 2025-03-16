package data

import (
	"time"

	"github.com/araaavind/zoko-im/internal/validator"
)

type Filters struct {
	Cursor   time.Time
	PageSize int
}

type Metadata struct {
	CurrentCursor time.Time `json:"current_cursor"`
	NextCursor    time.Time `json:"next_cursor"`
	PageSize      int       `json:"page_size"`
	TotalSize     int       `json:"total_size"`
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(validator.ValidTimestamp(f.Cursor), "cursor", "Cursor must be a valid timestamp")
	v.Check(validator.IsTimestampInPast(f.Cursor), "cursor", "Cursor must be in the past")
	v.Check(f.PageSize > 0, "page_size", "Page size must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "Page size must be a maximum of 100")
}

func getEmptyMetadata(cursor time.Time, pageSize int) Metadata {
	return Metadata{
		CurrentCursor: cursor,
		NextCursor:    cursor,
		PageSize:      pageSize,
		TotalSize:     0,
	}
}

func calculateMetadata(currentCursor, nextCursor time.Time, totalSize, pageSize int) Metadata {
	if totalSize == 0 {
		return getEmptyMetadata(currentCursor, pageSize)
	}

	return Metadata{
		CurrentCursor: currentCursor,
		NextCursor:    nextCursor,
		PageSize:      pageSize,
		TotalSize:     totalSize,
	}
}

func (f Filters) limit() int {
	return f.PageSize
}
