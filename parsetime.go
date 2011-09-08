/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

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
}

var DateTimeFormats []DateTimeFormat = []DateTimeFormat{
	{ "2006-1-2 15:04:05", false, true },
	{ "2006-1-2,15:04:05", false, true },
	{ "2006-1-2 15:04", false, true },
	{ "2006-1-2,15:04", false, true },
	
	{ "2/1 15:04:05", true, true },
	{ "2/1,15:04:05", true, true },
	{ "2/1 15:04", true, true },
	{ "2/1,15:04", true, true },
	{ "2/1 15", true, true },
	{ "2/1,15", true, true },
	
	{ "2/1", true, false },
	{ "2006-1-2", false, false },
	
}

var TimeOnlyFormats []DateTimeFormat = []DateTimeFormat{
	{ "15:04:05", false, true },
	{ "15:04", false, true },
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

func fixDateEx(datetime, ref *time.Time) {
	datetime.Year = ref.Year
	datetime.Month = ref.Month
	datetime.Day = ref.Day
	datetime.Weekday = ref.Weekday
}

func nextDay(atime *time.Time) *time.Time {
	return time.SecondsToUTC(atime.Seconds() + (24 * 60 * 60))
}

func FixDate(datetime *time.Time) {
	ref := time.UTC()
	if datetime.Format("15:04:05") < ref.Format("15:04:05") {
		ref = nextDay(ref)
	}
	fixDateEx(datetime, ref)
}

func SearchDayOfTheWeek(datetime *time.Time) *time.Time {
	weekday := datetime.Weekday // thing to search
	fixDateEx(datetime, time.UTC())
	for count := 10; count > 0; count-- {
		datetime = nextDay(datetime)
		if datetime.Weekday == weekday {
			return datetime
		}
	}
	return datetime
}

func TimeParseTimezone(layout, input string, timezone int) (*time.Time, os.Error) {
	t, err := time.Parse(layout, input)
	if err != nil { return nil, err }
	t = time.SecondsToUTC(t.Seconds() - (int64(timezone) * 60 * 60))
	t.ZoneOffset = timezone * 60
	return t, nil
}

func timeParseLoop(input string, timezone int, formats []DateTimeFormat) *time.Time {
	for _, dateTimeFormat := range formats {
		if datetime, err := TimeParseTimezone(dateTimeFormat.format, input, timezone); err == nil {
			if dateTimeFormat.shouldFixYear {
				FixYear(datetime, dateTimeFormat.hasTime)
			}
			return datetime
		}
	}

	return nil
}

var weekdayConversion map[string]int = map[string]int{
	"Mon": 1, "mon": 1,
	"Tue": 2, "tue": 2,
	"Wed": 3, "wed": 3,
	"Thu": 4, "thu": 4,
	"Fri": 5, "fri": 5,
	"Sat": 6, "sat": 6,
	"Sun": 0, "sun": 0,
}

func parseNextWeekdayTime(input string, timezone int) *time.Time {
	var datetime *time.Time = &time.Time{}
	
	split := strings.SplitN(input, ",", 2)
	if len(split) > 1 {
		datetime = timeParseLoop(split[1], timezone, TimeOnlyFormats)
		if datetime == nil { return nil }
	}

	value, ok := weekdayConversion[split[0]]
	if !ok { return nil }
	datetime.Weekday = value
	return SearchDayOfTheWeek(datetime)
}


func ParseDateTime(input string, timezone int) (*time.Time, os.Error) {
	input = strings.TrimSpace(input)

	if (input == "") { return nil, MakeParseError("Empty input") }

	if datetime := timeParseLoop(input, timezone, DateTimeFormats); datetime != nil {
		return datetime, nil
	}

	if datetime := timeParseLoop(input, timezone, TimeOnlyFormats); datetime != nil {
		FixDate(datetime)
		return datetime, nil
	}

	if datetime := parseNextWeekdayTime(input, timezone); datetime != nil {
		return datetime, nil
	}

	return nil, MakeParseError(fmt.Sprintf("Unparsable date: %s", input))
}

func TimeFormatTimezone(atime *time.Time, format string, timezone int) string {
	z := time.SecondsToUTC(atime.Seconds() + (int64(timezone) * 60 * 60))
	z.ZoneOffset = timezone * 60
	
	return z.Format(format)

}
