package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const SPACING = 4

type showList struct {
	Day   string
	Shows []stringWithOffset
}

type stringWithOffset struct {
	Str string
	Off int
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func process(results []UnmarshalEntry, dateList map[string][]string, tvfsubcols []string, dateFormat string) {
	for _, res := range results {
		v := strings.SplitN(res.Text, "\n#+\n", 2)
		if len(v) != 2 {
			continue
		}
		text := strings.TrimSpace(v[0])
		cols := parseCols(v[1])
		subcol := identifySubcol(cols, tvfsubcols)

		emit := func(title string, sdate, edate time.Time) {
			emitRange(dateList, subcol+"@"+title, sdate, edate, dateFormat)
		}

		if text == "" {
			processNoText(&res, cols, emit)
		} else {
			if !processText(&res, text, emit) {
				processNoText(&res, cols, emit)
			}
		}
	}
}

func identifySubcol(cols map[string]string, tvfsubcols []string) string {
	for _, k := range tvfsubcols {
		if _, ok := cols[k]; ok {
			return k
		}
	}
	return ""
}

func processNoText(res *UnmarshalEntry, cols map[string]string, emit func(title string, sdate, edate time.Time)) {
	if res.Priority != DONE {
		return
	}
	datetime, ok := cols["done-at"]
	if !ok {
		return
	}
	v := strings.SplitN(datetime, "_", 2)
	date := v[0]
	t, err := time.Parse("2006-01-02", date)
	if err == nil {
		emit(res.Title, t, t)
	}
}

func processText(res *UnmarshalEntry, text string, emit func(title string, sdate, edate time.Time)) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return false
	}
	oneok := false
	for i := range lines {
		if processOneLine(res, lines[i], emit) {
			oneok = true
		}
	}
	return oneok
}

func processOneLine(res *UnmarshalEntry, line string, emit func(title string, sdate, edate time.Time)) bool {
	if line == "" {
		return false
	}

	if t, ok := parseDate(line); ok {
		emit(res.Title, t, t)
		return true
	}

	if t1, t2, ok := parseDateRange(line); ok {
		emit(res.Title, t1, t2)
		return true
	}

	if subtitle, t1, t2, ok := parseSubtitleLine(line); ok {
		emit(res.Title+": "+subtitle, t1, t2)
		return true
	}

	return false
}

func parseDateRange(line string) (sdate, edate time.Time, ok bool) {
	parseSimpleDateRange := func(sep string) (sdate, edate time.Time, ok bool) {
		v := strings.SplitN(line, sep, 2)
		if len(v) != 2 {
			ok = false
			return
		}

		sdatestr := strings.TrimSpace(v[0])
		vv := strings.SplitN(strings.TrimSpace(v[1]), " ", 2)
		edatestr := strings.TrimSpace(vv[0])

		sdate, ok1 := parseDate(sdatestr)
		edate, ok2 := parseDate(edatestr)

		if ok1 && ok2 {
			ok = true
			return
		}

		ok = false
		return
	}

	if sdate, edate, ok = parseSimpleDateRange("-"); ok {
		return
	}
	if sdate, edate, ok = parseSimpleDateRange("–"); ok {
		return
	}
	if sdate, edate, ok = parseSimpleDateRange("—"); ok {
		return
	}

	ok = false
	return
}

func parseDate(dstr string) (time.Time, bool) {
	if dstr == "..." || dstr == "…" {
		return time.Now(), true
	}
	t, err := time.Parse("2006-01-02", dstr)
	if err != nil {
		return t, false
	}
	return t, true
}

var endparRe = regexp.MustCompile(`(.*?)\(([^\(\)]*?)\)$`)

func parseSubtitleLine(line string) (subtitle string, sdate, edate time.Time, ok bool) {
	if strings.HasPrefix(line, "×") {
		line = line[len("×"):]
	} else if line[0] == 'x' || line[0] == '-' {
		line = line[1:]
	}

	if ms := endparRe.FindStringSubmatch(line); ms != nil {
		subtitle = strings.TrimSpace(ms[1])
		partext := strings.TrimSpace(ms[2])
		if sdate, edate, ok = parseDateRange(partext); ok {
			return
		}
		if sdate, ok = parseDate(partext); ok {
			edate = sdate
			return
		}
		ok = false
		return
	}

	if idx := strings.Index(line, ": "); idx >= 0 {
		subtitle = line[:idx]
		dr := line[idx+len(": "):]
		if sdate, edate, ok = parseDateRange(dr); ok {
			return
		}
		if sdate, ok = parseDate(dr); ok {
			edate = sdate
			return
		}
	}

	ok = false
	return
}

func emitRange(dateList map[string][]string, title string, sdate, edate time.Time, dateFormat string) {
	//fmt.Printf("%s\t%s\t%s\n", sdate.Format("2006-01-02"), edate.Format("2006-01-02"), title)
	prevTstr := ""
	for t := sdate; t.Before(edate) || t.Equal(edate); t = t.Add(24 * time.Hour) {
		tstr := t.Format(dateFormat)
		if tstr != prevTstr {
			if _, ok := dateList[tstr]; !ok {
				dateList[tstr] = []string{}
			}
			dateList[tstr] = append(dateList[tstr], title)
		}
		prevTstr = tstr
	}
}

func postProcess(dateList map[string][]string, tabs, minimal bool) []showList {
	ml := 0
	ks := []string{}
	for k, v := range dateList {
		if len(v) > ml {
			ml = len(v)
		}
		ks = append(ks, k)
	}

	sort.Strings(ks)

	r := make([]showList, 0, len(ks))

	for _, k := range ks {
		var cur []stringWithOffset
		if minimal || len(r) <= 0 {
			cur = addOffsets(dateList[k])
		} else {
			if tabs {
				cur = stableOrder(dateList[k], r[len(r)-1].Shows, ml)
			} else {
				cur = stableColumns(dateList[k], r[len(r)-1].Shows, ml)
			}
		}

		r = append(r, showList{Day: k, Shows: cur})
	}

	return r
}

func contains(v []string, k string) int {
	for i := range v {
		if v[i] == k {
			return i
		}
	}
	return -1
}

func addOffsets(in []string) []stringWithOffset {
	r := make([]stringWithOffset, len(in))
	off := 0
	for i := range in {
		r[i].Str = in[i]
		r[i].Off = off
		off += len(r[i].Str) + SPACING
	}

	return r
}

func stableOrder(cur []string, prev []stringWithOffset, n int) []stringWithOffset {
	r := make([]stringWithOffset, n)
	for i := range prev {
		if idx := contains(cur, prev[i].Str); idx >= 0 {
			r[i] = prev[i]
			cur[idx] = ""
		}
	}
	for i := range cur {
		if cur[i] == "" {
			continue
		}
		found := false
		for j := range r {
			if r[j].Str != "" {
				continue
			}

			r[j].Str = cur[i]

			found = true
			break
		}

		if !found {
			r = append(r, stringWithOffset{Str: cur[i], Off: -1})
		}

	}

	last := -1
	for i := range r {
		if r[i].Str != "" {
			last = i
		}
	}
	r = r[:last+1]

	return r
}

func stableColumns(cur []string, prev []stringWithOffset, n int) []stringWithOffset {
	r := make([]stringWithOffset, 0, n)
	for i := range prev {
		if idx := contains(cur, prev[i].Str); idx >= 0 {
			r = append(r, prev[i])
			cur[idx] = ""
		}
	}

	for i := range cur {
		if cur[i] == "" {
			continue
		}

		r = stableColumnsInsert(cur[i], r)
	}

	return r
}

func stableColumnsInsert(e string, r []stringWithOffset) []stringWithOffset {
	// Find a place within r to fit the new element
	sz := utf8.RuneCountInString(e) + SPACING

	if len(r) == 0 {
		// Nothing there yet, just take up the space
		r = append(r, stringWithOffset{Str: e, Off: 0})
		return r
	}

	if sz < r[0].Off {
		// Can fit before the first element
		r = append(r, stringWithOffset{Str: "", Off: 0})
		copy(r[1:], r[:len(r)-1])
		r[0].Str = e
		r[0].Off = 0
		return r
	}

	found := false
	for i := 0; i < len(r)-1; i++ {
		rilen := utf8.RuneCountInString(r[i].Str)
		if r[i].Off+rilen+SPACING+sz <= r[i+1].Off {
			found = true
			r = append(r, stringWithOffset{Str: "", Off: 0})
			copy(r[i+2:], r[i+1:len(r)-1])
			r[i+1].Str = e
			r[i+1].Off = r[i].Off + rilen + SPACING
			break
		}
	}

	if found {
		return r
	}

	r = append(r, stringWithOffset{Str: e, Off: r[len(r)-1].Off + len(r[len(r)-1].Str) + SPACING})

	return r
}

func (sl *showList) Format(tabs, minimal bool) string {
	r := bytes.NewBuffer([]byte{})

	r.Write([]byte(sl.Day))
	r.Write([]byte{'\t'})

	off := 0
	for i := range sl.Shows {
		if tabs {
			r.Write([]byte(sl.Shows[i].Str))
			r.Write([]byte{'\t'})
		} else if minimal {
			if i != 0 {
				r.Write([]byte{'\t'})
			}
			r.Write([]byte(sl.Shows[i].Str))
			r.Write([]byte{'\n'})
		} else {
			for ; off < sl.Shows[i].Off; off++ {
				r.Write([]byte{' '})
			}
			r.Write([]byte(sl.Shows[i].Str))
			off += utf8.RuneCountInString(sl.Shows[i].Str)
		}
	}

	return r.String()
}

func sortBySubcol(dateList map[string][]string) {
	for k, _ := range dateList {
		sort.Strings(dateList[k])
		for i := range dateList[k] {
			v := strings.SplitN(dateList[k][i], "@", 2)
			dateList[k][i] = v[1]
		}
	}
}

func main() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	doTabs := false
	doPostProcess := true
	doMonths := false
	doYears := false
	for i := range os.Args {
		switch os.Args[i] {
		case "tabs":
			doTabs = true
		case "months":
			doPostProcess = false
			doMonths = true
		case "years":
			doPostProcess = false
			doYears = true
		}
	}

	if !doPostProcess {
		doTabs = false
	}

	base, tok := readToken()

	oq := url.Values{"apiToken": []string{tok}}
	ontology := readOntology(client, base+"ontology?"+oq.Encode())
	tvfsubcols := findSubcols(ontology, "#tvf")

	q := url.Values{"q": []string{"#:w/done #tvf #hlater #+ notq(priorityq('later'))"}, "apiToken": []string{tok}}
	body := readUrl(client, base+"list.json?"+q.Encode())

	dateList := map[string][]string{}
	dateFormat := "2006-01-02"
	if doMonths {
		dateFormat = "2006-01"
	} else if doYears {
		dateFormat = "2006"
	}
	process(body.Results, dateList, tvfsubcols, dateFormat)
	sortBySubcol(dateList)
	var shows []showList
	shows = postProcess(dateList, doTabs, !doPostProcess)

	for i := range shows {
		fmt.Printf("%s\n", shows[i].Format(doTabs, !doPostProcess))
	}
}
