/*
 This program is distributed under the terms of GPLv3
 Copyright 2010-2013, Alessandro Arzilli
*/

package pooch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TasklistWithIdServer func(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string)
type TasklistServer func(c http.ResponseWriter, req *http.Request, tl *Tasklist)

func SingleWrapperTasklistWithIdServer(fn TasklistWithIdServer) http.HandlerFunc {
	return func(c http.ResponseWriter, req *http.Request) {
		WithOpenDefault(func(tl *Tasklist) {
			id := req.FormValue("id")
			if !tl.Exists(id) {
				panic(fmt.Sprintf("Non-existent id specified"))
			}
			fn(c, req, tl, id)
		})
	}
}

func SingleWrapperTasklistServer(fn TasklistServer) http.HandlerFunc {
	return func(c http.ResponseWriter, req *http.Request) {
		WithOpenDefault(func(tl *Tasklist) {
			fn(c, req, tl)
		})
	}
}

func WrapperServer(sub http.HandlerFunc) http.HandlerFunc {
	return func(c http.ResponseWriter, req *http.Request) {
		defer func() {
			if rerr := recover(); rerr != nil {
				Log(ERROR, "Error while serving:", rerr)
				WriteStackTrace(rerr, LoggerWriter)
				io.WriteString(c, fmt.Sprintf("Internal server error: %s", rerr))
			}
		}()

		if !strings.HasPrefix(req.RemoteAddr, "127.0.0.1") {
			Log(ERROR, "Rejected request from:", req.RemoteAddr)
			return
		}

		if req.Method == "HEAD" {
			//do nothing
		} else {
			sub(c, req)
		}
	}
}

/*
 * Minimal test server
 */
func HelloServer(c http.ResponseWriter, req *http.Request) {
	io.WriteString(c, "hello, world!\n")
}

/*
 * Serves static pages (or 404s)
 */
func StaticInMemoryServer(c http.ResponseWriter, req *http.Request) {
	var ct string
	switch {
	case strings.HasSuffix(req.URL.Path, ".png"):
		ct = "image/png"
	case strings.HasSuffix(req.URL.Path, ".js"):
		ct = "text/javascript; charset=utf-8"
	case strings.HasSuffix(req.URL.Path, ".css"):
		ct = "text/css; charset=utf-8"
	case strings.HasSuffix(req.URL.Path, ".gif"):
		ct = "image/gif"
	case strings.HasSuffix(req.URL.Path, ".woff"):
		ct = "application/font-woff"
	default:
		ct = "text/html; charset=utf-8"
	}

	if req.URL.Path == "/" {
		http.Redirect(c, req, "/list", 301)
	} else if signature := SUMS[req.URL.Path[1:]]; signature == "" {
		Logf(ERROR, "404 Not found\n")
		io.WriteString(c, "404, Not found")
	} else {
		if len(req.Header["If-None-Match"]) > 0 {
			if ifNoneMatch := StripQuotes(req.Header["If-None-Match"][0]); ifNoneMatch == signature {
				Logf(DEBUG, "Page not modified, replying")
				c.WriteHeader(http.StatusNotModified)
				return
			}
		}

		c.Header().Set("ETag", "\""+signature+"\"")
		c.Header().Set("Content-Type", ct)

		io.WriteString(c, decodeStatic(req.URL.Path[1:]))
	}
}

func CheckFormValue(req *http.Request, name string) string {
	v := req.FormValue(name)
	if v == "" {
		panic(fmt.Sprintf("Parameter %s not specified", name))
	}
	return v
}

func CheckBool(in string, name string) bool {
	if in == "true" {
		return true
	} else if in == "false" {
		return false
	} else {
		panic(fmt.Sprintf("Parameter %s not in true or false", name))
	}

	return false
}

func ChangePriorityServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	special := CheckBool(CheckFormValue(req, "special"), "special")

	priority := tl.UpgradePriority(id, special)

	io.WriteString(c, fmt.Sprintf("priority-change-to: %d %s", priority, strings.ToUpper(priority.String())))
}

func GetServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	entry := tl.Get(id)
	addCols := !(req.FormValue("nocols") == "1")
	io.WriteString(c, time.Now().UTC().Format("2006-01-02 15:04:05")+"\n")
	json.NewEncoder(c).Encode(MarshalEntry(entry, tl.GetTimezone(), addCols))
}

func RemoveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	tl.Remove(id)
	io.WriteString(c, "removed")
}

func QaddServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	entry := tl.ParseNew(CheckFormValue(req, "text"), req.FormValue("q"))

	tl.Add(entry)

	isi, parent := IsSubitem(entry.Columns())
	if isi {
		io.WriteString(c, "added: "+entry.Id()+" "+parent)
	} else {
		io.WriteString(c, "added: "+entry.Id())
	}
}

func SaveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	umentry := &UnmarshalEntry{}

	if err := json.NewDecoder(req.Body).Decode(umentry); err != nil {
		panic(err)
	}

	if !tl.Exists(umentry.Id) {
		panic("Specified id does not exists")
	}

	entry := DemarshalEntry(umentry, tl.GetTimezone())

	if CurrentLogLevel <= DEBUG {
		Log(DEBUG, "Saving entry:\n")
		entry.Print()
	}

	tl.Update(entry, false)

	io.WriteString(c, "saved-at-timestamp: "+time.Now().UTC().Format("2006-01-02 15:04:05"))
}

func ErrorLogServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	css := tl.GetSetting("theme")
	errors := tl.RetrieveErrors()

	ErrorLogHeaderHTML(map[string]string{"name": "error log", "theme": css, "code": ""}, c)

	for idx, error := range errors {
		htmlClass := "entry"
		if idx%2 != 0 {
			htmlClass += " oddentry"
		}

		ErrorLogEntryHTML(map[string]string{
			"htmlClass": htmlClass,
			"time":      error.TimeString(),
			"message":   error.Message}, c)
	}

	ErrorLogEnderHTML(nil, c)
}

func ExplainServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	css := tl.GetSetting("theme")

	theselect, code, _, isSavedSearch, isEmpty, showCols, options, err := tl.ParseSearch(req.FormValue("q"), nil)

	myexplain := ""

	myexplain += fmt.Sprintf("Errors: %s\nSaved Search: %v\n\nEmpty: %v\n\nShow Cols: %v\n\nOptions: %v\n\nSQL:\n%s\n\nCODE:\n%s\n\nSQLITE OPCODES:\n", err, isSavedSearch, isEmpty, showCols, options, theselect, code)

	ErrorLogHeaderHTML(map[string]string{"name": "explanation", "theme": css, "code": myexplain}, c)
	ExplainEntryHeaderHTML(nil, c)

	theselect = "EXPLAIN " + theselect

	expls := tl.ExplainRetrieve(theselect)

	for idx, expl := range expls {
		htmlClass := "entry"
		if idx%2 != 0 {
			htmlClass += " oddentry"
		}

		ExplainEntryHTML(map[string]interface{}{
			"htmlClass": htmlClass,
			"explain":   expl}, c)
	}

	ErrorLogEnderHTML(nil, c)
}

func queryForTitle(query string) string {
	split := strings.SplitN(query, "\n", 2)
	queryForTitle := split[0]
	if len(split) > 1 {
		queryForTitle += "…"
	}

	if query == "" {
		queryForTitle = "Index"
	}

	return queryForTitle
}

func headerInfo(tl *Tasklist, pageName string, query string, trigger string, isSavedSearch bool, showOtherLink bool, parseError, retrieveError error, options map[string]string) map[string]interface{} {
	css := tl.GetSetting("theme")
	timezone := tl.GetTimezone()
	removeSearch := ""
	if isSavedSearch {
		removeSearch = "remove-search"
	}
	var otherPageName, otherPageLink string
	if showOtherLink {
		if pageName == "/list" {
			otherPageName = "/list?view=cal"
			otherPageLink = "calendar"
		} else {
			otherPageName = "/list?view=list"
			otherPageLink = "list"
		}
	}

	r := map[string]interface{}{
		"pageName":      pageName,
		"query":         query,
		"queryForTitle": queryForTitle(query),
		"theme":         css,
		"timezone":      fmt.Sprintf("%d", timezone),
		"removeSearch":  removeSearch,
		"retrieveError": retrieveError,
		"parseError":    parseError,
		"otherPageName": otherPageName,
		"otherPageLink": otherPageLink,
	}

	if options != nil {
		if _, ok := options["hidetimecol"]; ok {
			r["hide_etime"] = "do"
		}
		if _, ok := options["hideprioritycol"]; ok {
			r["hide_epr"] = "do"
		}
		if _, ok := options["hidecatscol"]; ok {
			r["hide_ecats"] = "do"
		}
		if _, ok := options["showidcol"]; !ok {
			r["hide_eid"] = "do"
		}
		if _, ok := options["hideprioritychange"]; ok {
			r["hide_prchange"] = "do"
		}
	}

	return r
}

func StatServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	headerInfo := headerInfo(tl, "/list", "", "", false, false, nil, nil, nil)

	CommonHeaderHTML(headerInfo, c)

	StatHeaderHTML(nil, c)

	for i, stat := range tl.GetStatistics() {
		htmlClass := "entry"
		if i%2 == 0 {
			htmlClass += " oddentry"
		}
		StatEntryHTML(map[string]interface{}{"htmlClass": htmlClass, "entry": stat}, c)
	}

	ListEnderHTML(nil, c)
}

func RunServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	commandstr := strings.Replace(req.FormValue("text"), "\r", "", -1)
	command := strings.SplitN(commandstr, " ", -1)

	Logf(INFO, "Running command: "+command[0])

	fentry := tl.Get(command[0])
	tl.DoRunString(fentry.Text(), command[1:len(command)])

	headerInfo := headerInfo(tl, "/list", commandstr, "", false, false, nil, nil, map[string]string{"hideprioritycol": "", "showidcol": "", "hidecatscol": ""})

	CommonHeaderHTML(headerInfo, c)
	EntryListHeaderHTML(nil, c)

	if tl.luaFlags.showReturnValue {
		v, showCols := tl.LuaResultToEntries()
		timezone := tl.GetTimezone()

		if len(v) > 0 {
			EntryListPriorityChangeHTML(map[string]interface{}{"entry": v[0], "colNames": showCols, "PrioritySize": 4}, c)
		}

		for idx, entry := range v {
			htmlClass := "entry"
			if idx%2 != 0 {
				htmlClass += " oddentry"
			}

			cols := []string{}
			for _, colName := range showCols {
				cols = append(cols, entry.Columns()[colName])
			}

			entryEntry := map[string](interface{}){
				"heading":   entry.Id(),
				"entry":     entry,
				"etime":     TimeString(entry.TriggerAt(), entry.Sort(), timezone),
				"ecats":     "",
				"htmlClass": htmlClass,
				"cols":      cols,
			}

			EntryListEntryHTML(entryEntry, c)
		}
	}

	ListEnderHTML(nil, c)
}

func ListJsonServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	var answ ListJsonAnswer
	serializeAnswer := func() {
		if err := json.NewEncoder(c).Encode(answ); err != nil {
			panic(fmt.Sprintf("Error while encoding response: %s", err))
		}
	}

	timezone := tl.GetTimezone()
	query := strings.Replace(req.FormValue("q"), "\r", "", -1)

	theselect, code, _, _, _, _, options, perr := tl.ParseSearch(query, nil)

	answ.ParseError = perr
	if perr != nil {
		serializeAnswer()
		return
	}

	_, incsub := options["sub"]
	v, rerr := tl.Retrieve(theselect, code, incsub)

	answ.RetrieveError = rerr
	if rerr != nil {
		serializeAnswer()
		return
	}

	answ.Results = make([]UnmarshalEntry, 0)

	for _, entry := range v {
		answ.Results = append(answ.Results, *MarshalEntry(entry, timezone, true))
	}

	serializeAnswer()
}

func ListServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	gutsOnly := req.FormValue("guts") != ""
	query := strings.Replace(req.FormValue("q"), "\r", "", -1)
	timezone := tl.GetTimezone()

	theselect, code, trigger, isSavedSearch, _, showCols, options, perr := tl.ParseSearch(query, nil)

	calView := false

	if req.FormValue("view") == "cal" {
		calView = true
	} else if req.FormValue("view") == "list" {
		// list view is default
	} else if _, ok := options["cal"]; ok {
		calView = true
	}

	if calView {
		// we want to use a calendar!
		CalendarServerInner(c, req, tl, trigger, isSavedSearch, perr)
		return
	}

	_, incsub := options["sub"]
	v, rerr := tl.Retrieve(theselect, code, incsub)

	_, subsort := options["ssort"]

	prioritySize := 5

	if _, ok := options["hidetimecol"]; ok {
		prioritySize--
	}
	if _, ok := options["hideprioritycol"]; ok {
		prioritySize--
	}
	if _, ok := options["hidecatscol"]; ok {
		prioritySize--
	}
	if _, ok := options["showidcol"]; ok {
		prioritySize++
	}

	catordering := tl.CategoryDepth()

	headerInfo := headerInfo(tl, "/list", query, trigger, isSavedSearch, true, perr, rerr, options)

	if !gutsOnly {
		CommonHeaderHTML(headerInfo, c)
		EntryListHeaderHTML(nil, c)
	}

	var curp Priority = INVALID
	for idx, entry := range v {
		if (curp != entry.Priority()) && !subsort && (len(v) > 1) {
			EntryListPriorityChangeHTML(map[string]interface{}{"entry": entry, "colNames": showCols, "PrioritySize": prioritySize}, c)
			curp = entry.Priority()
		}

		htmlClass := "entry"
		if idx%2 != 0 {
			htmlClass += " oddentry"
		}

		cols := []string{}
		for _, colName := range showCols {
			cols = append(cols, entry.Columns()[colName])
		}

		entryEntry := map[string](interface{}){
			"heading":   entry.Id(),
			"entry":     entry,
			"etime":     TimeString(entry.TriggerAt(), entry.Sort(), timezone),
			"ecats":     entry.CatString(catordering),
			"htmlClass": htmlClass,
			"cols":      cols,
		}

		text := entry.text
		if len(v) > 1 {
			entry.text = ""
		}

		EntryListEntryHTML(entryEntry, c)

		entry.text = text

		EntryListEntryEditorHTML(entryEntry, c)
	}

	if !gutsOnly {
		ListEnderHTML(nil, c)
	} else {
		ListGutsEnderHTML(map[string]string{"guts": req.FormValue("guts")}, c)
	}
}

func ChildsServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	id := req.FormValue("id")

	objs := childsServerRec(tl, id)

	Must(json.NewEncoder(c).Encode(objs))
}

func NewServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	id := req.FormValue("id")
	asChild := req.FormValue("child")

	var pid string
	if asChild != "0" {
		pid = id
	} else {
		entry := tl.Get(id)
		var ok bool
		ok, pid = IsSubitem(entry.Columns())
		if !ok {
			fmt.Fprintf(c, "error")
			return
		}
	}

	subcol := "sub/" + pid
	childs := tl.GetChildren(pid)
	nentry := tl.ParseNew("#"+subcol, "")
	nentry.SetColumn(subcol, strconv.Itoa(len(childs)))
	tl.Add(nentry)

	if asChild == "0" {
		newchilds := addAfter(nentry.Id(), id, childs)
		tl.UpdateChildren(pid, newchilds)
	}

	io.WriteString(c, nentry.Id())
}

func addAfter(newEl string, after string, items []string) []string {
	newitems := make([]string, len(items)+1)
	d := 0
	added := false
	for i := range items {
		if items[i] == after {
			newitems[d] = items[i]
			d++
			newitems[d] = newEl
			d++
			added = true
		} else if items[i] == newEl {
			// skipped
		} else {
			newitems[d] = items[i]
			d++
		}
	}
	if !added {
		newitems[d] = newEl
		d++
	}
	return newitems[:d]
}

func MoveChildServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	src := req.FormValue("src")
	dst := req.FormValue("dst")
	asChild := req.FormValue("child")

	sentry := tl.Get(src)

	wasChild, spid := IsSubitem(sentry.Columns())

	if wasChild {
		sentry.RemoveColumn("sub/" + spid)
	}

	pid := dst
	if asChild == "0" {
		dentry := tl.Get(dst)
		_, pid = IsSubitem(dentry.Columns())
	}

	siblings := tl.GetChildren(pid)
	sentry.SetColumn("sub/"+pid, strconv.Itoa(len(siblings)))
	tl.Update(sentry, false)

	if asChild == "0" {
		newsiblings := addAfter(src, dst, siblings)
		tl.UpdateChildren(pid, newsiblings)
	}

	if wasChild {
		oldsiblings := tl.GetChildren(spid)
		tl.UpdateChildren(spid, oldsiblings)
	}
}

func childsServerRec(tl *Tasklist, id string) []*Object {
	objs := childsToObjects(tl, id)
	for i := range objs {
		objs[i].Children = childsServerRec(tl, objs[i].Id)
	}
	return objs
}

func CalendarServerInner(c http.ResponseWriter, req *http.Request, tl *Tasklist, trigger string, isSavedSearch bool, perr error) {
	query := req.FormValue("q")

	CommonHeaderHTML(headerInfo(tl, "/cal", query, trigger, isSavedSearch, true, perr, nil, nil), c)
	CalendarHTML(map[string]string{"query": query}, c)
}

func GetCalendarEvents(tl *Tasklist, query string, start, end string, endSecs int64) []EventForJSON {
	pr := tl.ParseEx(query)
	pr = pr.ResolveSavedSearch(tl) // necessary, to modify the result

	pr.AddIncludeClause(&SimpleExpr{":when", "notnull", "", nil, 0, ""})
	pr.AddIncludeClause(&SimpleExpr{":when", ">", start, nil, 0, ""})
	pr.AddIncludeClause(&SimpleExpr{":when", "<", end, nil, 0, ""})
	pr.options["w/done"] = "w/done"
	theselect, _, _ := pr.IntoSelect(tl, nil)
	v, _ := tl.Retrieve(theselect, pr.command, false)

	timezone := tl.GetTimezone()

	r := make([]EventForJSON, 0)

	for _, entry := range v {
		className := fmt.Sprintf("alt%d", entry.CatHash()%6)

		r = append(r, ToCalendarEvent(entry, className, timezone))

		if entry.Priority() != TIMED {
			continue
		}
		if freq := entry.Freq(); freq > 0 {
			for newEntry := entry.NextEntry(""); newEntry.Before(endSecs); newEntry = newEntry.NextEntry("") {
				r = append(r, ToCalendarEvent(newEntry, className, timezone))
			}
		}
	}

	return r
}

func CalendarEventServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	var startSecs, endSecs int64
	var err error
	if startSecs, err = strconv.ParseInt(req.FormValue("start"), 10, 64); err != nil {
		panic(fmt.Sprintf("Error converting start parameter to int %s: %s", req.FormValue("start"), err))
	}
	if endSecs, err = strconv.ParseInt(req.FormValue("end"), 10, 64); err != nil {
		panic(fmt.Sprintf("Error converting start parameter to int %s: %s", req.FormValue("end"), err))
	}

	start := time.Unix(startSecs, 0).Format("2006-01-02")
	end := time.Unix(endSecs, 0).Format("2006-01-02")

	query := req.FormValue("q")

	r := GetCalendarEvents(tl, query, start, end, endSecs)

	Log(DEBUG, "For req:", req, "return:", r)

	if err := json.NewEncoder(c).Encode(r); err != nil {
		panic(fmt.Sprintf("Error while encoding response: %s", err))
	}
}

func HtmlGetServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	entry := tl.Get(id)

	entryEntry := map[string](interface{}){
		"heading": nil,
		"entry":   entry,
		"etime":   TimeString(entry.TriggerAt(), entry.Sort(), tl.GetTimezone()),
		"ecats":   entry.CatString(nil),
		"cols":    []string{},
	}

	EntryListEntryHTML(entryEntry, c)
	io.WriteString(c, "\u2029")
	EntryListEntryEditorHTML(entryEntry, c)
}

func SaveSearchServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	name := req.FormValue("name")
	query := req.FormValue("query")

	if (len(name) > 2) && isQuickTagStart(rune(name[0])) && (name[1] == '%') {
		Logf(INFO, "Converting: %s\n", name)
		name = name[2:len(name)]
		Logf(INFO, "Converting: %s\n", name)
	}

	if query != "" {
		tl.SaveSearch(name, query)
	} else {
		query = tl.GetSavedSearch(name)
	}
	Logf(INFO, "Query: [%s] [%s]", name, query)
	io.WriteString(c, "query-saved: "+query)
}

func RemoveSearchServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	pr := tl.ParseEx(req.FormValue("query"))
	if pr.savedSearch != "" {
		tl.RemoveSaveSearch(pr.savedSearch)
	}
}

func RenTagServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	sourceTag := req.FormValue("from")
	destTag := req.FormValue("to")
	tl.RenameTag(sourceTag, destTag)
	io.WriteString(c, "rename successful")
}

var LONG_OPTION map[string]bool = map[string]bool{
	"setup": true,
}

func OptionServer(c http.ResponseWriter, req *http.Request, multiuserDb *MultiuserDb, tl *Tasklist) {
	if req.FormValue("save") == "save" {
		Must(req.ParseForm())
		settings := tl.GetSettings()
		for k, v := range req.Form {
			if k != "save" {
				settings[k] = v[0]
			}
		}
		tl.SetSettings(settings)

		if settings["setup"] != "" {
			tl.DoString(settings["setup"], nil)
		}
	}

	settings := tl.GetSettings()

	OptionsPageHeader(nil, c)

	for k, v := range settings {
		if LONG_OPTION[k] {
			OptionsLongPageLine(map[string]string{"name": k, "value": v}, c)
		} else {
			OptionsPageLine(map[string]string{"name": k, "value": v}, c)
		}
	}

	if multiuserDb != nil {
		OptionsPageAPITokens(multiuserDb.ListAPITokens(multiuserDb.UsernameFromReq(req)), c)
	}

	OptionsPageEnd(nil, c)
}

func getTagsInOntology(knownTags map[string]bool, ontology []OntologyNodeIn) {
	if ontology == nil {
		return
	}
	for _, on := range ontology {
		knownTags[on.Data] = true
		getTagsInOntology(knownTags, on.Children)
	}
}

func convertOntologyNodeInToOut(on OntologyNodeIn) interface{} {
	if (on.Children == nil) || (len(on.Children) == 0) {
		return on.Data
	}
	return OntologyNodeOut{on.Data, "open", convertOntologyInToOut(on.Children)}
}

func convertOntologyInToOut(ontology []OntologyNodeIn) []interface{} {
	r := []interface{}{}
	for _, on := range ontology {
		r = append(r, convertOntologyNodeInToOut(on))
	}
	return r
}

func ontologyServerGet(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	ontology := ontologyServerGetIn(tl)

	r := convertOntologyInToOut(ontology)

	//r := []interface{}{ "Uno", OntologyNode{ "Due", "open", []interface{}{ "Tre" } }, OntologyNode{ "Tre", "open", []interface{}{} } }
	e := json.NewEncoder(c)
	Must(e.Encode(r))
}

func ontologyServerCheck(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	timezone := tl.GetTimezone()
	errors := tl.OntoCheck(true)
	headerInfo := headerInfo(tl, "/list", "", "", false, true, nil, nil, map[string]string{})
	CommonHeaderHTML(headerInfo, c)
	EntryListHeaderHTML(nil, c)

	first := true

	colNames := []string{"problem", "hint"}

	catordering := tl.CategoryDepth()

	for idx, oee := range errors {
		entry := oee.Entry
		if first {
			EntryListPriorityChangeHTML(map[string]interface{}{"entry": entry, "colNames": colNames, "PrioritySize": 5}, c)
			first = false
		}

		htmlClass := "entry"
		if idx%2 != 0 {
			htmlClass += " oddentry"
		}

		cols := []string{}
		cols = append(cols, oee.ProblemCategory)
		cols = append(cols, oee.ProblemDetail)

		entryEntry := map[string]interface{}{
			"heading":   entry.Id(),
			"entry":     entry,
			"etime":     TimeString(entry.TriggerAt(), entry.Sort(), timezone),
			"ecats":     entry.CatString(catordering),
			"htmlClass": htmlClass,
			"cols":      cols,
		}

		EntryListEntryHTML(entryEntry, c)
		EntryListEntryEditorHTML(entryEntry, c)
	}

	ListEnderHTML(nil, c)
}

func OntologyServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	if req.FormValue("check") == "1" {
		ontologyServerCheck(c, req, tl)
	} else if req.FormValue("move") == "1" {
		src := req.FormValue("src")
		dst := req.FormValue("dst")
		mty := req.FormValue("mty")
		ontology := ontologyServerGetIn(tl)
		if mty == "sibling" {
			ontology = ontologyMoveSibling(src, dst, ontology)
		} else {
			ontology = ontologyMoveChildren(src, dst, ontology)
		}

		mor, err := json.Marshal(ontology)
		Must(err)
		tl.SetSetting("ontology", string(mor))

		c.Write([]byte("ok"))
	} else {
		ontologyServerGet(c, req, tl)
	}

}

func OntologySaveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	or := []OntologyNodeIn{}
	Must(json.NewDecoder(req.Body).Decode(&or))
	//fmt.Printf("Output: %v\n", or)

	mor, err := json.Marshal(or)
	Must(err)
	tl.SetSetting("ontology", string(mor))

	c.Write([]byte("ok"))
}

func SetupHandleFunc(wrapperTasklistServer func(TasklistServer) http.HandlerFunc, wrapperTasklistWithIdServer func(TasklistWithIdServer) http.HandlerFunc, multiuserDb *MultiuserDb) {
	http.HandleFunc("/", WrapperServer(StaticInMemoryServer))
	http.HandleFunc("/static-hello.html", WrapperServer(HelloServer))

	// Entry point urls
	http.HandleFunc("/list", WrapperServer(wrapperTasklistServer(ListServer)))
	http.HandleFunc("/run", WrapperServer(wrapperTasklistServer(RunServer)))
	http.HandleFunc("/stat", WrapperServer(wrapperTasklistServer(StatServer)))
	http.HandleFunc("/opts", WrapperServer(wrapperTasklistServer(
		func(res http.ResponseWriter, req *http.Request, tl *Tasklist) {
			OptionServer(res, req, multiuserDb, tl)
		})))
	http.HandleFunc("/errorlog", WrapperServer(wrapperTasklistServer(ErrorLogServer)))
	http.HandleFunc("/explain", WrapperServer(wrapperTasklistServer(ExplainServer)))

	// List ajax urls
	http.HandleFunc("/change-priority", WrapperServer(wrapperTasklistWithIdServer(ChangePriorityServer)))
	http.HandleFunc("/get", WrapperServer(wrapperTasklistWithIdServer(GetServer)))
	http.HandleFunc("/save", WrapperServer(wrapperTasklistServer(SaveServer)))
	http.HandleFunc("/qadd", WrapperServer(wrapperTasklistServer(QaddServer)))
	http.HandleFunc("/remove", WrapperServer(wrapperTasklistWithIdServer(RemoveServer)))
	http.HandleFunc("/htmlget", WrapperServer(wrapperTasklistWithIdServer(HtmlGetServer)))
	http.HandleFunc("/save-search", WrapperServer(wrapperTasklistServer(SaveSearchServer)))
	http.HandleFunc("/remove-search", WrapperServer(wrapperTasklistServer(RemoveSearchServer)))
	http.HandleFunc("/ontology", WrapperServer(wrapperTasklistServer(OntologyServer)))
	http.HandleFunc("/ontologysave", WrapperServer(wrapperTasklistServer(OntologySaveServer)))

	// Json interface entry points
	http.HandleFunc("/list.json", WrapperServer(wrapperTasklistServer(ListJsonServer)))

	// Calendar ajax urls
	http.HandleFunc("/calevents", WrapperServer(wrapperTasklistServer(CalendarEventServer)))

	// Options support urls
	http.HandleFunc("/rentag", WrapperServer(wrapperTasklistServer(RenTagServer)))

	// New frontend
	http.HandleFunc("/nf/update.json", WrapperServer(wrapperTasklistServer(nfUpdateHandler)))
	http.HandleFunc("/childs.json", WrapperServer(wrapperTasklistServer(ChildsServer)))
	http.HandleFunc("/newsubitem", WrapperServer(wrapperTasklistServer(NewServer)))
	http.HandleFunc("/movechild", WrapperServer(wrapperTasklistServer(MoveChildServer)))
}

func Serve(port string) {
	SetupHandleFunc(SingleWrapperTasklistServer, SingleWrapperTasklistWithIdServer, nil)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		Log(ERROR, "Couldn't serve:", err)
		return
	}
	fmt.Printf("Done serving\n")
}

type EventForJSON map[string]interface{}

func ToCalendarEvent(entry *Entry, className string, timezone int) EventForJSON {
	return map[string]interface{}{
		"id":             entry.Id(),
		"title":          entry.Title(),
		"allDay":         true,
		"start":          TimeFormatTimezone(entry.TriggerAt(), time.RFC3339, timezone),
		"className":      className,
		"ignoreTimezone": true,
	}
}
