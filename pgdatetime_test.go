package pgdatetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseAndFormat(t *testing.T) {
	california, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	adelaide, err := time.LoadLocation("Australia/Adelaide")
	require.NoError(t, err)
	negativeSecondOffset := time.FixedZone("negative land", -(7*60*60 + 15*60 + 8))
	positiveSecondOffset := time.FixedZone("positive land", (7*60*60 + 15*60 + 8))

	testCases := []struct {
		desc                   string
		time                   time.Time
		isoDMY, isoMDY, isoYMD string
	}{
		{
			desc:   "california time",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 123456000, california),
			isoDMY: "25-12-2015 15:30:45.123456-08",
			isoMDY: "12-25-2015 15:30:45.123456-08",
			isoYMD: "2015-12-25 15:30:45.123456-08",
		},
		{
			desc:   "california time, less milliseconds",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 120400000, california),
			isoDMY: "25-12-2015 15:30:45.1204-08",
			isoMDY: "12-25-2015 15:30:45.1204-08",
			isoYMD: "2015-12-25 15:30:45.1204-08",
		},
		{
			desc:   "california time, no milliseconds",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 0, california),
			isoDMY: "25-12-2015 15:30:45-08",
			isoMDY: "12-25-2015 15:30:45-08",
			isoYMD: "2015-12-25 15:30:45-08",
		},
		{
			desc:   "utc time",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 123456000, time.UTC),
			isoDMY: "25-12-2015 15:30:45.123456+00",
			isoMDY: "12-25-2015 15:30:45.123456+00",
			isoYMD: "2015-12-25 15:30:45.123456+00",
		},
		{
			desc:   "adelaide time",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 123456000, adelaide),
			isoDMY: "25-12-2015 15:30:45.123456+10:30",
			isoMDY: "12-25-2015 15:30:45.123456+10:30",
			isoYMD: "2015-12-25 15:30:45.123456+10:30",
		},
		{
			desc:   "negative land time",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 123456000, negativeSecondOffset),
			isoDMY: "25-12-2015 15:30:45.123456-07:15:08",
			isoMDY: "12-25-2015 15:30:45.123456-07:15:08",
			isoYMD: "2015-12-25 15:30:45.123456-07:15:08",
		},
		{
			desc:   "positive land time",
			time:   time.Date(2015, 12, 25, 15, 30, 45, 123456000, positiveSecondOffset),
			isoDMY: "25-12-2015 15:30:45.123456+07:15:08",
			isoMDY: "12-25-2015 15:30:45.123456+07:15:08",
			isoYMD: "2015-12-25 15:30:45.123456+07:15:08",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			for _, it := range []struct {
				order    Order
				expected string
			}{
				{order: OrderMDY, expected: tc.isoMDY},
				{order: OrderDMY, expected: tc.isoDMY},
				{order: OrderYMD, expected: tc.isoYMD},
			} {
				t.Run(it.order.String(), func(t *testing.T) {
					require.Equal(t, it.expected, Format(DateStyle{Style: StyleISO, Order: it.order}, tc.time))
				})
			}
		})
	}
}
