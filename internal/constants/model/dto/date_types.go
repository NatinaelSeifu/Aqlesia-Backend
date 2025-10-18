package dto

import (
	"fmt"
	"strings"
	"time"
)

// Date represents a date-only field that can be marshaled/unmarshaled as YYYY-MM-DD
type Date struct {
	time.Time
}

// DateLayout is the layout for date-only strings
const DateLayout = "2006-01-02"

// NewDate creates a new Date from a time.Time
func NewDate(t time.Time) Date {
	return Date{Time: t.Truncate(24 * time.Hour)}
}

// UnmarshalJSON implements json.Unmarshaler for Date
func (d *Date) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	str := strings.Trim(string(data), "\"")
	
	// Handle empty string
	if str == "" || str == "null" {
		return nil
	}
	
	// Parse the date string
	parsed, err := time.Parse(DateLayout, str)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD, got %s: %w", str, err)
	}
	
	d.Time = parsed
	return nil
}

// MarshalJSON implements json.Marshaler for Date
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", d.Time.Format(DateLayout))), nil
}

// String returns the date in YYYY-MM-DD format
func (d Date) String() string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(DateLayout)
}

// ToTime returns the underlying time.Time
func (d Date) ToTime() time.Time {
	return d.Time
}
