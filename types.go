/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"fmt"
	"time"
	"container/vector"
	"sort"
	"strings"
	"hash/crc32"
	"os"
)

type Priority int;

const (
	STICKY Priority = Priority(iota)
	NOW
	LATER
	NOTES
	TIMED
	DONE
	INVALID
)

func (p *Priority) String() string {
	switch *p {
	case STICKY:
		return "sticky"
	case NOW:
		return "now"
	case LATER:
		return "later"
	case NOTES:
		return "notes"
	case TIMED:
		return "timed"
	case DONE:
		return "done"
	}
	return "unknown"
}

func (p *Priority) ToInteger() int {
	return int(*p)
}

func FrequencyToString(freq int) string {
	switch freq {
	case 1:
		return "daily"
	case 7:
		return "weekly"
	case 14:
		return "biweekly"
	case 30:
		return "monthly"
	case 365:
		return "yearly"
	}

	return fmt.Sprintf("%d", int64(freq))
}

type UnmarshalEntry struct {
	Id string
	Title string
	Text string
	Priority Priority
	TriggerAt string
	Sort string
	Cols string
}

type Columns map[string]string

type Entry struct {
	id string
	title string
	text string
	priority Priority
	triggerAt *time.Time
	sort string
	columns Columns
}

func must(err os.Error) {
	if err != nil { panic(err) }
}

func MakeUnmarshalEntry(id string, title string, text string, priority Priority, triggerAt string, sort string, cols string) *UnmarshalEntry {
	return &UnmarshalEntry{id, title, text, priority, triggerAt, sort, cols}
}

func MakeEntry(id string, title string, text string, priority Priority, triggerAt *time.Time, sort string, columns Columns) *Entry {
	return &Entry{id, title, text, priority, triggerAt, sort, columns}
}

func (e *Entry) Title() string { return e.title; }
func (e *Entry) Text() string { return e.text; }
func (e *Entry) SetText(text string)  *Entry { e.text = text; return e }
func (e *Entry) Id() string { return e.id; }
func (e *Entry) SetId(id string) *Entry { e.id = id; return e}
func (e *Entry) Priority() Priority { return e.priority; }
func (e *Entry) SetPriority(p Priority) { e.priority = p; }
func (e *Entry) TriggerAt() *time.Time { return e.triggerAt; }
func (e *Entry) SetTriggerAt(tat *time.Time) { e.triggerAt = tat; }
func (e *Entry) SetSort(sort string) { e.sort = sort; }
func (e *Entry) Sort() string { return e.sort; }
func (e *Entry) Columns() Columns { return e.columns; }
func (e *Entry) Column(name string) (value string, ok bool) { value, ok = e.columns[name]; return; }
func (e *Entry) SetColumns(cols Columns) *Entry { e.columns = cols; return e }

func (e *Entry) MergeColumns(cols Columns) *Entry {
	for k, v := range cols {
		e.columns[k] = v
	}
	return e
}

func (e *Entry) Freq() int {
	freqStr, ok := e.Column("freq")
	if !ok { return -1 }
	freq := ParseFrequency(freqStr)
	if freq > 0 { return freq }
	return -1
}

func (entry *Entry) NextEntry(newId string) *Entry {
	newTriggerAt := time.SecondsToUTC(entry.TriggerAt().Seconds() + int64(entry.Freq() * 24 * 60 * 60))

	return MakeEntry(newId, entry.Title(), entry.Text(), entry.Priority(), newTriggerAt, entry.Sort(), entry.Columns())
}

func (e *Entry) Before(time int64) bool {
	return e.triggerAt.Seconds() < time
}

func (e *Entry) CatHash() uint32 {
	var catsVector vector.StringVector

	for key, value := range e.Columns() {
		if value == "" { catsVector.Push(key) }
	}

	cats := ([]string)(catsVector)

	sort.SortStrings(cats)

	catstring := strings.Join(cats, "#")

	hasher := crc32.NewIEEE()
	hasher.Write(([]uint8)(catstring))

	return hasher.Sum32()
}

func StripQuotes(in string) string {
	if in == "" { return in }
	if in[0] != '"' && in[0] != '\'' { return in }
	if in[len(in)-1] != '"' && in[len(in)-1] != '\'' { return in }
	return in[1:len(in)-1]
}

/*
		 +--------+		   +---------+
		 |        |------->|         |
		 | STICKY |		   | NOTES   |
		 |        |<-------|         |
		 +--------+		   +---------+
			   ===				 =
				  ====			 =
					  =====		 =	 Special transitions (press SHIFT)
						   ====	 =
 							   ==V
			 +---------+ 	 +--------+  	+---------+
			 |     	   | 	 |        |  	|         |
			 | LATER   |---->| NOW    |---->| DONE    |
			 |         | 	 |        |  	|         |
			 +---------+ 	 +--------+  	+---------+
				 A                               |
				 |                               |
				 +-------------------------------|

 When there is a triggerAt (When) field set:


	   +----------+			+----------+		+-----------+
	   |          |			|          |------->|           |
	   | ANYTHING |-------->| TIMED    |		| DONE      |
       |          |			|          |<-------|           |
 	   +----------+			+----------+		+-----------+


 */
func (e *Entry) UpgradePriority(special bool) {
	if e.TriggerAt() != nil {
		switch e.Priority() {
		case NOW:
			e.priority = DONE
		case TIMED:
			e.priority = DONE
		default:
			if e.TriggerAt().Seconds() > time.UTC().Seconds() { // trigger time is in the future
				e.priority = TIMED
			} else {
				e.priority = NOW
			}
		}
	} else if (e.priority == NOTES) || (e.priority == STICKY) {
		if special {
			e.priority = NOW
		} else {
			switch e.Priority() {
			case STICKY: e.priority = NOTES
			case NOTES: e.priority = STICKY
			}
		}
	} else { // anything else
		if special {
			e.priority = NOTES
		} else {
			switch e.Priority() {
			case TIMED:
				e.priority = LATER
			case DONE:
				e.priority = LATER
			case LATER:
				e.priority = NOW
			case NOW:
				e.priority = DONE
			}
		}
	}
}

func RepeatString(ch string, num int) string {
	if num < 0 { return "" }
	return strings.Repeat(ch, num)
}

