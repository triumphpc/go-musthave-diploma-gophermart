// Package jsontime prepare custom formats for marshal
// @author Vrulin Sergey (aka Alex Versus)
package jsontime

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type JSONTime time.Time

const DefaultFormat = time.RFC3339

var layouts = []string{
	DefaultFormat,
	"2006-01-02T15:04Z",        // ISO 8601 UTC
	"2006-01-02T15:04:05Z",     // ISO 8601 UTC
	"2006-01-02T15:04:05.000Z", // ISO 8601 UTC
	"2006-01-02T15:04:05",      // ISO 8601 UTC
	"2006-01-02 15:04",         // Custom UTC
	"2006-01-02 15:04:05",      // Custom UTC
	"2006-01-02 15:04:05.000",  // Custom UTC
}

// JSONTime
func (jt *JSONTime) String() string {
	t := time.Time(*jt)
	tl := t.In(time.Local)

	return tl.Format(DefaultFormat)
}

func (jt JSONTime) MarshalJSON() ([]byte, error) {

	return []byte(fmt.Sprintf(`"%s"`, jt.String())), nil
}

func (jt *JSONTime) UnmarshalJSON(b []byte) error {
	timeString := strings.Trim(string(b), `"`)
	for _, layout := range layouts {
		t, err := time.Parse(layout, timeString)
		if err == nil {
			*jt = JSONTime(t)
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Invalid date format: %s", timeString))
}

func (jt *JSONTime) ToTime() time.Time {
	return time.Time(*jt)
}
