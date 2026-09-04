// Package page provides pagination for list endpoints, mirroring the API
// envelope `{ items, total, page, page_size }` (contracts/api.md → Conventions).
package page

import "strconv"

// Default page size used when the client omits page_size.
const DefaultSize = 20

// MaxSize caps page_size to bound query load.
const MaxSize = 100

// Cursor describes a page of results.
type Cursor struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// ParseCursor normalises raw query values into a valid cursor.
func ParseCursor(pageStr, sizeStr string) Cursor {
	page := parsePositive(pageStr, 1)
	size := parsePositive(sizeStr, DefaultSize)
	if size > MaxSize {
		size = MaxSize
	}
	return Cursor{Page: page, PageSize: size}
}

// Offset returns the SQL OFFSET for the current page.
func (c Cursor) Offset() int { return (c.Page - 1) * c.PageSize }

// PageCount returns the number of pages for a given total.
func (c Cursor) PageCount(total int) int {
	if total <= 0 || c.PageSize <= 0 {
		return 0
	}
	return (total + c.PageSize - 1) / c.PageSize
}

// HasMore reports whether there is another page after this one.
func (c Cursor) HasMore(total int) bool { return c.Page < c.PageCount(total) }

func parsePositive(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}