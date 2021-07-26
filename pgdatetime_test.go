package pgdatetime

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/datadriven"
	"github.com/stretchr/testify/require"
)

// TestFormat tests formatting works, and when formatted, will re-parse
// to itself correctly.
func TestFormat(t *testing.T) {
	datadriven.RunTest(t, "testdata/format", func(t *testing.T, d *datadriven.TestData) string {
		switch d.Cmd {
		case "test":
			fz := "fixed offset"
			fzp := "fixed offset"
			for _, arg := range d.CmdArgs {
				switch arg.Key {
				case "fixed_zone":
					fz = arg.Vals[0]
				case "fixed_zone_prefix":
					fzp = arg.Vals[0]
				default:
					t.Fatalf("arg unknown for cmd %s: %s", d.Cmd, arg.Key)
				}
			}

			splitted := strings.Split(d.Input, "\n")
			if len(splitted) != 2 {
				t.Fatalf("expected two lines: one line with time, one line with time zone")
			}
			inTime, inTZ := splitted[0], splitted[1]

			tz, err := time.LoadLocation(inTZ)
			if err != nil {
				val, valErr := strconv.Atoi(inTZ)
				if valErr != nil {
					t.Fatalf("expected timezone offset or timezone name, found %s (tz err: %s, val err: %s)", inTZ, err, valErr)
				}
				tz = time.FixedZone(fz, val)
			}
			tt, err := time.ParseInLocation("2006-01-02 15:04:05.999999", inTime, tz)
			require.NoError(t, err)

			retStr := ""
			for _, itz := range []struct {
				prePrint string
				include  bool
			}{
				{"with time zones", true},
				{"no time zones", false},
			} {
				retStr += fmt.Sprintf("** %s **\n", itz.prePrint)
				for _, style := range []Style{StyleISO, StyleSQL, StyleGerman, StylePostgres} {
					for _, order := range []Order{OrderYMD, OrderDMY, OrderMDY} {
						retStr += fmt.Sprintf(
							"%s/%s: %s\n",
							style,
							order,
							Format(DateStyle{Style: style, Order: order, FixedZonePrefix: fzp}, tt, itz.include),
						)
					}
				}
			}
			return retStr
		default:
			t.Fatalf("command unknown: %s", d.Cmd)
		}
		return ""
	})
}
