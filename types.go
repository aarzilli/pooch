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

type Entry struct {
	id string
	title string
	text string
	priority Priority
	freq Frequency
	triggerAt *time.Time
	sort string
}

func MakeUnmarshalEntry(id string, title string, text string, priority Priority, freq string, triggerAt string, sort string, tasklist string) *UnmarshalEntry {
	return &UnmarshalEntry{id, title, text, priority, freq, triggerAt, sort, tasklist}
}

func MakeEntry(id string, title string, text string, priority Priority, freq Frequency, triggerAt *time.Time, sort string) *Entry {
	return &Entry{id, title, text, priority, freq, triggerAt, sort}
}

func (e *Entry) Title() string { return e.title; }
func (e *Entry) Text() string { return e.text; }
func (e *Entry) Id() string { return e.id; }
func (e *Entry) SetId(id string) { e.id = id; }
func (e *Entry) Priority() Priority { return e.priority; }
func (e *Entry) Freq() Frequency { return e.freq; }
func (e *Entry) TriggerAt() *time.Time { return e.triggerAt; }
func (e *Entry) Sort() string { return e.sort; }

func (entry *Entry) NextEntry(newId string) *Entry {
	freq := entry.Freq()
	newTriggerAt := time.SecondsToUTC(entry.TriggerAt().Seconds() + int64(freq.ToInteger() * 24 * 60 * 60))

	return MakeEntry(newId, entry.Title(), entry.Text(), entry.Priority(), entry.Freq(), newTriggerAt, entry.Sort())
}

func (e *Entry) Before(time int64) bool {
	return e.triggerAt.Seconds() < time
}

/*
	  +--------+	                 +--------+
	  | STICKY |<=================== | NOTES  |
	  +--------+	                 +--------+
			====   	   	   	   	   		^
			    ======				    =
				      ======		    =
 		   +----------------======------=---------------------------------+
		   V			          ==V	=								  |
 	  +--------+	              +--------+		                  +--------+
	  | LATER  |----------------->| NOW    |------------------------->| DONE   |
   	  +--------+	              +--------+		                  +--------+
		  == 						  ^
			====					  |
			    ====				  |
				    ===				  |
				       ====			  |
                           ====  +---------+
	                           =>| TIMED   |
	                             +---------+
 */
func (e *Entry) UpgradePriority(special bool) {
	switch e.Priority() {
	case STICKY:
		if special {
			e.priority = NOW
		} else {
			e.priority = STICKY
		}
	case NOW:
		if special {
			e.priority = NOTES
		} else {
			e.priority = DONE
		}
	case LATER:
		if special {
			e.priority = TIMED
		} else {
			e.priority = NOW
		}
	case NOTES:
		if special {
			e.priority = STICKY
		} else {
			e.priority = NOTES
		}
	case TIMED:
		e.priority = NOW
	case DONE:
		e.priority = LATER
	}
}

