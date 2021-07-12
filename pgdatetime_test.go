package pgdatetime

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/datadriven"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	flag.Parse()

	datadriven.RunTest(t, "testdata/times", func(t *testing.T, d *datadriven.TestData) string {
		switch d.Cmd {
		case "test":
			fzp := "fixed offset"
			for _, arg := range d.CmdArgs {
				switch arg.Key {
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
				val, err2 := strconv.Atoi(inTZ)
				if err2 != nil {
					t.Fatalf("expected timezone offset or timezone name, found %s", inTZ)
				}
				tz = time.FixedZone(fzp, val)
			}
			tt, err := time.ParseInLocation("2006-01-02 15:04:05.999999", inTime, tz)
			require.NoError(t, err)

			retStr := ""
			for _, style := range []Style{StyleISO, StyleSQL} {
				for _, order := range []Order{OrderYMD, OrderDMY, OrderMDY} {
					retStr += fmt.Sprintf(
						"%s/%s: %s\n",
						style,
						order,
						Format(DateStyle{Style: style, Order: order, FixedZonePrefix: "fixed offset"}, tt),
					)
				}
			}
			return retStr
		default:
			t.Fatalf("command unknown: %s", d.Cmd)
		}
		return ""
	})
}