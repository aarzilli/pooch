package main

import (
	"time"
	"os"
	"strings"
	"fmt"
)

type DateTimeFormat struct {
	format string
	shouldFixYear bool
	hasTime bool
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

func FixYear(datetime *time.Time, withTime bool) {
	format := "01-02"
	if withTime { format = "01-02 15:04" }

	if datetime.Format(format) > time.UTC().Format(format) {
		datetime.Year = time.UTC().Year	
	} else {
		datetime.Year = time.UTC().Year+1
	}
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
			return
		}
	}
	
	error = MakeParseError(fmt.Sprintf("Unparsable date: %s", input))
	return
}
