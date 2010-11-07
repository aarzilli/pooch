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
	"bufio"
	"tabwriter"
)

var TRIGGER_AT_FORMAT string = "2006-01-02 15:04"
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
	
	if (strings.Index(input, " ") != -1) || (strings.Index(input, ",") != -1){ // has time
		if datetime, err = time.Parse("2006-1-2 15:04", input); err == nil { return }
		if datetime, err = time.Parse("2006-1-2,15:04", input); err == nil { return }

		if datetime, err = time.Parse("2006-1-2 15:04:05", input); err == nil { return }
		if datetime, err = time.Parse("2006-1-2,15:04:05", input); err == nil { return }

		if datetime, err = time.Parse("2/1 15:04:05", input); err == nil { FixYear(datetime); return }
		if datetime, err = time.Parse("2/1,15:04:05", input); err == nil { FixYear(datetime); return }

		if datetime, err = time.Parse("2/1 15:04", input); err == nil { FixYear(datetime); return }
		if datetime, err = time.Parse("2/1,15:04", input); err == nil { FixYear(datetime); return }

		if datetime, err = time.Parse("2/1 15", input); err == nil { FixYear(datetime); return }
		if datetime, err = time.Parse("2/1,15", input); err == nil { FixYear(datetime); return }
	} else { // doesn't have time
		if datetime, err = time.Parse("2/1", input); err == nil { FixYear(datetime); return }

		if datetime, err = time.Parse("2006-1-2", input); err == nil { FixYear(datetime); return }
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
	r := make([]string, 0)
	i, lastIdx := 0, 0
	for {
		i = strings.IndexAny(input[lastIdx+1:len(input)], chars)
		if i == -1 { break }
		r = append(r, input[lastIdx:lastIdx+i+1])
		lastIdx += i+1
	}

	return append(r, input[lastIdx:len(input)])
}

var SanitizeRE *regexp.Regexp = regexp.MustCompile("[^a-zA-Z0-9.,/\\^!?]")

func SearchParseToken(input string, tl *Tasklist, set map[string]string, guessParse bool) (op uint8, theselect string) {
	var r vector.StringVector
	
	op = input[0]
	//fmt.Printf("op: %c\n", op)
	for _, col := range Strtok(input[1:len(input)], "@#") {
		colsplit := strings.Split(col, "=", 2)

		if len(colsplit) == 1 {
			col = SanitizeRE.ReplaceAllString(col, "")
			if guessParse {
				r.Push(col)
			} else {
				r.Push(fmt.Sprintf("SELECT id FROM columns WHERE name = '%s'", col))
			}
			
			if set != nil { set[col] = "" }
		} else {
			colsplit[0] = SanitizeRE.ReplaceAllString(colsplit[0], "")
			if !guessParse {
				r.Push(fmt.Sprintf("SELECT id FROM columns WHERE name = '%s' AND value = %s", colsplit[0], tl.Quote(colsplit[1])))
			}
			if set != nil { set[colsplit[0]] = colsplit[1] }
		}
		
		//fmt.Printf("   col: [%s]\n", col)
	}
	
	if guessParse {
		return op, strings.Join(([]string)(r), "\t")
	}
	
	return op, strings.Join(([]string)(r), " INTERSECT ")
}

func SearchParseSub(tl *Tasklist, input string, ored, removed *vector.StringVector, guessParse bool) {
	for _, token := range Strtok(input, "+-") {
		op, theselect := SearchParseToken(token, tl, nil, guessParse)

		switch op {
		case '+':
			if guessParse {
				ored.Push(theselect)
			} else {
				ored.Push(fmt.Sprintf("id IN (%s)", theselect))
			}
		case '-': removed.Push(fmt.Sprintf("id IN (%s)", theselect))
		}
	}
}

func SearchParse(input string, wantsDone, guessParse bool, extraWhereClauses []string, tl *Tasklist) (theselect, query string){
	if (len(input) > 2) && (input[0:2] == "@%") {
		name := input[2:len(input)]
		search := tl.GetSavedSearch(name)
		Logf(DEBUG, "Retrieving saved query: %s [%s]\n", name, search)
		return SearchParse(search, wantsDone, guessParse, extraWhereClauses, tl)
	}
	
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

		SearchParseSub(tl, quickTag, &ored, &removed, guessParse)

		if guessParse && (ored.Len() > 0) {
			return ored.At(0), ""
		}
	}

	if guessParse {
		return "", ""
	}

	oredStr := strings.Join(([]string)(ored), " OR ")
	removedStr := strings.Join(([]string)(removed), " OR ")

	var whereClauses vector.StringVector

	for _, v := range extraWhereClauses {
		whereClauses.Push(v)
	}

	if removed.Len() != 0 && ored.Len() != 0{
		whereClauses.Push(fmt.Sprintf("(%s AND NOT (%s))", oredStr, removedStr))
	} else if removed.Len() != 0 {
		whereClauses.Push(fmt.Sprintf("(NOT (%s))", removedStr))
	} else if ored.Len() != 0 {
		whereClauses.Push(fmt.Sprintf("(%s)", oredStr))
	} 

	if lastEnd < len(input) {
		r += input[lastEnd:len(input)]
	}

	r = strings.Trim(r, " \t\r\n\v")

	if r != "" { whereClauses.Push("id IN (SELECT id FROM ridx WHERE title_field MATCH ? UNION SELECT id FROM ridx WHERE text_field MATCH ?)") }

	if !wantsDone { whereClauses.Push("tasks.priority <> 5") }

	whereClause := ""
	if whereClauses.Len() != 0 { whereClause = " WHERE " + strings.Join(([]string)(whereClauses), " AND ") }
	
	return fmt.Sprintf("SELECT tasks.id, tasks.title_field, tasks.text_field, tasks.priority, tasks.repeat_field, tasks.trigger_at_field, tasks.sort, group_concat(columns.name||':'||columns.value, '\n') FROM tasks NATURAL JOIN columns %s GROUP BY tasks.id ORDER BY priority, trigger_at_field ASC, sort DESC", whereClause), r
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

 The query string will be utilized to guess categories to associate with the input string
 */

func QuickParse(input string, query string, tl *Tasklist) (*Entry, *vector.StringVector) {
	lastEnd := 0
	r := ""
	errors := new(vector.StringVector)

	priority := NOW
	
	var freq Frequency = 0
	var triggerAt *time.Time = nil
	cols := make(Columns)

	catfound := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if !isQuickTagStart(ch) { continue }

		r += input[lastEnd:i]

		quickTag, j := ExtractQuickTag(input[i+1:len(input)])
		i += j
		lastEnd = i+1

		//removedASpace := false
		// skips a space if there are two contiguous
		if lastEnd < len(input) && input[lastEnd] == ' ' && len(r) > 0 && r[len(r)-1] == ' ' {
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
				catfound = true;
			} else {
				priority = TIMED
				if (len(quickTagSplit) > 1) {
					freq, _ = ParseFrequency(quickTagSplit[1])
				}
				Logf(DEBUG, "Found quickTag:[%s] -- time: %v %v", quickTag, triggerAt, freq)
			}
		}
	}

	if tl != nil {
		extraCats, _ := SearchParse(query, false, true, nil, tl)

		if extraCats != "" {
			Logf(DEBUG, "Extra categories: %s\n", extraCats)
			
			for _, extraCat := range strings.Split(extraCats, "\t", -1) {
				cols[extraCat] = ""
				catfound = true;
			}
		}
	}
		
	if !catfound {
		cols["uncat"] = ""
		Logf(DEBUG, "Setting uncat\n")
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
		showTime := (triggerAt.Format("15:04") != "00:00")

		var formatString string
		if showYear {
			formatString = "2006-01-02"
		} else {
			formatString = "02/01"
		}

		if showTime {
			formatString += " 15:04"
		}
		
		return "@ " + triggerAt.Format(formatString)
	} else {
		return sort
	}

	return ""
}

var numberRE *regexp.Regexp = regexp.MustCompile("^[0-9.]+$")

func isNumber(tk string) (n float, ok bool) {
	if !numberRE.MatchString(tk) { return -1, false }
	n, err := strconv.Atof(tk)
	if err != nil { return -1, false }
	return n, true
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

	foundcat := false
	for _, v := range strings.Split(umentry.Cols, "\n", -1) {
		vs := strings.Split(v, ":", 2)

		if len(vs) == 0 { continue }

		if len(vs) == 1 {
			// it's a category
			x := strings.TrimSpace(v)
			Logf(DEBUG, "Adding [%s]\n", x)
			if x != "" {
				cols[x] = ""
				foundcat = true
			}
		} else {
			// it (may) be a column
			key := strings.TrimSpace(vs[0])
			value := strings.TrimSpace(vs[1])
			
			if key != "" {
				// Normalizes value
				
				Logf(DEBUG, "Normalizing: [%s]\n", value)
				if t, _ := ParseDateTime(value); t != nil {
					value = t.Format(TRIGGER_AT_FORMAT)
				} else if n, ok := isNumber(value); ok {
					value = fmt.Sprintf("%0.6f", n)
				}

				Logf(DEBUG, "Adding [%s] -> [%s]\n", key, value)

				cols[key] = value

				if value == "" { foundcat = true }
			}
		}
	}

	if !foundcat {
		cols["uncat"] = ""
	}
	
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

	return MakeUnmarshalEntry(
		entry.Id(),
		entry.Title(),
		entry.Text(),
		entry.Priority(),
		freq.String(),
		triggerAtString,
		entry.Sort(),
		entry.ColString()) 
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

func ParseTsvFormat(in string, tl *Tasklist) *Entry {
	fields := strings.Split(in, "\t", 4)

	entry, _ := QuickParse(fields[1], "", tl)

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

	entry.SetId(fields[0])
	entry.SetPriority(priority)
	entry.SetTriggerAt(triggerAt)
	entry.SetSort(sort)

	return entry
}

func (e *Entry) CatString() string {
	var r vector.StringVector

	for k, v := range e.Columns() {
		if v != "" { continue; }
		r.Push(k)
	}
	
	return "#" + strings.Join(([]string)(r), "#")
}

func (e *Entry) ColString() string {
	var r vector.StringVector

	for k, v := range e.Columns() {
		r.Push(k + ": " + v)
	}

	return strings.Join(([]string)(r), "\n") + "\n"
}

func (entry *Entry) Print() {
	fmt.Printf("%s\n%s\n", entry.Title(), entry.Text())
	
	tw := tabwriter.NewWriter(os.Stdout, 8, 8, 4, ' ', 0)
	w := bufio.NewWriter(tw)
	
	pr := entry.Priority()
	w.WriteString(fmt.Sprintf("Priority:\t%s\n", pr.String()))
	if entry.TriggerAt() != nil {
		w.WriteString(fmt.Sprintf("When:\t%s\n", entry.TriggerAt()))
	} else {
		w.WriteString("When:\tN/A\n")
	}
	fr := entry.Freq()
	w.WriteString(fmt.Sprintf("Recur:\t%s\n", fr.String()))
	w.WriteString(fmt.Sprintf("Sort:\t%s\n", entry.Sort()))
	for k, v := range entry.Columns() {
		pv := v
		if v == "" { pv = "<category>" }
		w.WriteString(fmt.Sprintf("%s:\t%v\n", k, pv))
	}
	w.WriteString("\n")
	w.Flush()
	tw.Flush()
}
