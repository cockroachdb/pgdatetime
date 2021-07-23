package pgdatetime

import (
	"fmt"
	"strings"
	"testing"

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
					fmt.Sprintf("type: %s, val: %s, idx: %d", token.dateTokenType, token.val, token.idx),
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
		{"  +/", NewParseError("expected letters or characters after + or -", 2)},
	} {
		t.Run(tc.s, func(t *testing.T) {
			_, err := tokenizeDateTime(tc.s)
			require.Error(t, err)
			require.Equal(t, tc.err, err)
		})
	}
}
