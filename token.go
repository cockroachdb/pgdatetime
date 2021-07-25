package pgdatetime

import (
	"fmt"
	"strings"
	"unicode"
)

// tokenType maps to a DTK object in PostgreSQL.
type tokenType int

//go:generate stringer -type=tokenType -trimprefix=tokenType

const (
	// tokenTypeNumber can hold dates, e.g. (yy.dd)
	tokenTypeNumber tokenType = iota
	// tokenTypeString can hold months (e.g. January) or time zones (e.g. PST)
	tokenTypeString

	// tokenTypeDate can hold time zones (e.g. GMT-8)
	tokenTypeDate
	tokenTypeTime
	tokenTypeTZ

	tokenTypeSpecial
)

// token represents a date token component of a datetime.
type token struct {
	tokenType tokenType
	val       string
	idx       int
}

func tokenizeDateTime(s string) ([]token, error) {
	s = strings.ToLower(s)
	i := 0
	ret := []token{}
	isDigit := func(b byte) bool {
		return unicode.IsDigit(rune(b))
	}
	isLetter := func(b byte) bool {
		return unicode.IsLetter(rune(b))
	}
	isLetterOrDigit := func(b byte) bool {
		return unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
	}
	advanceWhen := func(f func(b byte) bool) {
		for i < len(s) && f(s[i]) {
			i++
		}
	}
	appendDateToken := func(t tokenType, start int) {
		ret = append(
			ret,
			token{tokenType: t, val: s[start:i], idx: start},
		)
	}

	for i < len(s) {
		start := i

		// Read all digits.
		switch {
		case unicode.IsSpace(rune(s[i])):
			// Ignore spaces.
			advanceWhen(func(b byte) bool {
				return unicode.IsSpace(rune(b))
			})
		case isDigit(s[i]):
			// Starting with a digit.
			advanceWhen(isDigit)
			// If we've reached the end, treat it as a number.
			if i >= len(s) {
				appendDateToken(tokenTypeNumber, start)
				break
			}
			switch s[i] {
			case ':':
				// It is a time element.
				// Read the remaining time characters.
				advanceWhen(func(b byte) bool {
					return isDigit(b) || b == ':' || b == '.'
				})
				appendDateToken(tokenTypeTime, start)
			case '-', '/', '.':
				// It is a date element.
				// Mark the delimiter.
				delimiter := s[i]
				i++
				if i < len(s) && isDigit(s[i]) {
					// Read the second set of digits if any.
					advanceWhen(isDigit)
					// If it's two fields and separated by a '.', treat it as a number.
					// Otherwise, treat two or three fields as a date.
					t := tokenTypeDate
					if delimiter == '.' {
						t = tokenTypeNumber
					}
					if i < len(s) && s[i] == delimiter {
						t = tokenTypeDate
						advanceWhen(func(b byte) bool {
							return isDigit(b) || b == delimiter
						})
					}
					appendDateToken(t, start)
					break
				}
				// This could be a date with text, e.g. 13/Feb/2021.
				advanceWhen(func(b byte) bool {
					return isLetterOrDigit(b) || b == delimiter
				})
				appendDateToken(tokenTypeDate, start)
			default:
				appendDateToken(tokenTypeNumber, start)
			}
		case s[i] == '.':
			// Fractional seconds.
			i++
			advanceWhen(isDigit)
			appendDateToken(tokenTypeNumber, start)
		case isLetter(s[i]):
			// Text - could be date string, month, DOW, special or timezone.
			advanceWhen(isLetter)

			t := tokenTypeString
			// Could be a date with a leading text month.
			if i < len(s) && (s[i] == '-' || s[i] == '/' || s[i] == '.') {
				delimiter := s[i]
				advanceWhen(func(b byte) bool {
					return isDigit(b) || b == delimiter
				})
				t = tokenTypeDate
			}
			appendDateToken(t, start)
		case s[i] == '+' || s[i] == '-':
			// Timezone or special.
			i++
			advanceWhen(func(b byte) bool {
				return unicode.IsSpace(rune(b))
			})
			if i == len(s) {
				return nil, NewParseError(
					"expected letters or characters after + or -",
					start,
				)
			}
			switch {
			case isDigit(s[i]):
				advanceWhen(func(b byte) bool {
					return isDigit(b) || b == ':' || b == '.'
				})
				appendDateToken(tokenTypeTZ, start)
			case isLetter(s[i]):
				advanceWhen(isLetter)
				appendDateToken(tokenTypeSpecial, start)
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
