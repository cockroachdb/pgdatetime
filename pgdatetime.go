package pgdatetime

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// Order refers to the order of the date.
type Order uint8

//go:generate stringer -type=Order -trimprefix=Order

const (
	OrderMDY Order = iota
	OrderDMY
	OrderYMD
)

// Style refers to the style of the date.
type Style uint8

//go:generate stringer -type=Style -trimprefix=Style

const (
	StyleISO Style = iota
	StyleSQL
	StylePostgres
	StyleGerman
)

// DateStyle refers to the output style supported by PostgreSQL.
// See also: https://www.postgresql.org/docs/current/datatype-datetime.html#DATATYPE-DATETIME-OUTPUT
type DateStyle struct {
	Order Order
	Style Style

	// FixedZonePrefix is set if we should ignore printing out the shorthand
	// timezone if it begins with this prefix.
	// Leave blank to always output timezone name.
	FixedZonePrefix string
}

// ParseError is an error that appears during parsing.
type ParseError struct {
	Description string
	Idx         int
}

// ParseResultType is the type of result time returns.
type ParseResultType int

//go:generate stringer -type=ParseResultType -trimprefix=ParseResultType

const (
	// ParseResultTypeAbsoluteTime signifies an absolute return type of time.
	ParseResultTypeAbsoluteTime ParseResultType = iota
	// ParseResultTypeRelativeTime signifies time was parsed to a relative
	// to a given point in time, e.g. yesterday, today, tomorrow.
	ParseResultTypeRelativeTime
	// ParseResultTypePosInfinity signifies time is +Infinity.
	ParseResultTypePosInfinity
	// ParseResultTypeNegInfinity signifies time is -Infinity.
	ParseResultTypeNegInfinity
)

// ParseResult returns the result of parsing a time.
type ParseResult struct {
	Type ParseResultType
	Time time.Time
}

// NewParseError returns a ParseError with the given fields.
func NewParseError(idx int, description string) *ParseError {
	return &ParseError{Description: description, Idx: idx}
}

// NewParseErrorf returns a ParseError with the given fields.
func NewParseErrorf(idx int, descriptionf string, args ...interface{}) *ParseError {
	return &ParseError{Description: fmt.Sprintf(descriptionf, args...), Idx: idx}
}

// Error implements the error interface.
func (pe *ParseError) Error() string {
	return fmt.Sprintf(
		"error parsing datetime at index %d: %s",
		pe.Idx,
		pe.Description,
	)
}

var _ error = (*ParseError)(nil)

// ParseTimestampTZ parses a TimestampTZ element.
func ParseTimestampTZ(dateStyle DateStyle, now time.Time, s string) (ParseResult, error) {
	tokens, err := tokenizeDateTime(s)
	if err != nil {
		return ParseResult{}, err
	}
	return decodeTokens(dateStyle, now, tokens)
}

func writeTimeToBuffer(buf *bytes.Buffer, t time.Time) {
	buf.WriteString(t.Format(" 15:04:05.999999"))
}

func writeTextTimeZoneToBuffer(buf *bytes.Buffer, ds DateStyle, t time.Time) {
	buf.WriteRune(' ')
	z, _ := t.Zone()
	// Only write zone name if it exists.
	if ds.FixedZonePrefix == "" || !strings.HasPrefix(z, ds.FixedZonePrefix) {
		buf.WriteString(t.Format("MST"))
	}
}

// WriteToBuffer writes the given time into the given buffer.
func WriteToBuffer(buf *bytes.Buffer, ds DateStyle, t time.Time, includeTimeZone bool) {
	// In years <= 0, should as BC.
	isBC := false
	year := t.Year()
	if year <= 0 {
		year = -year + 1
		isBC = true
	}
	outputYear := func() {
		buf.WriteString(fmt.Sprintf("%04d", int64(year)))
	}
	switch ds.Style {
	case StyleSQL:
		switch ds.Order {
		case OrderYMD:
			outputYear()
			buf.WriteString(t.Format("/01/02"))
		case OrderDMY:
			buf.WriteString(t.Format("02/01/"))
			outputYear()
		default:
			buf.WriteString(t.Format("01/02/"))
			outputYear()
		}

		writeTimeToBuffer(buf, t)
		if includeTimeZone {
			writeTextTimeZoneToBuffer(buf, ds, t)
		}
	case StyleGerman:
		// Always DMY for German.
		buf.WriteString(t.Format("02.01."))
		outputYear()
		writeTimeToBuffer(buf, t)
		if includeTimeZone {
			writeTextTimeZoneToBuffer(buf, ds, t)
		}
	case StylePostgres:
		buf.WriteString(t.Format("Mon Jan 2 15:04:05.999999 "))
		outputYear()
		if includeTimeZone {
			writeTextTimeZoneToBuffer(buf, ds, t)
		}
	default:
		// Always YMD for ISO.
		outputYear()
		buf.WriteString(t.Format("-01-02"))

		writeTimeToBuffer(buf, t)
		if includeTimeZone {
			_, zoneOffset := t.Zone()
			if minOffsetInSecs := zoneOffset % 3600; minOffsetInSecs == 0 {
				buf.WriteString(t.Format("-07"))
			} else {
				// Only print the minute/second offset if it exists.
				if secOffset := zoneOffset % 60; secOffset != 0 {
					buf.WriteString(t.Format("-07:00:00"))
				} else {
					buf.WriteString(t.Format("-07:00"))
				}
			}
		}
	}

	if isBC {
		buf.WriteString(" BC")
	}
}

// Format formats the given time as the given DateStyle.
func Format(ds DateStyle, t time.Time, includeTimeZone bool) string {
	var b bytes.Buffer
	WriteToBuffer(&b, ds, t, includeTimeZone)
	return b.String()
}
