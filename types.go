/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"fmt"
	"time"
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

type Frequency int64

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

func (freq *Frequency) String() string {
	switch *freq {
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

	return fmt.Sprintf("%d", int64(*freq))
}

func (freq *Frequency) ToInteger() int {
	return int(*freq)
}

type UnmarshalEntry struct {
	Id string
	Title string
	Text string
	Priority Priority
	Freq string
	TriggerAt string
	Sort string
	Tasklist string
}

type CalendarEvent struct {
	id string
	title string
	allDay bool
	start string
	className string
}

type Columns map[string]string

type Entry struct {
	id string
	title string
	text string
	priority Priority
	freq Frequency
	triggerAt *time.Time
	sort string
	columns Columns
}

func MakeUnmarshalEntry(id string, title string, text string, priority Priority, freq string, triggerAt string, sort string, tasklist string) *UnmarshalEntry {
	return &UnmarshalEntry{id, title, text, priority, freq, triggerAt, sort, tasklist}
}

func MakeEntry(id string, title string, text string, priority Priority, freq Frequency, triggerAt *time.Time, sort string, columns Columns) *Entry {
	return &Entry{id, title, text, priority, freq, triggerAt, sort, columns}
}

func (e *Entry) Title() string { return e.title; }
func (e *Entry) Text() string { return e.text; }
func (e *Entry) Id() string { return e.id; }
func (e *Entry) SetId(id string) { e.id = id; }
func (e *Entry) Priority() Priority { return e.priority; }
func (e *Entry) Freq() Frequency { return e.freq; }
func (e *Entry) TriggerAt() *time.Time { return e.triggerAt; }
func (e *Entry) Sort() string { return e.sort; }
func (e *Entry) Columns() Columns { return e.columns; }

func (entry *Entry) NextEntry(newId string) *Entry {
	freq := entry.Freq()
	newTriggerAt := time.SecondsToUTC(entry.TriggerAt().Seconds() + int64(freq.ToInteger() * 24 * 60 * 60))

	return MakeEntry(newId, entry.Title(), entry.Text(), entry.Priority(), entry.Freq(), newTriggerAt, entry.Sort(), entry.Columns())
}

func (e *Entry) Before(time int64) bool {
	return e.triggerAt.Seconds() < time
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
			//fmt.Printf("trigger: %d cur: %d\n", e.TriggerAt().Seconds(), time.LocalTime().Seconds())
			if e.TriggerAt().Seconds() > time.LocalTime().Seconds() { // trigger time is in the future
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


