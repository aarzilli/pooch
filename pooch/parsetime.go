/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
*/

package pooch

import (
	"fmt"
	"strings"
	"time"
)

type DateTimeFormat struct {
	format        string
	shouldFixYear bool // the parsed date expression didn't have a year field which will need to be calculated
	hasTime       bool // the parsed date expression had an explicit time
}

type VarTime struct {
	Year    int
	Month   int
	Day     int
	Weekday int
	Hour    int
	Minute  int
	Second  int
}

func (vt *VarTime) ToTime() time.Time {
	return time.Date(vt.Year, time.Month(vt.Month), vt.Day, vt.Hour, vt.Minute, vt.Second, 0, time.UTC)
}

func (vt *VarTime) ToTimePtr() *time.Time {
	t := vt.ToTime()
	return &t
}

func VarTimeFromTime(t time.Time) *VarTime {
	return &VarTime{t.Year(), int(t.Month()), t.Day(), int(t.Weekday()), t.Hour(), t.Minute(), t.Second()}
}

var DateTimeFormats []DateTimeFormat = []DateTimeFormat{
	{"2006-1-2 15:04:05", false, true},
	{"2006-1-2,15:04:05", false, true},
	{"2006-1-2 15:04", false, true},
	{"2006-1-2,15:04", false, true},

	{"2/1 15:04:05", true, true},
	{"2/1,15:04:05", true, true},
	{"2/1 15:04", true, true},
	{"2/1,15:04", true, true},
	{"2/1 15", true, true},
	{"2/1,15", true, true},

	{"2/1", true, false},
	{"2006-1-2", false, false},
}

var TimeOnlyFormats []DateTimeFormat = []DateTimeFormat{
	{"15:04:05", false, true},
	{"15:04", false, true},
}

func FixYear(datetime *VarTime, withTime bool) {
	format := "01-02"
	if withTime {
		format = "01-02 15:04"
	}

	if datetime.ToTime().Format(format) > time.Now().UTC().Format(format) {
		datetime.Year = time.Now().UTC().Year()
	} else {
		datetime.Year = time.Now().UTC().Year() + 1
	}
}

func fixDateEx(datetime *VarTime, ref *VarTime) {
	datetime.Year = ref.Year
	datetime.Month = ref.Month
	datetime.Day = ref.Day
	datetime.Weekday = ref.Weekday
}

func nextDay(atime time.Time) time.Time {
	return time.Unix(atime.Unix()+(24*60*60), 0)
}

func FixDate(datetime *VarTime) {
	ref := time.Now().UTC()
	if datetime.ToTime().Format("15:04:05") < ref.Format("15:04:05") {
		ref = nextDay(ref)
	}
	fixDateEx(datetime, VarTimeFromTime(ref))
}

func SearchDayOfTheWeek(datetime *VarTime) *VarTime {
	weekday := datetime.Weekday // thing to search
	fixDateEx(datetime, VarTimeFromTime(time.Now().UTC()))
	for count := 10; count > 0; count-- {
		datetime = VarTimeFromTime(nextDay(datetime.ToTime()))
		if datetime.Weekday == weekday {
			return datetime
		}
	}
	return datetime
}

func TimeParseTimezone(layout, input string, timezone int) (*VarTime, error) {
	t, err := time.Parse(layout, input)
	if err != nil {
		return nil, err
	}
	t = time.Unix(t.Unix()-(int64(timezone)*60*60), 0).In(time.FixedZone("fixed-zone", timezone*60))
	return VarTimeFromTime(t), nil
}

func timeParseLoop(input string, timezone int, formats []DateTimeFormat) *VarTime {
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

func parseNextWeekdayTime(input string, timezone int) *VarTime {
	var datetime *VarTime = &VarTime{}

	split := strings.SplitN(input, ",", 2)
	if len(split) > 1 {
		datetime = timeParseLoop(split[1], timezone, TimeOnlyFormats)
		if datetime == nil {
			return nil
		}
	}

	value, ok := weekdayConversion[split[0]]
	if !ok {
		return nil
	}
	datetime.Weekday = value
	return SearchDayOfTheWeek(datetime)
}

func ParseDateTime(input string, timezone int) (*time.Time, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, MakeParseError("Empty input")
	}

	if datetime := timeParseLoop(input, timezone, DateTimeFormats); datetime != nil {
		return datetime.ToTimePtr(), nil
	}

	if datetime := timeParseLoop(input, timezone, TimeOnlyFormats); datetime != nil {
		FixDate(datetime)
		return datetime.ToTimePtr(), nil
	}

	if datetime := parseNextWeekdayTime(input, timezone); datetime != nil {
		return datetime.ToTimePtr(), nil
	}

	return nil, MakeParseError(fmt.Sprintf("Unparsable date: %s", input))
}

func TimeFormatTimezone(atime *time.Time, format string, timezone int) string {
	z := time.Unix(atime.Unix()+(int64(timezone)*60*60), 0).In(time.FixedZone("fixed-zone", timezone*60))

	return z.Format(format)

}
