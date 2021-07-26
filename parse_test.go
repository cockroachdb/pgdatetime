package pgdatetime

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/datadriven"
	"github.com/stretchr/testify/require"
)

func TestTokenizeDateTime(t *testing.T) {
	datadriven.RunTest(t, "testdata/tokenize", func(t *testing.T, d *datadriven.TestData) string {
		switch d.Cmd {
		case "test":
			tokens, err := tokenizeDateTime(d.Input)
			require.NoError(t, err)

			ret := []string{}
			for _, token := range tokens {
				ret = append(
					ret,
					fmt.Sprintf("type: %s, val: %s, idx: %d", token.tokenType, token.val, token.idx),
				)
			}
			return strings.Join(ret, "\n")
		default:
			t.Fatalf("command unknown: %s", d.Cmd)
		}
		return ""
	})
}

func TestTokenizeDateTimeError(t *testing.T) {
	for _, tc := range []struct {
		s   string
		err error
	}{
		{"  +/", NewParseError(2, "expected letters or characters after + or -")},
	} {
		t.Run(tc.s, func(t *testing.T) {
			_, err := tokenizeDateTime(tc.s)
			require.Error(t, err)
			require.Equal(t, tc.err, err)
		})
	}
}

func TestParse(t *testing.T) {
	datadriven.RunTest(t, "testdata/parse", func(t *testing.T, d *datadriven.TestData) string {
		now := time.Date(2020, 06, 26, 15, 16, 17, 123456000, time.UTC)
		switch d.Cmd {
		case "timestamptz":
			dateStyle := DefaultDateStyle()
			for _, arg := range d.CmdArgs {
				switch strings.ToLower(arg.Key) {
				case "datestyle":
					var err error
					for _, val := range arg.Vals {
						dateStyle, err = ParseDateStyle(val, dateStyle)
						require.NoError(t, err)
					}
				default:
					t.Fatalf("unknown key: %s", arg.Key)
				}
			}
			r, err := ParseTimestampTZ(dateStyle, now, d.Input)
			require.NoError(t, err)
			return fmt.Sprintf("%s\n%s", r.Type.String(), Format(dateStyle, r.Time, true /* includeTimeZone */))
		default:
			t.Fatalf("command unknown: %s", d.Cmd)
		}
		return ""
	})
}
