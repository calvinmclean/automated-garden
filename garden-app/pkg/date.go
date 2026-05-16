package pkg

import (
	"fmt"
	"time"
)

// Date represents a date without time component.
// It stores only year, month, and day, and serializes to "YYYY-MM-DD" format.
// Backward compatibility: when unmarshaling, it will try DateOnly format first,
// then fall back to RFC3339 and extract the date portion.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// NewDate creates a Date from a time.Time preserving the date in the time's location
func NewDate(t time.Time) Date {
	return Date{
		Year:  t.Year(),
		Month: t.Month(),
		Day:   t.Day(),
	}
}

// ToTimeInLocation converts Date to time.Time at midnight in the specified location
func (d Date) ToTimeInLocation(loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, loc)
}

// Equal returns true if d is equal to other
func (d Date) Equal(other Date) bool {
	return d.Year == other.Year && d.Month == other.Month && d.Day == other.Day
}

// MarshalJSON implements json.Marshaler (outputs "YYYY-MM-DD")
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", d.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler (accepts DateOnly or RFC3339)
func (d *Date) UnmarshalJSON(data []byte) error {
	// Remove surrounding quotes
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// Try parsing as DateOnly format first
	parsed, err := time.Parse(time.DateOnly, str)
	if err == nil {
		d.Year = parsed.Year()
		d.Month = parsed.Month()
		d.Day = parsed.Day()
		return nil
	}

	// Fall back to RFC3339 for backward compatibility
	parsed, err = time.Parse(time.RFC3339, str)
	if err == nil {
		utc := parsed.UTC()
		d.Year = utc.Year()
		d.Month = utc.Month()
		d.Day = utc.Day()
		return nil
	}

	return fmt.Errorf("unable to parse date: %q", str)
}

// String returns the date in YYYY-MM-DD format
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// ParseDate parses a date string in YYYY-MM-DD format
func ParseDate(s string) (Date, error) {
	if s == "" {
		return Date{}, nil
	}

	// Try DateOnly format first
	parsed, err := time.Parse(time.DateOnly, s)
	if err == nil {
		return NewDate(parsed), nil
	}

	// Fall back to RFC3339 for backward compatibility
	parsed, err = time.Parse(time.RFC3339, s)
	if err == nil {
		return NewDate(parsed), nil
	}

	return Date{}, fmt.Errorf("unable to parse date: %q", s)
}

// MustParseDate parses a date string and panics on error (for tests)
func MustParseDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse date %q: %v", s, err))
	}
	return d
}
