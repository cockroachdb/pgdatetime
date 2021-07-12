package pgdatetime

import (
	"bytes"
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
func WriteToBuffer(buf *bytes.Buffer, ds DateStyle, t time.Time) {
	switch ds.Style {
	case StyleSQL:
		switch ds.Order {
		case OrderYMD:
			buf.WriteString(t.Format("2006/01/02"))
		case OrderDMY:
			buf.WriteString(t.Format("02/01/2006"))
		default:
			buf.WriteString(t.Format("01/02/2006"))
		}

		writeTimeToBuffer(buf, t)
		writeTextTimeZoneToBuffer(buf, ds, t)
	case StyleGerman:
		switch ds.Order {
		case OrderYMD:
			buf.WriteString(t.Format("2006.01.02"))
		case OrderDMY:
			buf.WriteString(t.Format("02.01.2006"))
		default:
			buf.WriteString(t.Format("01.02.2006"))
		}
		writeTimeToBuffer(buf, t)
		writeTextTimeZoneToBuffer(buf, ds, t)
	default:
		switch ds.Order {
		case OrderYMD:
			buf.WriteString(t.Format("2006-01-02"))
		case OrderDMY:
			buf.WriteString(t.Format("02-01-2006"))
		default:
			buf.WriteString(t.Format("01-02-2006"))
		}

		writeTimeToBuffer(buf, t)

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

// Format formats the given time as the given DateStyle.
func Format(ds DateStyle, t time.Time) string {
	var b bytes.Buffer
	WriteToBuffer(&b, ds, t)
	return b.String()
}
