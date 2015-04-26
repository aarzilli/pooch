package pooch

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type JsonResult struct {
	Error   string
	Objects []*Object
}

type Object struct {
	Id            string
	Name          string
	Title         string
	Body          string
	ChildrenCount int
	FormattedText string
	Priority      string
	Editable      bool
	Children      []*Object
}

type IdType int

const (
	ID_IS_ID = IdType(iota)
	ID_IS_CATLIST
	ID_IS_CATLIST_AND_PRIORITY
	ID_IS_SAVEDQUERY
	ID_IS_SAVEDQUERY_AND_PRIORITY
	ID_IS_COMPLEX
	ID_IS_COMPLEX_AND_PRIORITY
)

func parseUpdateBody(body string) (title, text, colstr string) {
	body = strings.TrimLeft(body, "\n")
	v := strings.SplitN(body, "\n", 2)
	if len(v) <= 0 {
		return "", "", ""
	}
	title = v[0]
	if len(v) < 2 {
		return title, "", ""
	}

	if strings.HasPrefix(v[1], TEXT_COLS_SEPARATOR[1:]) {
		text = ""
		colstr = v[1][len(TEXT_COLS_SEPARATOR[1:]):]
		return
	}

	v = strings.SplitN(v[1], TEXT_COLS_SEPARATOR, 2)
	if len(v) <= 0 {
		return title, "", ""
	}
	text = strings.TrimLeft(strings.TrimRight(v[0], "\n"), "\n")
	colstr = ""
	if len(v) > 1 {
		colstr = v[1]
	}
	return
}

func nfUpdateHandler(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	defer panicToErrors(c)
	Must(req.ParseForm())
	id := getForm(req, "id")
	body := getForm(req, "body")
	idtype, realId, _ := identifyId(id)
	if idtype != ID_IS_ID {
		returnJson(c, "Not allowed on non-tasks", nil)
		return
	}
	entry := tl.Get(realId)
	title, text, _ := parseUpdateBody(body)
	entry.SetTitle(title)
	entry.SetText(text)
	tl.Update(entry, false)
	returnJson(c, "", []*Object{entryToObject(entry)})
}

func getForm(r *http.Request, name string) string {
	vs, ok := r.Form[name]
	if !ok {
		panic(fmt.Errorf("Missing required parameter %s", name))
	}
	if len(vs) != 1 {
		panic(fmt.Errorf("Multiple required parameters %s", name))
	}
	return vs[0]
}

func returnJson(w http.ResponseWriter, errstr string, r []*Object) {
	b, err := json.Marshal(&JsonResult{Error: errstr, Objects: r})
	Must(err)
	w.Write(b)
}

func panicToErrors(w http.ResponseWriter) {
	rerr := recover()
	if rerr == nil {
		return
	}
	WriteStackTrace(rerr, os.Stderr)
	b, _ := json.Marshal(&JsonResult{Error: rerr.(error).Error()})
	w.Write(b)
}

func identifyId(id string) (idtype IdType, realId string, cats []string) {
	complexCheck := func() (idtype IdType, realId string, cats []string) {
		for _, p := range []string{"#done", "#now", "#later", "#timed", "#notes", "#sticky"} {
			if strings.HasSuffix(id, p) {
				return ID_IS_COMPLEX_AND_PRIORITY, "", []string{p[1:]}
			}
		}
		return ID_IS_COMPLEX, "", nil
	}

	if len(id) == 0 || id[0] != '#' {
		return complexCheck()
	}

	if strings.HasPrefix(id, "#id=") {
		realId = id[len("#id="):]
		r := []rune(realId)
		for i := 0; i < len(r); i++ {
			if unicode.IsSpace(r[i]) || (r[i] == '#') {
				return complexCheck()
			}
		}
		return ID_IS_ID, realId, nil
	}

	if strings.HasPrefix(id, "#%") {
		savedQuery := id[len("#%"):]
		r := []rune(savedQuery)
		rest := []rune{}
		for i := 0; i < len(r); i++ {
			if unicode.IsSpace(r[i]) || (r[i] == '#') {
				savedQuery = string(r[:i])
				rest = r[i:]
				break
			}
		}

		if len(rest) == 0 {
			return ID_IS_SAVEDQUERY, "", []string{string(savedQuery)}
		}

		if rest[0] != '#' {
			return complexCheck()
		}

		reststr := string(rest)
		switch reststr {
		case "#now", "#later", "#done", "#timed", "#notes", "#sticky":
			return ID_IS_SAVEDQUERY_AND_PRIORITY, "", []string{savedQuery, reststr[1:]}
		default:
			return complexCheck()
		}
	}

	cats = strings.Split(id, "#")
	if (len(cats) < 2) && (cats[0] != "") {
		return complexCheck()
	}
	cats = cats[1:]

	for i := range cats {
		cat := []rune(cats[i])
		for j := range cat {
			if !unicode.IsLetter(cat[j]) && !unicode.IsDigit(cat[j]) && (cat[j] != '_') && (cat[j] != '-') {
				return complexCheck()
			}
		}
	}
	if len(cats) > 1 {
		switch cats[len(cats)-1] {
		case "now", "later", "done", "timed", "notes", "sticky":
			return ID_IS_CATLIST_AND_PRIORITY, "", cats
		default:
			return ID_IS_CATLIST, "", cats
		}
	}
	return ID_IS_CATLIST, "", cats
}

func childsToObjects(tl *Tasklist, id string) []*Object {
	q := "#:sub #:w/done #:ssort #sub/" + id
	return queryToObjects(tl, q, "")
}

type SubitemSort []*Entry

func (ss SubitemSort) Len() int {
	return len([]*Entry(ss))
}

func subitemSort(e *Entry) int {
	for k, v := range e.Columns() {
		if strings.HasPrefix(k, "sub/") {
			n, _ := strconv.ParseInt(v, 10, 32)
			return int(n)
		}
	}
	return -1
}

func (ss SubitemSort) Less(i, j int) bool {
	v := []*Entry(ss)
	return subitemSort(v[i]) < subitemSort(v[j])
}

func (ss SubitemSort) Swap(i, j int) {
	v := []*Entry(ss)
	t := v[i]
	v[i] = v[j]
	v[j] = t
}

func priorityObject(q string, p string) *Object {
	return &Object{
		Id:            idConvertSpaces(q + p),
		Name:          p,
		Body:          p,
		ChildrenCount: 1,
		FormattedText: "<b>" + html.EscapeString(p) + "</b>",
		Priority:      "unknown",
		Editable:      false,
		Children:      []*Object{},
	}
}

func queryToObjects(tl *Tasklist, q string, priority string) []*Object {
	query := strings.Replace(q, "\r", "", -1)
	theselect, code, _, _, _, _, options, perr := tl.ParseSearch(query, nil)
	Must(perr)

	_, incsub := options["sub"]
	v, rerr := tl.Retrieve(theselect, code, incsub)
	Must(rerr)

	os := []*Object{}

	_, ssort := options["ssort"]

	if (priority != "") || ssort {
		p := ParsePriority(priority)
		if ssort {
			sort.Sort(SubitemSort(v))
		}
		for _, e := range v {
			if ssort || (e.Priority() == p) {
				os = append(os, entryToObject(e))
			}
		}
	} else {
		var osPr *Object
		curp := INVALID
		doneDone := false
		for _, e := range v {
			if e.Priority() != curp {
				p := e.Priority()
				x := "#" + p.String()
				osPr = priorityObject(q, x)
				if p == DONE {
					doneDone = true
				}
				os = append(os, osPr)
				curp = e.Priority()
			}
			osPr.Children = append(osPr.Children, entryToObject(e))
		}
		if !doneDone {
			os = append(os, &Object{
				Id:            idConvertSpaces(q + "#done"),
				Name:          "#done",
				Body:          "#done",
				ChildrenCount: 1,
				FormattedText: "#done",
				Priority:      "unknown",
				Editable:      false,
				Children:      nil,
			})
		}
	}

	return os
}

func formatEntry(e *Entry) (body string, formattedText string) {
	body = e.Title()
	if e.Text() != "" {
		body += "\n\n" + e.Text()
	}
	body += TEXT_COLS_SEPARATOR + e.ColString(true)

	formattedText += "<p>"

	if e.Text() != "" {
		formattedText += strings.Replace(html.EscapeString(e.Text()), "\n", "<br>", -1)
	}

	formattedText += "</p>"

	return
}

func entryToObject(e *Entry) *Object {
	body, formattedText := formatEntry(e)
	var sort string
	if x := subitemSort(e); x >= 0 {
		sort = fmt.Sprintf("%d", x)
	} else {
		sort = e.Sort()
	}
	p := e.Priority()
	return &Object{
		Id:            e.Id(),
		Title:         e.Title(),
		Name:          sort,
		Body:          body,
		ChildrenCount: 1,
		FormattedText: formattedText,
		Priority:      p.String(),
		Editable:      true,
	}
}

func idConvertSpaces(id string) string {
	return strings.Replace(id, " ", "·", -1)
}
