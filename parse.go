/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"os"
	"fmt"
	"strings"
	"container/vector"
	"time"
	"strconv"
)

var TRIGGER_AT_FORMAT string = "2006-01-02 15:04:05"
var TRIGGER_AT_SHORT_FORMAT string = "02/01 15:04"

func ParsePriority(s string) (p Priority, err string) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sticky": return STICKY, ""
	case "now": return NOW, ""
	case "later": return LATER, ""
	case "notes": return NOTES, ""
	case "timed": return TIMED, ""
	case "done": return DONE, ""
	}
	return INVALID, fmt.Sprintf("Unknown priority: ", s)
}

func ParseFrequency(s string) (freq Frequency, err string) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "daily": return Frequency(1), ""
	case "weekly": return Frequency(7), ""
	case "biweekly": return Frequency(14), ""
	case "monthly": return Frequency(30), ""
	case "yearly": return Frequency(365), ""
	case "": return Frequency(0), ""
	default:
		i, err := strconv.Atoi(s)
		if err == nil {
			return Frequency(i), ""
		}
	}

	return Frequency(0), fmt.Sprintf("Unparsable frequency: %s", s)
}

func FixYear(datetime *time.Time) {
	if datetime.Format("01-02") > time.LocalTime().Format("01-02") {
		datetime.Year = time.LocalTime().Year	
	} else {
		datetime.Year = time.LocalTime().Year+1
	}
}

func ParseDateTime(input string) (datetime *time.Time, error string) {
	//Date formats:
	// dd/mm
	// yyyy-mm-dd
	//Time formats:
	// hh:mm
	// hh:mm:ss

	datetime = nil
	error = ""

	var err os.Error
	input = strings.TrimSpace(input)

	if (input == "") {
		return
	}
	
	if (strings.Index(input, " ") != -1) { // has time
		if datetime, err = time.Parse("2006-1-2 15:04", input); err == nil {
			return
		}

		if datetime, err = time.Parse("2006-1-2 15:04:05", input); err == nil {
			return
		}

		if datetime, err = time.Parse("2/1 15:04", input); err == nil {
			FixYear(datetime)
			return
		}

		if datetime, err = time.Parse("2/1 15:04:05", input); err == nil {
			FixYear(datetime)			
			return
		}
	} else { // doesn't have time
		if datetime, err = time.Parse("2/1", input); err == nil {
			FixYear(datetime)
			return
		}

		if datetime, err = time.Parse("2006-1-2", input); err == nil {
			FixYear(datetime)
			return
		}
	}

	error = fmt.Sprintf("Unparsable date: %s", input)
	return
}

func ExtractQuickTag(input string) (string, int) {
	for i, ch := range input {
		if (ch == ' ') || (ch == '\t') || (ch == '\n') || (ch == '\r') {
			return input[0:i], i
		}
	}

	return input, len(input)
}

func SortFromTriggerAt(triggerAt *time.Time) string {
	if triggerAt != nil {
		return triggerAt.Format("2006-01-02")
	}
	
	return time.LocalTime().Format("2006-01-02")
}


/*
 Supported # syntax:

 #<date> - triggerAt set to <date> (using ParseDateTime and a split to space)
 #<date>+<recur> - triggerAt set to <date> plus recurring field, use ParseFrequency (and split with space)
 #l, #later - later
 #n, #now - now
 #d, #done - done
 #$, #N, #Notes - notes
 #$$, #StickyNotes - sticky notes
 */

func QuickParse(input string) (*Entry, *vector.StringVector) {
	lastEnd := 0
	r := ""
	errors := new(vector.StringVector)

	priority := NOW
	
	var freq Frequency = 0
	var triggerAt *time.Time = nil

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if ch != '#' {
			continue
		}

		r += input[lastEnd:i]

		quickTag, j := ExtractQuickTag(input[i+1:len(input)])
		i += j
		lastEnd = i+1

		removedASpace := false
		// skips a space if there are two contiguous
		if lastEnd < len(input) && input[lastEnd] == ' ' && len(r) > 0 && r[len(r)-1] == ' ' {
			removedASpace = true
			lastEnd++
		}


		switch quickTag {
		case "later", "l":
			Logf(DEBUG, "Found quickTag:[%s] -- later", quickTag)
			priority = LATER

		case "n", "now":
			Logf(DEBUG, "Found quickTag:[%s] -- now", quickTag)
			priority = NOW

		case "d", "done":
			Logf(DEBUG, "Found quickTag:[%s] -- done", quickTag)
			priority = DONE

		case "$", "N", "Notes":
			Logf(DEBUG, "Found quickTag:[%s] -- notes", quickTag)
			priority = NOTES

		case "$$", "StickyNotes":
			Logf(DEBUG, "Found quickTag:[%s] -- sticky", quickTag)
			priority = STICKY

		default:
			quickTagSplit := strings.Split(quickTag, "+", 2)
			triggerAt, _ = ParseDateTime(quickTagSplit[0])

			if (triggerAt == nil) {
				Logf(DEBUG, "Found quickTag:[%s] -- not a quickTag, discarding")
				// this is not a quickTag, leave it alone
				r += "#" + quickTag
				if removedASpace { r += " " }
			} else {
				priority = TIMED
				if (len(quickTagSplit) > 1) {
					freq, _ = ParseFrequency(quickTagSplit[1])
				}
				Logf(DEBUG, "Found quickTag:[%s] -- time: %v %v", quickTag, triggerAt, freq)
			}
		}
	}

	r += input[lastEnd:len(input)]

	r = strings.Trim(r, " \t\r\n\v")

	sort := SortFromTriggerAt(triggerAt)

	return MakeEntry("", r, "", priority, freq, triggerAt, sort), errors
}

func TimeString(triggerAt *time.Time, sort string) string {
	if triggerAt != nil {
		now := time.LocalTime()
		showYear := (triggerAt.Format("2006") != now.Format("2006"))
		showTime := (triggerAt.Format("15:04:05") != "00:00:00")

		var formatString string
		if showYear {
			formatString = "2006-01-02"
		} else {
			formatString = "02/01"
		}

		if showTime {
			formatString += " 15:04:05"
		}
		
		return "@ " + triggerAt.Format(formatString)
	} else {
		return sort
	}

	return ""
}

func DemarshalEntry(umentry *UnmarshalEntry) *Entry {
	triggerAt, err := ParseDateTime(umentry.TriggerAt)

	if err != "" {
		panic("demarshalling error: " + err)
	}
	
	sort := umentry.Sort

	if sort == "" {
		sort = SortFromTriggerAt(triggerAt)
	}

	freq, err := ParseFrequency(umentry.Freq)

	if err != "" {
		panic(err)
	}
	
	return MakeEntry(
		umentry.Id,
		umentry.Title,
		umentry.Text,
		umentry.Priority,
		freq,
		triggerAt,
		sort)
}

func MarshalEntry(entry *Entry) *UnmarshalEntry {
	triggerAt := entry.TriggerAt()
	triggerAtString := ""
	if triggerAt != nil {
		triggerAtString = triggerAt.Format(TRIGGER_AT_FORMAT)
	}

	freq := entry.Freq()
	
	return MakeUnmarshalEntry(
		entry.Id(),
		entry.Title(),
		entry.Text(),
		entry.Priority(),
		freq.String(),
		triggerAtString,
		entry.Sort(),
		"") // task list isn't watched on other side
}

func ToCalendarEvent(entry *Entry, className string) *CalendarEvent {
	r := CalendarEvent{}

	r.id = entry.Id()
	r.title = entry.Title()
	r.allDay = true
	//r.start = fmt.Sprintf("%d", entry.TriggerAt().Seconds() +10)
	r.start = entry.TriggerAt().Format(time.RFC3339)
	r.className = className

	return &r
}

func ParseTsvFormat(in string) *Entry {
	fields := strings.Split(in, "\t", 4)

	priority, err := ParsePriority(fields[2])
	if err != "" {
		panic(fmt.Sprintf("Error parsing tsv line: %s", err))
	}

	var triggerAt *time.Time = nil
	var sort string
	if priority == TIMED {
		var dterr string
		if triggerAt, dterr = ParseDateTime(fields[3]); dterr != "" {
			panic(fmt.Sprintf("Error parsing tsv line: %s", dterr))
		}
		sort = SortFromTriggerAt(triggerAt)
	} else {
		sort = fields[3]
	}

	return MakeEntry(
		fields[0], // id
		fields[1], // title
		"", // text
		priority,
		0,
		triggerAt,
		sort)
}
