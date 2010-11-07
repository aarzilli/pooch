/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"encoding/base64"
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

func DecodeBase64(in string) string {
	decbuf := make([]byte, base64.StdEncoding.DecodedLen(len(in)))
	base64.StdEncoding.Decode(decbuf, []byte(in))
	return string(decbuf)
}

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

		if !strings.HasPrefix(c.RemoteAddr(), "127.0.0.1:") { Log(ERROR, "Rejected request from:", c.RemoteAddr()); return }

		Logf(INFO, "REQ\t%s\t%s\n", c.RemoteAddr(), req)

		if req.Method == "HEAD" {
			//do nothing
		} else {
			sub(c, req)
		}

		Logf(INFO, "QER\t%s\t%s\n", c.RemoteAddr(), req)
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
	case strings.HasSuffix(req.URL.Path, ".js"):
		ct = "text/javascript"
	case strings.HasSuffix(req.URL.Path, ".css"):
		ct = "text/css"
	default:
		ct = "text/html"
	}
	
	if req.URL.Path == "/" {
		http.Redirect(c, req, "/list", 301)
	} else if signature := SUMS[req.URL.Path[1:]]; signature == "" {
		io .WriteString(c, "404, Not found")
	} else {
		if ifNoneMatch := StripQuotes(req.Header["If-None-Match"]); ifNoneMatch == signature {
			Logf(DEBUG, "Page not modified, replying")
			c.WriteHeader(http.StatusNotModified)
			return
		}

		c.SetHeader("ETag", "\"" + signature + "\"")
		c.SetHeader("Content-Type", ct + "; charset=utf-8")

		content := FILES[req.URL.Path[1:]]
		z := DecodeBase64(content)
		var i int
		for i = len(z)-1; i > 0; i-- {
			if z[i] != 0 {
				break
			}
		}
		z = z[0:i+1]
		io.WriteString(c, z);
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
	io.WriteString(c, time.LocalTime().Format("2006-01-02 15:04:05") + "\n")
	json.NewEncoder(c).Encode(MarshalEntry(entry))
}

func RemoveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist, id string) {
	tl.Remove(id)
	io.WriteString(c, "removed")
}

func QaddServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	entry, _ := QuickParse(CheckFormValue(req, "text"), req.FormValue("q"), tl)
	entry.SetId(tl.MakeRandomId())
	
	tl.Add(entry)
	io.WriteString(c, "added: " + entry.Id())
}

func SaveServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	umentry := &UnmarshalEntry{}
	
	if err := json.NewDecoder(req.Body).Decode(umentry); err != nil { panic(err) }
	
	if !tl.Exists(umentry.Id) { panic("Specified id does not exists") }
	
	entry := DemarshalEntry(umentry)
	
	if CurrentLogLevel <= DEBUG {
		Log(DEBUG, "Saving entry:\n")
		entry.Print()
	}
	
	tl.Update(entry, false);
	
	io.WriteString(c, "saved-at-timestamp: " + time.LocalTime().Format("2006-01-02 15:04:05"))
}

func ShowSubcols(c http.ResponseWriter, query string, tl *Tasklist) {
	SubcolEntryHTML(map[string]string{"name": "index", "dst": ""}, c)

	for _, v := range tl.GetSavedSearches() {
		SubcolEntryHTML(map[string]string{"name": "@%"+v, "dst": "@%"+v}, c)
	}

	io.WriteString(c, "<hr/>\n")
	
	for _, v := range tl.GetSubcols("") {
		SubcolEntryHTML(map[string]string{"name": "@"+v, "dst": "@"+v}, c)
	}

	Logf(DEBUG, "Query is: %s\n", query)

	if len(query) > 0 && isQuickTagStart(query[0]) && strings.IndexAny(query, " ") == -1 {
		Logf(DEBUG, "Adding stuff in\n");
		set := make(map[string]string)
		_, theselect := SearchParseToken("+"+query, tl, set, false)
		subcols := tl.GetSubcols(theselect);

		for _, v := range subcols {
			if _, ok := set[v]; ok { continue }
			dst := fmt.Sprintf("%s@%s", query, v)
			SubcolEntryHTML(map[string]string{"name": dst, "dst": dst}, c)
		}
	}
}

/*
 * Tasklist
 */
func ListServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	css := tl.GetSetting("theme")
	includeDone := req.FormValue("done") != ""
	query := req.FormValue("q")
	includeDoneStr := ""; if includeDone { includeDoneStr = "checked" }
	
	v := tl.Retrieve(SearchParse(query, includeDone, false, nil, tl))
	
	ListHeaderHTML(map[string]string{ "query": query, "theme": css, "includeDone": includeDoneStr }, c)
	ShowSubcols(c, query, tl)
	SubcolsEnder(map[string]string{ }, c)
	
	var curp Priority = INVALID
	for _, entry := range v {
		if curp != entry.Priority() {
			EntryListPriorityChangeHTML(entry, c)
			curp = entry.Priority()
		}
		
		entryEntry := map[string](interface{}){
			"entry": entry,
			"etime": TimeString(entry.TriggerAt(), entry.Sort()),
			"ecats": entry.CatString(),
		}
		
		EntryListEntryHTML(entryEntry, c)
		EntryListEntryEditorHTML(entryEntry, c)
	}

	ListEnderHTML(nil, c)
}

func CalendarServer(c http.ResponseWriter, req *http.Request) {
	query := req.FormValue("q")
	CalendarHeaderHTML(map[string]string{ "query": query }, c)
	CalendarHTML(map[string]string{ "query": query }, c)
}

func GetCalendarEvents(tl *Tasklist, query string, r *vector.Vector, start, end string, endSecs int64) {
	theselect, query := SearchParse(query, true, false, []string { "tasks.trigger_at_field IS NOT NULL", "tasks.trigger_at_field > " + tl.Quote(start), "tasks.trigger_at_field < " + tl.Quote(end) }, tl)
	v := tl.Retrieve(theselect, query)

	for _, entry := range v {
		className := fmt.Sprintf("alt%d", entry.CatHash() % 6)

		r.Push(ToCalendarEvent(entry, className))
		if (entry.Freq() > 0) && (entry.Priority() == TIMED) {
			for newEntry := entry.NextEntry(""); newEntry.Before(endSecs); newEntry = newEntry.NextEntry("") {
				r.Push(ToCalendarEvent(newEntry, className))
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

	start := time.SecondsToLocalTime(startSecs).Format("2006-01-02")
	end := time.SecondsToLocalTime(endSecs).Format("2006-01-02")

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
		"entry": entry,
		"etime": TimeString(entry.TriggerAt(), entry.Sort()),
		"ecats": entry.CatString(),
	}
	
	EntryListEntryHTML(entryEntry, c)
	io.WriteString(c, "\u2029")
	EntryListEntryEditorHTML(entryEntry, c)
}

func SaveSearchServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	name := req.FormValue("name")
	query := req.FormValue("query")
	tl.SaveSearch(name, query)
	io.WriteString(c, "query-saved: " + name)
}

func OptionServer(c http.ResponseWriter, req *http.Request, tl *Tasklist) {
	if req.FormValue("save") == "save" {
		must(req.ParseForm())
		settings := make(map[string]string)
		for k, v := range req.Form {
			if k != "save" { settings[k] = v[0] }
		}
		tl.SetSettings(settings)
	}
	
	settings := tl.GetSettings()

	OptionsPageHeader(nil, c)
	
	for k, v := range settings {
		OptionsPageLine(map[string]string{ "name": k, "value": v }, c)
	}
	
	OptionsPageEnd(nil, c)
}

func SetupHandleFunc(wrapperTasklistServer func(TasklistServer)http.HandlerFunc, wrapperTasklistWithIdServer func(TasklistWithIdServer)http.HandlerFunc) {
	http.HandleFunc("/", WrapperServer(StaticInMemoryServer))
	http.HandleFunc("/static-hello.html", WrapperServer(HelloServer))

	// List urls
	http.HandleFunc("/change-priority", WrapperServer(wrapperTasklistWithIdServer(ChangePriorityServer)))
	http.HandleFunc("/get", WrapperServer(wrapperTasklistWithIdServer(GetServer)))
	http.HandleFunc("/list", WrapperServer(wrapperTasklistServer(ListServer)))
	http.HandleFunc("/save", WrapperServer(wrapperTasklistServer(SaveServer)))
	http.HandleFunc("/qadd", WrapperServer(wrapperTasklistServer(QaddServer)))
	http.HandleFunc("/remove", WrapperServer(wrapperTasklistWithIdServer(RemoveServer)))
	http.HandleFunc("/htmlget", WrapperServer(wrapperTasklistWithIdServer(HtmlGetServer)))
	http.HandleFunc("/save-search", WrapperServer(wrapperTasklistServer(SaveSearchServer)))
	http.HandleFunc("/opts", WrapperServer(wrapperTasklistServer(OptionServer)))

	// Calendar urls
	http.HandleFunc("/cal", WrapperServer(CalendarServer))
	http.HandleFunc("/calevents", WrapperServer(wrapperTasklistServer(CalendarEventServer)))
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

