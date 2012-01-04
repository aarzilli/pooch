/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"http"
	"io"
	"fmt"
	"strings"
	"json"
	"time"
	"container/vector"
	"strconv"
	"os"
)

type TasklistWithIdServer func(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string)
type TasklistServer func(c http.ResponseWriter, req *http.Request, tl *Tasklist)


func SingleWrapperTasklistWithIdServer(fn TasklistWithIdServer) http.HandlerFunc {
	return func (c http.ResponseWriter, req *http.Request) {
		WithOpenDefault(func(tl *Tasklist) {
			id := req.FormValue("id")
			if !tl.Exists(id) { panic(fmt.Sprintf("Non-existent id specified")) }
			fn(c, req, tl, id)
		})
	}
}

func SingleWrapperTasklistServer(fn TasklistServer) http.HandlerFunc {
	return func (c http.ResponseWriter, req *http.Request) {
		WithOpenDefault(func (tl *Tasklist) {
			fn(c, req, tl)
		})
	}
}

func WrapperServer(sub http.HandlerFunc) http.HandlerFunc {
	return func(c http.ResponseWriter, req *http.Request) {
		defer func() {
			if rerr := recover(); rerr != nil {
				Log(ERROR, "Error while serving:", rerr)
				WriteStackTrace(rerr, loggerWriter)
				io.WriteString(c, fmt.Sprintf("Internal server error: %s", rerr))
			}
		}()

		if !strings.HasPrefix(req.RemoteAddr, "127.0.0.1") { Log(ERROR, "Rejected request from:", req.RemoteAddr); return }

		Logf(INFO, "REQ\t%s\t%s\n", req.RemoteAddr, req)

		if req.Method == "HEAD" {
			//do nothing
		} else {
			sub(c, req)
		}

		Logf(INFO, "QER\t%s\t%s\n", req.RemoteAddr, req)
	}
}

/*
 * Minimal test server
 */
func HelloServer(c http.ResponseWriter, req *http.Request) {
	io.WriteString(c, "hello, world!\n");
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
	default:
		ct = "text/html; charset=utf-8"
	}

	if req.URL.Path == "/" {
		http.Redirect(c, req, "/list", 301)
	} else if signature := SUMS[req.URL.Path[1:]]; signature == "" {
		Logf(ERROR, "404 Not found\n")
		io.WriteString(c, "404, Not found")
	} else {
		Logf(ERROR, "Serving")
		if len(req.Header["If-None-Match"]) > 0 {
			if ifNoneMatch := StripQuotes(req.Header["If-None-Match"][0]); ifNoneMatch == signature {
				Logf(DEBUG, "Page not modified, replying")
				c.WriteHeader(http.StatusNotModified)
				return
			}
		}

		c.Header().Set("ETag", "\"" + signature + "\"")
		c.Header().Set("Content-Type", ct)

		io.WriteString(c, decodeStatic(req.URL.Path[1:]));
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
	io.WriteString(c, time.UTC().Format("2006-01-02 15:04:05") + "\n")
	json.NewEncoder(c).Encode(MarshalEntry(entry, tl.GetTimezone()))
}

func RemoveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	tl.Remove(id)
	io.WriteString(c, "removed")
}

func QaddServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	entry := tl.ParseNew(CheckFormValue(req, "text"), req.FormValue("q"))

	tl.Add(entry)
	io.WriteString(c, "added: " + entry.Id())
}

func SaveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	umentry := &UnmarshalEntry{}

	if err := json.NewDecoder(req.Body).Decode(umentry); err != nil { panic(err) }

	if !tl.Exists(umentry.Id) { panic("Specified id does not exists") }

	entry := DemarshalEntry(umentry, tl.GetTimezone())

	if CurrentLogLevel <= DEBUG {
		Log(DEBUG, "Saving entry:\n")
		entry.Print()
	}

	tl.Update(entry, false, false);

	io.WriteString(c, "saved-at-timestamp: " + time.UTC().Format("2006-01-02 15:04:05"))
}

func ErrorLogServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	css := tl.GetSetting("theme")
	errors := tl.RetrieveErrors()

	ErrorLogHeaderHTML(map[string]string{ "name": "error log", "theme": css, "code": "" }, c)

	for idx, error := range errors {
		htmlClass := "entry"
		if idx % 2 != 0 {
			htmlClass += " oddentry"
		}

		ErrorLogEntryHTML(map[string]string{
			"htmlClass": htmlClass,
			"time": error.TimeString(),
			"message": error.Message }, c)
	}

	ErrorLogEnderHTML(nil, c)
}




func ExplainServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	css := tl.GetSetting("theme")

	theselect, code, _, isSavedSearch, isEmpty, showCols, options, err := tl.ParseSearch(req.FormValue("q"), nil)

	myexplain := ""

	myexplain += fmt.Sprintf("Errors: %s\nSaved Search: %v\n\nEmpty: %v\n\nShow Cols: %v\n\nOptions: %v\n\nSQL:\n%s\n\nCODE:\n%s\n\nSQLITE OPCODES:\n", err, isSavedSearch, isEmpty, showCols, options, theselect, code)
	
	ErrorLogHeaderHTML(map[string]string{ "name": "explanation", "theme": css, "code": myexplain }, c)
	ExplainEntryHeaderHTML(nil, c)

	theselect = "EXPLAIN " + theselect

	expls := tl.ExplainRetrieve(theselect)

	for idx, expl := range expls {
		htmlClass := "entry"
		if idx % 2 != 0 {
			htmlClass += " oddentry"
		}

		ExplainEntryHTML(map[string]interface{}{
			"htmlClass": htmlClass,
			"explain": expl }, c)
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

func headerInfo(tl *Tasklist, pageName string, query string, trigger string, isSavedSearch bool, showOtherLink bool, parseError, retrieveError os.Error, options map[string]string) map[string]interface{} {
	css := tl.GetSetting("theme")
	timezone := tl.GetTimezone()
	removeSearch := ""; if isSavedSearch { removeSearch = "remove-search" }
	var otherPageName, otherPageLink string
	if showOtherLink {
		if pageName == "/list" {
			otherPageName = "/cal"
			otherPageLink = "calendar"
		} else {
			otherPageName = "/list"
			otherPageLink = "list"
		}
	}

	savedSearches := tl.GetSavedSearches()
	subtags := tl.subcolumns[trigger]
	toplevel := make([]string, 0)
	for _, tag := range tl.GetTags() {
		if !tl.ignoreColumn[tag] {
			toplevel = append(toplevel, "#"+tag)
		}
	}

	r := map[string]interface{}{
		"pageName": pageName,
		"query": query,
		"queryForTitle": queryForTitle(query),
		"theme": css,
		"timezone": fmt.Sprintf("%d", timezone),
		"removeSearch": removeSearch,
		"retrieveError": retrieveError,
		"parseError": parseError,
		"otherPageName": otherPageName,
		"otherPageLink": otherPageLink,
		
		"savedSearches": savedSearches,
		"subtags": subtags,
		"toplevel": toplevel,
	};

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
	
	ListHeaderHTML(headerInfo, c)
	CommonHeaderHTML(headerInfo, c)

	StatHeaderHTML(nil, c)

	for i, stat := range tl.GetStatistics() {
		htmlClass := "entry"
		if i % 2 == 0 {
			htmlClass += " oddentry"
		}
		StatEntryHTML(map[string]interface{}{ "htmlClass": htmlClass, "entry": stat }, c)
	}

	ListEnderHTML(nil, c)
}

func RunServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	commandstr := strings.Replace(req.FormValue("text"), "\r", "", -1)
	command := strings.SplitN(commandstr, " ", -1)

	Logf(INFO, "Running command: " + command[0])

	fentry := tl.Get(command[0])
	tl.DoRunString(fentry.Text(), command[1:len(command)])
	
	headerInfo := headerInfo(tl, "/list", commandstr, "", false, false, nil, nil, map[string]string{ "hideprioritycol": "", "showidcol": "", "hidecatscol": "" })

	//TODO: options?

	ListHeaderHTML(headerInfo, c)
	CommonHeaderHTML(headerInfo, c)
	EntryListHeaderHTML(nil, c)

	if tl.luaFlags.showReturnValue {
		v, showCols := tl.LuaResultToEntries()
		timezone := tl.GetTimezone()

		if len(v) > 0 {
			EntryListPriorityChangeHTML(map[string]interface{}{ "entry": v[0], "colNames": showCols, "PrioritySize": 4 }, c)
		}
		
		for idx, entry := range v {
			htmlClass := "entry"
			if idx % 2 != 0 {
				htmlClass += " oddentry"
			}
			
			cols := []string{}
			for _,  colName := range showCols {
				cols = append(cols, entry.Columns()[colName])
			}

			entryEntry := map[string](interface{}){
				"heading": entry.Id(),
				"entry": entry,
				"etime": TimeString(entry.TriggerAt(), entry.Sort(), timezone),
				"ecats": "",
				"htmlClass": htmlClass,
				"cols": cols,
			}
			
			EntryListEntryHTML(entryEntry, c)
		}
	} 

	ListEnderHTML(nil, c)
}


func ListServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	query := strings.Replace(req.FormValue("q"), "\r", "", -1)
	timezone := tl.GetTimezone()

	theselect, code, trigger, isSavedSearch, _, showCols, options, perr := tl.ParseSearch(query, nil)

	/* Will not allow adding elements to an empty page
	if isEmpty {
		http.Redirect(c, req, "/stat", 301)
		return
	}*/
	
	v, rerr := tl.Retrieve(theselect, code)

	prioritySize := 5

	if _, ok := options["hidetimecol"]; ok { prioritySize-- }
	if _, ok := options["hideprioritycol"]; ok { prioritySize-- }
	if _, ok := options["hidecatscol"]; ok { prioritySize-- }
	if _, ok := options["showidcol"]; ok { prioritySize++ }

	headerInfo := headerInfo(tl, "/list", query, trigger, isSavedSearch, true, perr, rerr, options)
	
	ListHeaderHTML(headerInfo, c)
	CommonHeaderHTML(headerInfo, c)
	EntryListHeaderHTML(nil, c)
	
	var curp Priority = INVALID
	for idx, entry := range v {
		if curp != entry.Priority() {
			EntryListPriorityChangeHTML(map[string]interface{}{ "entry": entry, "colNames": showCols, "PrioritySize": prioritySize }, c)
			curp = entry.Priority()
		}

		htmlClass := "entry"
		if idx % 2 != 0 {
			htmlClass += " oddentry"
		}

		cols := []string{}
		for _,  colName := range showCols {
			cols = append(cols, entry.Columns()[colName])
		}		

		entryEntry := map[string](interface{}){
			"heading": entry.Id(),
			"entry": entry,
			"etime": TimeString(entry.TriggerAt(), entry.Sort(), timezone),
			"ecats": entry.CatString(),
			"htmlClass": htmlClass,
			"cols": cols,
		}
		
		EntryListEntryHTML(entryEntry, c)
		EntryListEntryEditorHTML(entryEntry, c)
	}

	ListEnderHTML(nil, c)
}

func CalendarServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	query := req.FormValue("q")
	_, _, trigger, isSavedSearch, _, _, _, err := tl.ParseSearch(query, nil)

	theme := tl.GetSetting("theme")

	CalendarHeaderHTML(map[string]string{ "query": query, "theme": theme  }, c)
	CommonHeaderHTML(headerInfo(tl, "/cal", query, trigger, isSavedSearch, true, err, nil, nil), c)
	CalendarHTML(map[string]string{ "query": query }, c)
}

func GetCalendarEvents(tl *Tasklist, query string, r *vector.Vector, start, end string, endSecs int64) {
	pr := tl.ParseEx(query)
	pr = pr.ResolveSavedSearch(tl) // necessary, to modify the result
	
	pr.AddIncludeClause(&SimpleExpr{ ":when", "notnull", "", nil, 0, "" })
	pr.AddIncludeClause(&SimpleExpr{ ":when", ">", start, nil, 0, ""  })
	pr.AddIncludeClause(&SimpleExpr{ ":when", "<", end, nil, 0, "" })
	pr.options["w/done"] = "w/done"
	theselect, _, _ := pr.IntoSelect(tl, nil)
	v, _ := tl.Retrieve(theselect, pr.command)

	timezone := tl.GetTimezone()

	for _, entry := range v {
		className := fmt.Sprintf("alt%d", entry.CatHash() % 6)

		r.Push(ToCalendarEvent(entry, className, timezone))

		if entry.Priority() != TIMED { continue }
		if freq := entry.Freq(); freq > 0 {
			for newEntry := entry.NextEntry(""); newEntry.Before(endSecs); newEntry = newEntry.NextEntry("") {
				r.Push(ToCalendarEvent(newEntry, className, timezone))
			}
		}
	}
}

func CalendarEventServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	var startSecs, endSecs int64
	var err os.Error
	if startSecs, err = strconv.Atoi64(req.FormValue("start")); err != nil {
		panic(fmt.Sprintf("Error converting start parameter to int %s: %s", req.FormValue("start"), err))
	}
	if endSecs, err = strconv.Atoi64(req.FormValue("end")); err != nil {
		panic(fmt.Sprintf("Error converting start parameter to int %s: %s", req.FormValue("end"), err))
	}

	start := time.SecondsToUTC(startSecs).Format("2006-01-02")
	end := time.SecondsToUTC(endSecs).Format("2006-01-02")

	r := new(vector.Vector)

	query := req.FormValue("q")

	GetCalendarEvents(tl, query, r, start, end, endSecs)

	Log(DEBUG, "For req:", req, "return:", r)

	if err := json.NewEncoder(c).Encode(r); err != nil {
		panic(fmt.Sprintf("Error while encoding response: %s", err))
	}
}

func HtmlGetServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	entry := tl.Get(id)

	entryEntry := map[string](interface{}){
		"heading": nil,
		"entry": entry,
		"etime": TimeString(entry.TriggerAt(), entry.Sort(), tl.GetTimezone()),
		"ecats": entry.CatString(),
		"cols": []string{},
	}
	
	EntryListEntryHTML(entryEntry, c)
	io.WriteString(c, "\u2029")
	EntryListEntryEditorHTML(entryEntry, c)
}

func SaveSearchServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	name := req.FormValue("name")
	query := req.FormValue("query")

	if (len(name) > 2) && isQuickTagStart(int(name[0])) && (name[1] == '%') {
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
	io.WriteString(c, "query-saved: " + query)
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

func OptionServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	if req.FormValue("save") == "save" {
		must(req.ParseForm())
		settings := make(map[string]string)
		for k, v := range req.Form {
			if k != "save" { settings[k] = v[0] }
		}
		tl.SetSettings(settings)

		tl.ResetSetup()
		if settings["setup"] != "" {
			tl.DoString(settings["setup"], nil)
		}
	}
	
	settings := tl.GetSettings()

	OptionsPageHeader(nil, c)
	
	for k, v := range settings {
		if LONG_OPTION[k] {
			OptionsLongPageLine(map[string]string{ "name": k, "value": v }, c)
		} else {
			OptionsPageLine(map[string]string{ "name": k, "value": v }, c)
		}
	}
	
	OptionsPageEnd(nil, c)
}

func SetupHandleFunc(wrapperTasklistServer func(TasklistServer)http.HandlerFunc, wrapperTasklistWithIdServer func(TasklistWithIdServer)http.HandlerFunc) {
	http.HandleFunc("/", WrapperServer(StaticInMemoryServer))
	http.HandleFunc("/static-hello.html", WrapperServer(HelloServer))

	// Entry point urls
	http.HandleFunc("/list", WrapperServer(wrapperTasklistServer(ListServer)))
	http.HandleFunc("/run", WrapperServer(wrapperTasklistServer(RunServer)))
	http.HandleFunc("/stat", WrapperServer(wrapperTasklistServer(StatServer)))
	http.HandleFunc("/cal", WrapperServer(wrapperTasklistServer(CalendarServer)))
	http.HandleFunc("/opts", WrapperServer(wrapperTasklistServer(OptionServer)))
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

	// Calendar ajax urls
	http.HandleFunc("/calevents", WrapperServer(wrapperTasklistServer(CalendarEventServer)))

	// Options support urls
	http.HandleFunc("/rentag", WrapperServer(wrapperTasklistServer(RenTagServer)))
}

func Serve(port string) {
	SetupHandleFunc(SingleWrapperTasklistServer, SingleWrapperTasklistWithIdServer)
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
 		Log(ERROR, "Couldn't serve:", err)
		return
	}
	fmt.Printf("Done serving\n")
}

func ToCalendarEvent(entry *Entry, className string, timezone int) map[string]interface{} {
	return map[string]interface{}{
		"id": entry.Id(),
		"title": entry.Title(),
		"allDay": true,
		"start": TimeFormatTimezone(entry.TriggerAt(), time.RFC3339, timezone),
		"className": className,
		"ignoreTimezone": true,
	}
}

