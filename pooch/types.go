/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch


import (
	"fmt"
	"time"
	"sort"
	"strings"
	"strconv"
	"hash/crc32"
	"os"
	"encoding/base64"
	"text/tabwriter"
	"bufio"
)

const TRIGGER_AT_FORMAT = "2006-01-02 15:04"
const TEXT_COLS_SEPARATOR = "\n#+\n"

type ParseError struct {
	error string
}

func (e *ParseError) Error() string {
	return e.error
}

func MakeParseError(error string) error {
	return &ParseError{error}
}

func (pe *ParseError) String() string {
	return pe.error
}

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

type ErrorEntry struct {
	Time time.Time
	Message string
}

func (ee *ErrorEntry) TimeString() string {
	return ee.Time.Format("2006-01-02 15:04:05")
}

func Must(err error) {
	if err != nil { panic(err) }
}

func MarshalEntry(entry *Entry, timezone int) *UnmarshalEntry {
	triggerAtString := entry.TriggerAtString(timezone)

	return &UnmarshalEntry{
		entry.Id(),
		entry.Title(),
		entry.Text() + "\n" + TEXT_COLS_SEPARATOR + entry.ColString(),
		entry.Priority(),
		triggerAtString,
		entry.Sort()}
}

func DemarshalEntry(umentry *UnmarshalEntry, timezone int) *Entry {
	triggerAt, _ := ParseDateTime(umentry.TriggerAt, timezone)

	sort := umentry.Sort
	if sort == "" { sort = SortFromTriggerAt(triggerAt, false) }

	v := strings.SplitN(umentry.Text, TEXT_COLS_SEPARATOR, 2)
	text := strings.TrimRight(v[0], "\n")
	var colstr = ""
	if len(v) > 1 {
		colstr = v[1]
	}

	cols, foundcat := ParseCols(colstr, timezone)

	if !foundcat {
		cols["uncat"] = ""
	}

	return MakeEntry(
		umentry.Id,
		umentry.Title,
		text,
		umentry.Priority,
		triggerAt,
		sort,
		cols)
}

func MakeEntry(id string, title string, text string, priority Priority, triggerAt *time.Time, sort string, columns Columns) *Entry {
	return &Entry{id, title, text, priority, triggerAt, sort, columns}
}

func (e *Entry) Title() string { return e.title; }
func (e *Entry) SetTitle(title string) *Entry { e.title = title; return e }
func (e *Entry) Text() string { return e.text; }
func (e *Entry) SetText(text string) *Entry { e.text = text; return e }
func (e *Entry) Id() string { return e.id; }
func (e *Entry) SetId(id string) *Entry { e.id = id; return e}
func (e *Entry) Priority() Priority { return e.priority; }
func (e *Entry) SetPriority(p Priority) *Entry { e.priority = p; return e}
func (e *Entry) TriggerAt() *time.Time { return e.triggerAt; }
func (e *Entry) SetTriggerAt(tat *time.Time) { e.triggerAt = tat; }
func (e *Entry) SetSort(sort string) { e.sort = sort; }
func (e *Entry) Sort() string { return e.sort; }
func (e *Entry) Columns() Columns { return e.columns; }
func (e *Entry) ColumnOk(name string) (value string, ok bool) { value, ok = e.columns[name]; return; }
func (e *Entry) Column(name string) string { return e.columns[name];  }
func (e *Entry) SetColumns(cols Columns) *Entry { e.columns = cols; return e }
func (e *Entry) SetColumn(name, value string) *Entry { e.columns[name] = value; return e }
func (e *Entry) RemoveColumn(name string) *Entry { delete(e.columns, name); return e }

func IsSubitem(cols Columns) (bool, string) {
	for k, _ := range cols {
		if strings.HasPrefix(k, "sub/") {
			return true, k[4:]
		}
	}
	return false, ""
}

func (e *Entry) MergeColumns(cols Columns) *Entry {
	for k, v := range cols {
		e.columns[k] = v
	}
	return e
}

func ParseFrequency(freq string) int {
	switch freq {
	case "daily": return 1
	case "weekly": return 7
	case "biweekly": return 14
	case "monthly": return 30
	case "yearly": return 365
	}
	v, _ := strconv.Atoi(freq)
	return v
}

func (e *Entry) Freq() int {
	freqStr, ok := e.ColumnOk("freq")
	if !ok { return -1 }
	freq := ParseFrequency(freqStr)
	if freq > 0 { return freq }
	return -1
}

/*
 * Rules for timezones: add timezone to convert to local time
 * subtract timezone to convert to utc
 */

func (entry *Entry) TriggerAtString(timezone int) string {
	triggerAt := entry.TriggerAt()
	triggerAtString := ""
	if triggerAt != nil {
		z := time.Unix(triggerAt.Unix() + (int64(timezone) * 60 * 60), 0).In(time.FixedZone("fixed-zone", timezone * 60))
		triggerAtString = z.Format(TRIGGER_AT_FORMAT)
	}

	return triggerAtString
}

func (entry *Entry) NextEntry(newId string) *Entry {
	newTriggerAt := time.Unix(entry.TriggerAt().Unix() + int64(entry.Freq() * 24 * 60 * 60), 0)

	return MakeEntry(newId, entry.Title(), entry.Text(), entry.Priority(), &newTriggerAt, entry.Sort(), entry.Columns())
}

func (e *Entry) Before(time int64) bool {
	return e.triggerAt.Unix() < time
}

func (e *Entry) CatHash() uint32 {
	cats := make([]string, 0)

	for key, value := range e.Columns() {
		if value == "" {
			cats = append(cats, key)
		}
	}

	sort.Strings(cats)

	catstring := strings.Join(cats, "#")

	hasher := crc32.NewIEEE()
	hasher.Write(([]uint8)(catstring))

	return hasher.Sum32()
}

func (e *Entry) ColString() string {
	r := make([]string, 0)

	for k, v := range e.Columns() {
		if strings.IndexAny(v, "\r\n") != -1 {
			r = append(r, k + ": {\n" + v + "}")
		} else {
			r = append(r, k + ": " + v)
		}
	}

	return strings.Join(r, "\n") + "\n"
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
func (e *Entry) UpgradePriority(special bool) bool {
	if e.TriggerAt() != nil {
		switch e.Priority() {
		case NOW:
			e.priority = DONE
			e.columns["done-at"] = time.Now().UTC().Format("2006-01-02_15:04:05")
			return false
		case TIMED:
			e.priority = DONE
			e.columns["done-at"] = time.Now().UTC().Format("2006-01-02_15:04:05")
			return false
		default:
			if e.TriggerAt().Unix() > time.Now().UTC().Unix() { // trigger time is in the future
				e.priority = TIMED
			} else {
				e.priority = NOW
			}
			return true
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
		return true
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
				e.columns["done-at"] = time.Now().UTC().Format("2006-01-02_15:04:05")
				return false
			}
			return true
		}
	}

	return true
}

func RepeatString(ch string, num int) string {
	if num < 0 { return "" }
	return strings.Repeat(ch, num)
}

func DecodeBase64(in string) string {
	decbuf := make([]byte, base64.StdEncoding.DecodedLen(len(in)))
	n, err := base64.StdEncoding.Decode(decbuf, []byte(in))
	Must(err)
	return string(decbuf[:n])
}

func decodeStatic(name string) string {
	content := FILES[name]
	z := DecodeBase64(content)
	return z
	/*
	var i int
	for i = len(z)-1; i > 0; i-- {
		if z[i] != 0 {
			break
		}
	}
	return z[0:i+1]*/
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

type Pair struct {
	Key string
	Value int
}

type Pairs []Pair

func (ps Pairs) Len() int {
	return len(ps)
}

func (ps Pairs) Less(i, j int) bool {
	return ps[i].Value < ps[j].Value
}

func (ps Pairs) Swap(i, j int) {
	t := ps[i]
	ps[i] = ps[j]
	ps[j] = t
}

func (e *Entry) CatString(catordering map[string]int) string {
	r := make([]string, 0)

	for k, v := range e.Columns() {
		if v != "" { continue; }
		r = append(r, k)
	}

	if catordering != nil {
		ps := make(Pairs, len(r))
		for i, v := range r {
			vv, ok := catordering[v]
			if !ok {
				vv = 1000
			}
			ps[i] = Pair{ v, vv }
		}
		sort.Sort(ps)
		for i, v := range ps {
			r[i] = v.Key
		}
	}

	return "#" + strings.Join(r, "#")
}

func TimeString(triggerAt *time.Time, sort string, timezone int) string {
	if triggerAt != nil {
		now := time.Now().UTC()
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

		return "@ " + TimeFormatTimezone(triggerAt, formatString, timezone)
		//return "@ " + triggerAt.Format(formatString)
	} else {
		return sort
	}

	return ""
}

type ListJsonAnswer struct {
	ParseError error
	RetrieveError error
	Results []UnmarshalEntry
}

type OntologyNodeOut struct {
	Data string `json:"data,omitempty"`
	State string `json:"state"`
	Children []interface{} `json:"children"`
}

type OntologyNodeIn struct {
	Data string `json:"data,omitempty"`
	State string `json:"state"`
	Children []OntologyNodeIn `json:"children"`
}

type OntoCheckError struct {
	Entry *Entry
	ProblemCategory string
	ProblemDetail string
}