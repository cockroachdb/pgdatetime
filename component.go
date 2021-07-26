package pgdatetime

// Component is a component of date or time.
type Component uint64

const (
	ComponentNumber Component = 1 << iota
	ComponentString

	ComponentDate
	ComponentTime
	ComponentTZ
	ComponentAgo

	ComponentSpecial
	ComponentEarly
	ComponentLate
	ComponentEpoch
	ComponentNow
	ComponentYesterday
	ComponentToday
	ComponentTomorrow
	ComponentZulu

	ComponentDelta
	ComponentSecond
	ComponentMinute
	ComponentHour
	ComponentDay
	ComponentWeek
	ComponentMonth
	ComponentQuarter
	ComponentYear
	ComponentDecade
	ComponentCentury
	ComponentMillennium
	ComponentMillis
	ComponentMicros
	ComponentJulian

	ComponentDOW
	ComponentDOY
	ComponentTZHour
	ComponentTZMinute
	ComponentISOYear
	ComponentISODOW

	ComponentTimeMask = (ComponentHour | ComponentMinute | ComponentSecond)
	ComponentDateMask = (ComponentDay | ComponentMonth | ComponentYear)
)
