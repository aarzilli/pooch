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
	"regexp"
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


func isQuickTagStart(ch uint8) bool {
	return ch == '#' || ch == '@'
}

func Strtok(input, chars string) []string {
	var r vector.StringVector
	i, lastIdx := 0, 0
	for {
		i = strings.IndexAny(input[lastIdx+1:len(input)], chars)
		if i == -1 { break }
		r.Push(input[lastIdx:lastIdx+i+1])
		lastIdx += i+1
	}

	r.Push(input[lastIdx:len(input)])

	return ([]string)(r)
}

var SanitizeRE *regexp.Regexp = regexp.MustCompile("[^a-zA-Z0-9.,/\\^!?]")

func SearchParseToken(input string) (op uint8, theselect string) {
	var r vector.StringVector
	
	op = input[0]
	//fmt.Printf("op: %c\n", op)
	for _, col := range Strtok(input[1:len(input)], "@#") {
		col = SanitizeRE.ReplaceAllString(col, "")
		r.Push(fmt.Sprintf("SELECT id FROM columns WHERE name = '%s'", col))
		//fmt.Printf("   col: [%s]\n", col)
	}
	return op, strings.Join(([]string)(r), " INTERSECT ")
}

func SearchParseSub(input string, ored, removed *vector.StringVector) {
	for _, token := range Strtok(input, "+-") {
		op, theselect := SearchParseToken(token)

		switch op {
		case '+': ored.Push(fmt.Sprintf("id IN (%s)", theselect))
		case '-': removed.Push(fmt.Sprintf("id IN (%s)", theselect))
		}
	}


}

func SearchParse(input string) (theselect, query string){
	lastEnd := 0
	r := ""
	
	var ored, removed vector.StringVector
	
	for i := 0; i < len(input); i++ {
		ch := input[i]

		addPlus := true
		if (i+1 < len(input)) && ((ch == '+') || (ch == '-')) {
			addPlus = false
			if !isQuickTagStart(input[i+1]) { continue }
		} else if !isQuickTagStart(ch) {
			continue
		}

		r += input[lastEnd:i]

		quickTag, j := ExtractQuickTag(input[i:len(input)])
		i += j
		lastEnd = i+1

		if addPlus { quickTag = "+" + quickTag }

		//fmt.Printf("QuickTagParam: [%s]\n", quickTag)

		SearchParseSub(quickTag, &ored, &removed)
	}

	oredStr := strings.Join(([]string)(ored), " OR ")
	removedStr := strings.Join(([]string)(removed), " OR ")

	var tagSelectStr string
	if removed.Len() != 0 {
		tagSelectStr = fmt.Sprintf("(%s AND NOT (%s))", oredStr, removedStr)
	} else {
		tagSelectStr = fmt.Sprintf("(%s)", oredStr)
	}


	if lastEnd < len(input) {
		r += input[lastEnd:len(input)]
	}

	r = strings.Trim(r, " \t\r\n\v")

	matchClause := ""
	if r != "" { matchClause = "id IN (SELECT id FROM ridx WHERE title_field MATCH ? UNION SELECT id FROM ridx WHERE text_field MATCH ?) AND" }

	return fmt.Sprintf("SELECT tasks.id, tasks.title_field, tasks.text_field, tasks.priority, tasks.repeat_field, tasks.trigger_at_field, tasks.sort, group_concat(columns.name||':'||columns.value, '\n') FROM tasks NATURAL JOIN columns WHERE %s %s GROUP BY tasks.id ORDER BY priority, sort ASC", matchClause, tagSelectStr), r
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
	cols := make(Columns)

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if !isQuickTagStart(ch) { continue }

		r += input[lastEnd:i]

		quickTag, j := ExtractQuickTag(input[i+1:len(input)])
		i += j
		lastEnd = i+1

		//removedASpace := false
		// skips a space if there are two contiguous
		if lastEnd < len(input) && input[lastEnd] == ' ' && r[len(r)-1] == ' ' {
			//removedASpace = true
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
				Logf(DEBUG, "Found quickTag:[%s] -- no special meaning found, using it as a category", quickTag)
				cols[quickTag] = ""
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

	return MakeEntry("", r, "", priority, freq, triggerAt, sort, cols), errors
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

	cols := make(Columns)
	//TODO: Demarshalling of columns (must parse)
	
	return MakeEntry(
		umentry.Id,
		umentry.Title,
		umentry.Text,
		umentry.Priority,
		freq,
		triggerAt,
		sort,
		cols)
}

func MarshalEntry(entry *Entry) *UnmarshalEntry {
	triggerAt := entry.TriggerAt()
	triggerAtString := ""
	if triggerAt != nil {
		triggerAtString = triggerAt.Format(TRIGGER_AT_FORMAT)
	}

	freq := entry.Freq()

	//TODO: Marshalling of columns (must deparse)
	
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

	cols := make(Columns)

	// TODO: parsing columns from tsv

	return MakeEntry(
		fields[0], // id
		fields[1], // title
		"", // text
		priority,
		0,
		triggerAt,
		sort,
		cols)
}
