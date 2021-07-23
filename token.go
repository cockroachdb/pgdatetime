package pgdatetime

import (
	"fmt"
	"strings"
	"unicode"
)

// dateTokenType maps to a DTK object in PostgreSQL.
type dateTokenType int

//go:generate stringer -type=dateTokenType -trimprefix=dateTokenType

const (
	// dateTokenTypeNumber can hold dates, e.g. (yy.dd)
	dateTokenTypeNumber dateTokenType = iota
	// dateTokenTypeString can hold months (e.g. January) or time zones (e.g. PST)
	dateTokenTypeString

	// dateTokenTypeDate can hold time zones (e.g. GMT-8)
	dateTokenTypeDate
	dateTokenTypeTime
	dateTokenTypeTZ

	dateTokenTypeSpecial
)

// dateToken represents a date token component of a datetime.
type dateToken struct {
	dateTokenType dateTokenType
	val           string
	idx           int
}

func tokenizeDateTime(s string) ([]dateToken, error) {
	s = strings.ToLower(s)
	i := 0
	ret := []dateToken{}
	isDigit := func(b byte) bool {
		return unicode.IsDigit(rune(b))
	}
	isLetter := func(b byte) bool {
		return unicode.IsLetter(rune(b))
	}
	isLetterOrDigit := func(b byte) bool {
		return unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
	}
	appendDateToken := func(t dateTokenType, start int) {
		ret = append(
			ret,
			dateToken{dateTokenType: t, val: s[start:i], idx: start},
		)
	}

	for i < len(s) {
		start := i

		// Read all digits.
		switch {
		case unicode.IsSpace(rune(s[i])):
			// Ignore spaces.
			i++
		case isDigit(s[i]):
			// Starting with a digit.
			for ; i < len(s) && isDigit(s[i]); i++ {
			}
			// If we've reached the end, treat it as a number.
			if i >= len(s) {
				appendDateToken(dateTokenTypeNumber, start)
				break
			}
			switch s[i] {
			case ':':
				// It is a time element.
				// Read the remaining time characters.
				for ; i < len(s) && (isDigit(s[i]) || s[i] == ':' || s[i] == '.'); i++ {
				}
				appendDateToken(dateTokenTypeTime, start)
			case '-', '/', '.':
				// It is a date element.
				// Mark the delimiter.
				delimiter := s[i]
				i++
				if i < len(s) && isDigit(s[i]) {
					// Read the second set of digits if any.
					for ; i < len(s) && isDigit(s[i]); i++ {
					}

					// If it's two fields and separated by a '.', treat it as a number.
					// Otherwise, treat two or three fields as a date.
					t := dateTokenTypeDate
					if delimiter == '.' {
						t = dateTokenTypeNumber
					}
					if i < len(s) && s[i] == delimiter {
						t = dateTokenTypeDate
						for ; i < len(s) && (isDigit(s[i]) || s[i] == delimiter); i++ {
						}
					}
					appendDateToken(t, start)
					break
				}
				// This could be a date with text, e.g. 13/Feb/2021.
				for ; i < len(s) && (isLetterOrDigit(s[i]) || s[i] == delimiter); i++ {
				}
				appendDateToken(dateTokenTypeDate, start)
			default:
				appendDateToken(dateTokenTypeNumber, start)
			}
		case s[i] == '.':
			// Fractional seconds.
			i++
			for ; i < len(s) && isDigit(s[i]); i++ {
			}
			appendDateToken(dateTokenTypeNumber, start)
		case isLetter(s[i]):
			// Text - could be date string, month, DOW, special or timezone.
			for ; i < len(s) && isLetter(s[i]); i++ {
			}

			t := dateTokenTypeString
			// Could be a date with a leading text month.
			if i < len(s) && (s[i] == '-' || s[i] == '/' || s[i] == '.') {
				delimiter := s[i]
				for ; i < len(s) && (isDigit(s[i]) || s[i] == delimiter); i++ {
				}
				t = dateTokenTypeDate
			}
			appendDateToken(t, start)
		case s[i] == '+' || s[i] == '-':
			// Timezone or special.
			i++
			for ; i < len(s) && unicode.IsSpace(rune(s[i])); i++ {
			}
			if i == len(s) {
				return nil, NewParseError(
					"expected letters or characters after + or -",
					start,
				)
			}
			switch {
			case isDigit(s[i]):
				for ; i < len(s) && (isDigit(s[i]) || s[i] == ':' || s[i] == '.'); i++ {
				}
				appendDateToken(dateTokenTypeTZ, start)
			case isLetter(s[i]):
				for ; i < len(s) && isLetter(s[i]); i++ {
				}
				appendDateToken(dateTokenTypeSpecial, start)
			default:
				return nil, NewParseError(
					"expected letters or characters after + or -",
					start,
				)
			}
		case unicode.IsPunct(rune(s[i])):
			// Ignore other punctuation characters.
			i++
		default:
			return nil, NewParseError(
				fmt.Sprintf("unexpected character: %c", s[i]),
				start,
			)
		}
	}
	return ret, nil
}
