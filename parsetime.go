package main

import (
	"time"
	"os"
	"strings"
	"fmt"
)

type DateTimeFormat struct {
	format string
	shouldFixYear bool // the parsed date expression didn't have a year field which will need to be calculated
	hasTime bool // the parsed date expression had an explicit time
	shouldFixDate bool // the parsed time expression didn't have a date at all (just hour) the date field should be set to today or tomorrow

}

var DateTimeFormats []DateTimeFormat = []DateTimeFormat{
	{ "2006-1-2 15:04:05", false, true, false },
	{ "2006-1-2,15:04:05", false, true, false },
	{ "2006-1-2 15:04", false, true, false },
	{ "2006-1-2,15:04", false, true, false },
	
	{ "2/1 15:04:05", true, true, false },
	{ "2/1,15:04:05", true, true, false },
	{ "2/1 15:04", true, true, false },
	{ "2/1,15:04", true, true, false },
	{ "2/1 15", true, true, false },
	{ "2/1,15", true, true, false },
	
	{ "2/1", true, false, false },
	{ "2006-1-2", false, false, false },
	
	{ "15:04:05", false, false, true },
	{ "15:04", false, false, true },
}

func FixYear(datetime *time.Time, withTime bool) {
	format := "01-02"
	if withTime { format = "01-02 15:04" }

	if datetime.Format(format) > time.UTC().Format(format) {
		datetime.Year = time.UTC().Year	
	} else {
		datetime.Year = time.UTC().Year+1
	}
}

func FixDate(datetime *time.Time) {
	ref := time.UTC()
	if datetime.Format("15:04:05") < ref.Format("15:04:05") {
		ref = time.SecondsToUTC(ref.Seconds() + (24 * 60 * 60))
	}
	datetime.Year = ref.Year
	datetime.Month = ref.Month
	datetime.Day = ref.Day
	datetime.Weekday = ref.Weekday
}

func TimeParseTimezone(layout, input string, timezone int) (*time.Time, os.Error) {
	t, err := time.Parse(layout, input)
	if err != nil { return nil, err }
	t = time.SecondsToUTC(t.Seconds() - (int64(timezone) * 60 * 60))
	t.ZoneOffset = timezone * 60
	return t, nil
}

func ParseDateTime(input string, timezone int) (datetime *time.Time, error os.Error) {
	datetime = nil
	error = nil

	var err os.Error
	input = strings.TrimSpace(input)

	if (input == "") {
		return
	}

	for _, dateTimeFormat := range DateTimeFormats {
		if datetime, err = TimeParseTimezone(dateTimeFormat.format, input, timezone); err == nil {
			if dateTimeFormat.shouldFixYear {
				FixYear(datetime, dateTimeFormat.hasTime)
			}
			if dateTimeFormat.shouldFixDate {
				FixDate(datetime)
			}
			return
		}
	}

	error = MakeParseError(fmt.Sprintf("Unparsable date: %s", input))
	return
}

func TimeFormatTimezone(atime *time.Time, format string, timezone int) string {
	z := time.SecondsToUTC(atime.Seconds() + (int64(timezone) * 60 * 60))
	z.ZoneOffset = timezone * 60
	
	return z.Format(format)

}
