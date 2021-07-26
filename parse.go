package pgdatetime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

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

// token represents a date token Component of a datetime.
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
	appendToken := func(t tokenType, start int) {
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
				appendToken(tokenTypeNumber, start)
				break
			}
			switch s[i] {
			case ':':
				// It is a time element.
				// Read the remaining time characters.
				advanceWhen(func(b byte) bool {
					return isDigit(b) || b == ':' || b == '.'
				})
				appendToken(tokenTypeTime, start)
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
					appendToken(t, start)
					break
				}
				// This could be a date with text, e.g. 13/Feb/2021.
				advanceWhen(func(b byte) bool {
					return isLetterOrDigit(b) || b == delimiter
				})
				appendToken(tokenTypeDate, start)
			default:
				appendToken(tokenTypeNumber, start)
			}
		case s[i] == '.':
			// Fractional seconds.
			i++
			advanceWhen(isDigit)
			appendToken(tokenTypeNumber, start)
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
			appendToken(t, start)
		case s[i] == '+' || s[i] == '-':
			// Timezone or special.
			i++
			advanceWhen(func(b byte) bool {
				return unicode.IsSpace(rune(b))
			})
			if i == len(s) {
				return nil, NewParseError(start, "expected letters or characters after + or -")
			}
			switch {
			case isDigit(s[i]):
				advanceWhen(func(b byte) bool {
					return isDigit(b) || b == ':' || b == '.'
				})
				appendToken(tokenTypeTZ, start)
			case isLetter(s[i]):
				advanceWhen(isLetter)
				appendToken(tokenTypeSpecial, start)
			default:
				return nil, NewParseError(start, "expected letters or characters after + or -")
			}
		case unicode.IsPunct(rune(s[i])):
			// Ignore other punctuation characters.
			i++
		default:
			return nil, NewParseError(start, fmt.Sprintf("unexpected character: %c", s[i]))
		}
	}
	return ret, nil
}

type decodeTokenState struct {
	seen                        Component
	year, month, day            int
	hour, minute, second, nanos int
	loc                         *time.Location
	typ                         ParseResultType
	is2DigitYear                bool

	dateStyle DateStyle
	now       time.Time
}

func (s *decodeTokenState) hasSeen(c Component) bool {
	return (s.seen & c) == c
}

func (s *decodeTokenState) markSeen(c Component) {
	s.seen |= c
}

func (s *decodeTokenState) decodeDate(t token) error {
	if s.hasSeen(ComponentMonth | ComponentDay) {
		// If we've already seen the month and day, this could be a timezone.
		return nil
	}

	delimiterIdx := strings.IndexAny(t.val, "/-.")
	if delimiterIdx == -1 {
		return NewParseError(t.idx, "expected date separator but found none")
	}
	fields := strings.Split(t.val, t.val[delimiterIdx:delimiterIdx+1])
	// TODO: text month
	currLen := 0
	for _, field := range fields {
		if err := s.decodeNumber(token{val: field, idx: t.idx + currLen}); err != nil {
			return err
		}
		currLen += len(field) + 1
	}
	return nil
}

func (s *decodeTokenState) decodeNumber(t token) error {
	i := 0
	num, err := s.readDigits(t, &i)
	if err != nil {
		return err
	}
	// TODO: decimal point
	// TODO: day of year
	var seenMask Component
	switch s.seen & ComponentDateMask {
	case 0:
		// We have not seen day, month or year.
		if len(t.val) >= 3 || s.dateStyle.Order == OrderYMD {
			// If it is 3 digits long, or YMD, assume it is a year.
			seenMask |= ComponentYear
			s.year = num
		} else if s.dateStyle.Order == OrderDMY {
			seenMask |= ComponentDay
			s.day = num
		} else {
			seenMask |= ComponentMonth
			s.month = num
		}
	case ComponentYear:
		// If we've seen year, we're assuming MM of YYYY-MM-DD.
		seenMask |= ComponentMonth
		s.month = num
	case ComponentMonth:
		// TODO: check text month
		// Must be at second field of MM-DD-YYYY
		seenMask |= ComponentDay
		s.day = num
	case ComponentYear | ComponentMonth:
		// TODO: check text month
		// Must be at third field of YYYY-MM-DD.
		seenMask |= ComponentDay
		s.day = num
	case ComponentDay:
		// Must be at second field of DD-MM-YYYY.
		seenMask |= ComponentMonth
		s.month = num
	case ComponentDay | ComponentMonth:
		// Must be at third field of DD-MM-YYYY or MM-DD-YYYY.
		seenMask |= ComponentYear
		s.year = num
	case ComponentDay | ComponentMonth | ComponentYear:
		// TODO: have all three so it is time related.
	}
	if len(t.val) == 2 && seenMask == ComponentYear {
		s.is2DigitYear = true
	}
	s.seen |= seenMask
	return nil
}

func (s *decodeTokenState) readDigits(t token, i *int) (int, error) {
	start := *i
	for *i < len(t.val) && unicode.IsDigit(rune(t.val[*i])) {
		*i++
	}
	ret, err := strconv.ParseInt(t.val[start:*i], 10, 64)
	if err != nil {
		return 0, NewParseErrorf(t.idx+*i+start, "error parsing digits: %s", err.Error())
	}
	return int(ret), nil
}

func (s *decodeTokenState) decodeTime(t token) error {
	i := 0
	if s.hasSeen(ComponentTimeMask) {
		return NewParseErrorf(t.idx, "duplicate time Component: %s", t.val)
	}
	s.markSeen(ComponentTimeMask)
	var err error
	// Read hour.
	s.hour, err = s.readDigits(t, &i)
	if err != nil {
		return err
	}
	// Ensure we have a ':' separator.
	if t.val[i] != ':' {
		return NewParseErrorf(t.idx+i, "expected :, got %c", t.val[i])
	}
	i++
	// Read minutes.
	s.minute, err = s.readDigits(t, &i)
	if err != nil {
		return err
	}

	// End of t.valing, that's ok.
	if i == len(t.val) {
		return nil
	}

	// Check for seconds and fractional seconds,
	switch t.val[i] {
	case ':':
		i++
		if i == len(t.val) {
			return NewParseErrorf(t.idx+i, "expected digits but none found")
		}
		s.second, err = s.readDigits(t, &i)
		if err != nil {
			return err
		}
		if i == len(t.val) {
			return nil
		}
		if t.val[i] == '.' {
			return s.decodeFractionalSecond(token{val: t.val[i:], idx: t.idx + i})
		}
		return NewParseErrorf(t.idx+i, "expected ., found %c", t.val[i])
	case '.':
		return s.decodeFractionalSecond(token{val: t.val[i:], idx: t.idx + i})
	}
	return NewParseErrorf(t.idx+i, "expected : or ., found %c", t.val[i])
}

func (s *decodeTokenState) decodeFractionalSecond(t token) error {
	if len(t.val) == 0 {
		return NewParseError(t.idx, "expected fractional second, found empty string")
	}
	if t.val[0] != '.' {
		return NewParseErrorf(t.idx, "expected ., found %c", t.val[0])
	}
	s.markSeen(ComponentMicros)
	micros, err := strconv.ParseInt(t.val[1:], 10, 64)
	if err != nil {
		return NewParseErrorf(t.idx+1, "error parsing digits: %s", err.Error())
	}
	s.nanos = int(micros) * int(time.Microsecond)
	return nil
}

func decodeTokens(dateStyle DateStyle, now time.Time, tokens []token) (ParseResult, error) {
	s := decodeTokenState{
		typ:       ParseResultTypeAbsoluteTime,
		dateStyle: dateStyle,
		now:       now,
		loc:       now.Location(),
	}

	for _, t := range tokens {
		switch t.tokenType {
		case tokenTypeDate:
			// Julian?
			if err := s.decodeDate(t); err != nil {
				return ParseResult{}, err
			}
		case tokenTypeTime:
			if err := s.decodeTime(t); err != nil {
				return ParseResult{}, err
			}
		default:
			return ParseResult{}, NewParseErrorf(t.idx, "unknown token type %s", t.tokenType.String())
		}
	}
	return ParseResult{
		Type: s.typ,
		Time: time.Date(
			s.year,
			time.Month(s.month),
			s.day,
			s.hour,
			s.minute,
			s.second,
			s.nanos,
			s.loc,
		),
	}, nil
}
