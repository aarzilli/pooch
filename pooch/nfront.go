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

func nfNewHandler(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	defer panicToErrors(c)
	Must(req.ParseForm())
	id := idRevertSpaces(getForm(req, "id"))
	n, err := strconv.ParseInt(getForm(req, "n"), 10, 32)
	Must(err)

	idtype, realId, _ := identifyId(id)

	switch idtype {
	case ID_IS_ID:
		entry := tl.ParseNew("#sub/"+realId, "")
		tl.Add(entry)
		nfMove(tl, realId, entry.Id(), int(n), entry)
		returnJson(c, "", []*Object{entryToObject(tl.Get(entry.Id()))})
	default:
		entry := tl.ParseNew("", id)
		tl.Add(entry)
		returnJson(c, "", []*Object{entryToObject(entry)})
	}
}

func nfMove(tl *Tasklist, pid string, id string, n int, entry *Entry) {
	childs := tl.GetChildren(pid)
	newchilds := make([]string, len(childs))
	s := 0
	d := 0
	added := false
	for s < len(childs) {
		if d == n {
			added = true
			newchilds[d] = id
			entry.SetColumn("sub/"+pid, strconv.Itoa(d))
			d++
		} else if childs[s] == id {
			s++
		} else {
			newchilds[d] = childs[s]
			s++
			d++
		}
	}
	if !added {
		newchilds[d] = id
		entry.SetColumn("sub/"+pid, strconv.Itoa(d))
	}
	tl.UpdateChildren(pid, newchilds)
}

func nfListHandler(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	defer panicToErrors(c)
	Must(req.ParseForm())
	id := idRevertSpaces(getForm(req, "id"))
	dochilds := getForm(req, "c")
	switch dochilds {
	case "0":
		o := singleTaskOrCatToObject(tl, id)
		returnJson(c, "", []*Object{o})
	case "1":
		os := runQueryOrChildsToObjects(tl, id, true)
		returnJson(c, "", os)
	case "2":
		os := runQueryOrChildsToObjects(tl, id, false)
		returnJson(c, "", os)
	}
}

func singleTaskOrCatToObject(tl *Tasklist, id string) *Object {
	if id == "" {
		return &Object{Id: "", Name: "root", Body: "root", ChildrenCount: 1, FormattedText: "<b>root</b>", Priority: "unknown", Editable: false}
	}

	idtype, realId, cats := identifyId(id)
	switch idtype {
	case ID_IS_ID:
		return singleTaskToObject(tl, realId)
	case ID_IS_CATLIST, ID_IS_CATLIST_AND_PRIORITY, ID_IS_SAVEDQUERY_AND_PRIORITY, ID_IS_COMPLEX_AND_PRIORITY:
		return &Object{Id: idConvertSpaces(id), Name: id, Body: cats[len(cats)-1], ChildrenCount: 1, FormattedText: "<b>" + html.EscapeString("#"+cats[len(cats)-1]) + "</b>", Priority: "unknown", Editable: false}
	case ID_IS_COMPLEX, ID_IS_SAVEDQUERY:
		return &Object{Id: idConvertSpaces(id), Name: id, Body: id, ChildrenCount: 1, FormattedText: "<b>" + html.EscapeString(id) + "</b>", Priority: "unknown", Editable: false}
	}
	return nil
}

func runQueryOrChildsToObjects(tl *Tasklist, id string, autoExpand bool) []*Object {
	if id == "" {
		return rootOntology(tl)
	}

	idtype, realId, cats := identifyId(id)
	switch idtype {
	case ID_IS_ID:
		Logf(INFO, "NF\tchildsToObjects\t%s\n", realId)
		return childsToObjects(tl, realId)
	case ID_IS_CATLIST:
		Logf(INFO, "NF\tcatChildsToObjects\t%s\t%v\n", id, autoExpand)
		return catChildsToObjects(tl, id, cats, autoExpand)
	case ID_IS_CATLIST_AND_PRIORITY:
		Logf(INFO, "NF\tqueryToObjects\t%s\t%s\n", id, cats[len(cats)-1])
		return queryToObjects(tl, id, cats[len(cats)-1])
	case ID_IS_SAVEDQUERY_AND_PRIORITY:
		Logf(INFO, "NF\tqueryToObjects\t%s\t%s\n", id, cats[1])
		return queryToObjects(tl, id, cats[1])
	case ID_IS_COMPLEX_AND_PRIORITY:
		Logf(INFO, "NF\tqueryToObjects\t%s\t%s\n", id, cats[0])
		return queryToObjects(tl, id, cats[0])
	case ID_IS_COMPLEX, ID_IS_SAVEDQUERY:
		Logf(INFO, "NF\tqueryToObjects\t%s\n", id)
		return queryToObjects(tl, id, "")
	}
	return nil
}

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

/*
 n == -1: at the end
 n == <a number>: at that position
*/
func nfMoveHandler(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	defer panicToErrors(c)
	Must(req.ParseForm())
	id := getForm(req, "id")
	p := getForm(req, "p")
	n, err := strconv.ParseInt(getForm(req, "n"), 10, 32)
	Must(err)

	idtype, realId, cats := identifyId(id)
	pidtype, realPid, pcats := identifyId(p)
	if idtype == ID_IS_ID && pidtype == ID_IS_ID {
		nfMoveHandlerObjects(c, tl, n, realId, realPid)
		return
	}

	_ = cats
	_ = pcats
	/*
		if idtype == ID_IS_CATLIST && (pidtype == ID_IS_CATLIST || p == "") {
			nfMoveHandlerOntology(c, tl, n, cats, pcats)
			return
		}
	*/

	returnJson(c, "Operation only implemented between categories or between objects", nil)
}

func nfMoveHandlerObjects(c http.ResponseWriter, tl *Tasklist, n int64, realId, realPid string) {
	entry := tl.Get(realId)

	if ok, curPid := IsSubitem(entry.Columns()); ok {
		if curPid != realPid {
			entry.SetColumn("sub/"+realPid, "-1")
			entry.RemoveColumn("sub/" + curPid)
			tl.Update(entry, false)
		}
	} else {
		entry.SetColumn("sub/"+realPid, "-1")
		tl.Update(entry, false)
	}
	if m := subitemSort(entry); m >= 0 && m < int(n) {
		n--
	}
	nfMove(tl, realPid, realId, int(n), entry)
	returnJson(c, "", []*Object{entryToObject(entry)})
}

func nfMoveHandlerOntology(c http.ResponseWriter, tl *Tasklist, n int64, cats []string, pcats []string) {
	fmt.Printf("cats: %v pcats: %v\n", cats, pcats)
	ontology := tl.GetOntology()
	ontology, ok := ontologyRemove(ontology, cats[len(cats)-1])
	if !ok {
		returnJson(c, "Could not complete operation (1)", nil)
	}
	ontology, ok = ontologyAdd(ontology, pcats[len(pcats)-1], cats[len(cats)-1])
	if !ok {
		returnJson(c, "Could not complete operation (2)", nil)
	}
	mor, err := json.Marshal(ontology)
	Must(err)
	tl.SetSetting("ontology", string(mor))
	returnJson(c, "", []*Object{ontologyNodeToObject(cats[len(cats)-1])})
}

func nfRemoveHandler(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	defer panicToErrors(c)
	Must(req.ParseForm())
	id := getForm(req, "id")
	idtype, realId, _ := identifyId(id)

	if idtype == ID_IS_ID {
		tl.Remove(realId)
		returnJson(c, "", nil)
	} else {
		returnJson(c, "Can not be deleted", nil)
	}
}

func nfCurcutHandler(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	defer panicToErrors(c)
	Must(req.ParseForm())
	if req.Method == "PUT" {
		tl.curCut = getForm(req, "id")
		returnJson(c, "", []*Object{&Object{Id: tl.curCut}})
	} else {
		returnJson(c, "", []*Object{&Object{Id: tl.curCut}})
	}
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

func singleTaskToObject(tl *Tasklist, id string) *Object {
	entry := tl.Get(id)
	return entryToObject(entry)
}

func childsToObjects(tl *Tasklist, id string) []*Object {
	q := "#:sub #:w/done #:ssort #sub/" + id
	return queryToObjects(tl, q, "")
}

func catChildsToObjects(tl *Tasklist, q string, cats []string, autoExpand bool) []*Object {
	os := searchOntology(tl, cats)
	if (len(os) > 0) || !autoExpand {
		os = append(os, priorityObject(q, "#sticky"))
		os = append(os, priorityObject(q, "#now"))
		os = append(os, priorityObject(q, "#later"))
		os = append(os, priorityObject(q, "#notes"))
		os = append(os, priorityObject(q, "#timed"))
		os = append(os, priorityObject(q, "#done"))
	} else {
		os = queryToObjects(tl, q, "")
	}
	return os
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

	//TODO: switching to calview, how?

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

	//formattedText = "<b>" + html.EscapeString(e.Title()) + "</b>"

	formattedText += "<p>"

	if e.Text() != "" {
		formattedText += strings.Replace(html.EscapeString(e.Text()), "\n", "<br>", -1)
	}

	formattedText += "</p>"

	/*
		formattedText += "<p>"

		for k, v := range e.Columns() {
			if strings.HasPrefix(k, "sub/") {
				continue
			}
			formattedText += "<span class='formattedcol'>"
			if v != "" {
				formattedText += fmt.Sprintf("%s=%s", html.EscapeString(k), html.EscapeString(v))
			} else {
				formattedText += fmt.Sprintf("%s", html.EscapeString(k))
			}
			formattedText += "</span>"
		}

		if e.TriggerAt() != nil {
			formattedText += "<span class='formattedcol'>when=" + e.TriggerAt().Format(TRIGGER_AT_FORMAT) + "</span>"
		}

		formattedText += "</p>"*/

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

func rootOntology(tl *Tasklist) []*Object {
	ontology := tl.GetOntology()
	knownTags := map[string]bool{}

	getTagsInOntology(knownTags, ontology)

	savedSearches := tl.GetSavedSearches()
	tags := tl.GetTags()

	for _, ss := range savedSearches {
		n := "#%" + ss
		if _, ok := knownTags[n]; !ok {
			ontology = append(ontology, OntologyNodeIn{n, "open", []OntologyNodeIn{OntologyNodeIn{}}})
		}
	}

	for _, t := range tags {
		n := "#" + t
		if _, ok := knownTags[n]; !ok {
			ontology = append(ontology, OntologyNodeIn{n, "open", []OntologyNodeIn{OntologyNodeIn{}}})
		}
	}

	return ontologyToObjects(ontology)
}

func ontologyNodeToObject(ondata string) *Object {
	return &Object{
		Id:            ondata,
		Name:          ondata,
		Body:          ondata,
		ChildrenCount: 1,
		FormattedText: "<b>" + html.EscapeString(ondata) + "</b>",
		Priority:      "unknown",
		Editable:      false,
	}
}

func ontologyToObjects(ontology []OntologyNodeIn) []*Object {
	r := make([]*Object, len(ontology))
	for i := range ontology {
		r[i] = ontologyNodeToObject(ontology[i].Data)
	}
	return r
}

func searchOntology(tl *Tasklist, cats []string) []*Object {
	if cats == nil || len(cats) == 0 {
		return []*Object{}
	}

	ontology := tl.GetOntology()
	var searchOntologyEx func(node []OntologyNodeIn, cats []string) []OntologyNodeIn
	searchOntologyEx = func(node []OntologyNodeIn, cats []string) []OntologyNodeIn {
		if len(cats) == 0 {
			return node
		}
		for i := 0; i < len(node); i++ {
			if node[i].Data == "#"+cats[0] {
				return searchOntologyEx(node[i].Children, cats[1:])
			}
		}
		return []OntologyNodeIn{}
	}

	return ontologyToObjects(searchOntologyEx(ontology, cats))
}

func idConvertSpaces(id string) string {
	return strings.Replace(id, " ", "·", -1)
}

func idRevertSpaces(id string) string {
	return strings.Replace(id, "·", " ", -1)
}
