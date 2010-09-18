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
	"template"
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

func WrapperServer(sub func(c *http.Conn, req *http.Request)) func(c *http.Conn, req *http.Request) {
	return func(c *http.Conn, req *http.Request) {
		defer func() {
			if rerr := recover(); rerr != nil {
				Log(ERROR, "Error while serving:", rerr)
				io.WriteString(c, fmt.Sprintf("Internal server error: %s", rerr))
			}
		}()

		if !strings.HasPrefix(c.RemoteAddr, "127.0.0.1:") { Log(ERROR, "Rejected request from:", c.RemoteAddr); return }

		Logf(INFO, "REQ\t%s\t%s\n", c.RemoteAddr, req)

		sub(c, req)
	}
}

/*
 * Minimal test server
 */
func HelloServer(c *http.Conn, req *http.Request) {
	io.WriteString(c, "hello, world!\n");
}

/*
 * Serves static pages (or 404s)
 */
func StaticInMemoryServer(c *http.Conn, req *http.Request) {
	var ct string
	switch {
	case strings.HasSuffix(req.URL.Path, ".js"):
		ct = "text/javascript"
	case strings.HasSuffix(req.URL.Path, ".css"):
		ct = "text/css"
	default:
		ct = "text/html"
	}

	c.SetHeader("Content-Type", ct + "; charset=utf-8")
	
	if content := FILES[req.URL.Path[1:]]; content == "" {
		io .WriteString(c, "404, Not found")
	} else {
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

func WithOpenDefaultCheckId(req *http.Request, rest func(tl *Tasklist, id string)) {
	WithOpenDefault(func (tl *Tasklist) {
		id := req.FormValue("id")
		if !tl.Exists(id) { panic(fmt.Sprintf("Non-existent id specified")) }
		rest(tl, id)
	})
}

func ChangePriorityServer(c *http.Conn, req *http.Request) {
	WithOpenDefaultCheckId(req, func (tl *Tasklist, id string) {
		special := CheckBool(CheckFormValue(req, "special"), "special")
	
		priority := tl.UpgradePriority(id, special)

		io.WriteString(c, fmt.Sprintf("priority-change-to: %d %s", priority, strings.ToUpper(priority.String())))
	})
}

func GetServer(c *http.Conn, req *http.Request) {
	WithOpenDefaultCheckId(req, func (tl *Tasklist, id string) {
		entry := tl.Get(id)

		io.WriteString(c, time.LocalTime().Format("2006-01-02 15:04:05") + "\n")
		json.NewEncoder(c).Encode(MarshalEntry(entry))
	})
}

func RemoveServer(c *http.Conn, req *http.Request) {
	WithOpenDefaultCheckId(req, func (tl *Tasklist, id string) {
		tl.Remove(id)
		io.WriteString(c, "removed")
	})
}

func QaddServer(c *http.Conn, req *http.Request) {
	WithOpenDefaultCheckId(req, func (tl *Tasklist, id string) {
		entry, _ := QuickParse(CheckFormValue(req, "text"))
		entry.SetId(tl.MakeRandomId())
		
		tl.Add(entry)
		io.WriteString(c, "added: " + entry.Id())
	})
}

func SaveServer(c *http.Conn, req *http.Request) {
	WithOpenDefault(func (tl *Tasklist) {
		umentry := &UnmarshalEntry{}
		
		if err := json.NewDecoder(req.Body).Decode(umentry); err != nil { panic(err) }

		if !tl.Exists(umentry.Id) { panic("Specified id does not exists") }
	
		entry := DemarshalEntry(umentry)
		
		Log(DEBUG, "Saving entry:\n", entry)
		
		tl.Update(entry);
		
		io.WriteString(c, "saved-at-timestamp: " + time.LocalTime().Format("2006-01-02 15:04:05"))
	})
}

/*
 * Tasklist
 */
func ListServer(c *http.Conn, req *http.Request) {
	WithOpenDefault(func(tl *Tasklist) {
		includeDone := req.FormValue("done") != ""
		var css string
		if req.FormValue("theme") != "" {
			css = req.FormValue("theme")
		} else {
			css = "list.css"
		}
		
		query := req.FormValue("q")
		
		v := tl.Retrieve(SearchParse(query, includeDone, tl))
		
		ListHeaderHTML(map[string]string{ "query": query, "theme": css }, c)
		JavascriptInclude(c, "/shortcut.js")
		JavascriptInclude(c, "/json.js")
		JavascriptInclude(c, "/int.js")
		JavascriptInclude(c, "/calendar.js")
		
		ListHeaderCloseHTML(map[string]string{ "theme": css }, c)
		
		EntryListHeaderHTML(map[string]string{ "query": query, "theme": css }, c)
		
		io.WriteString(c, "<p><table width='100%' id='maintable' style='border-collapse: collapse;'>")
		
		var curp Priority = INVALID
		for _, e := range *v {
			entry := e.(*Entry)
			
			if curp != entry.Priority() {
				EntryListPriorityChangeHTML(entry, c)
				curp = entry.Priority()
			}
			
			entryEntry := map[string](interface{}){
				"entry": entry,
				"theme": css,
				"etime": TimeString(entry.TriggerAt(), entry.Sort()),
			}
			
			
			io.WriteString(c, "    <tr class='entry'>\n")
			EntryListEntryHTML(entryEntry, c)
			io.WriteString(c, "    </tr>\n")
			
			io.WriteString(c, "    <tr id='editor_")
			template.HTMLEscape(c, []byte(entry.Id()))
			io.WriteString(c, "' class='editor' style='display: none'>\n")
			EntryListEntryEditorHTML(entryEntry, c)
			io.WriteString(c, "    </tr>\n")
		}
		
		EntryListFooterHTML(nil, c)
	})
}

func CalendarServer(c *http.Conn, req *http.Request) {
	query := req.FormValue("q")
	CalendarHeaderHTML(map[string]string{ "query": query }, c)
	CalendarHTML(map[string]string{ "query": query }, c)
}

func GetCalendarEvents(tl *Tasklist, query string, r *vector.Vector, start, end string, endSecs int64) {
	v := tl.GetEventList(start, end)

	for _, e := range *v {
		entry := e.(*Entry)

		className := ""
		/*if tlidx % 6 != 0 {
			className = fmt.Sprintf("alt%d", tlidx%6)
		} */
		
		r.Push(ToCalendarEvent(entry, className))
		if (entry.Freq() > 0) && (entry.Priority() == TIMED) {
			for newEntry := entry.NextEntry(""); newEntry.Before(endSecs); newEntry = newEntry.NextEntry("") {
				r.Push(ToCalendarEvent(newEntry, className))
			}
		}
	}
}

func CalendarEventServer(c *http.Conn, req *http.Request) {
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

	WithOpenDefault(func (tl *Tasklist) {
		GetCalendarEvents(tl, query, r, start, end, endSecs)
	})

	Log(DEBUG, "For req:", req, "return:", r)

	if err := json.NewEncoder(c).Encode(r); err != nil {
		panic(fmt.Sprintf("Error while encoding response: %s", err))
	}
}

func HtmlGetServer(c *http.Conn, req *http.Request) {
	WithOpenDefaultCheckId(req, func(tl *Tasklist, id string) {
		entry := tl.Get(id)
		
		entryEntry := map[string](interface{}){
			"entry": entry,
			"etime": TimeString(entry.TriggerAt(), entry.Sort()),
		}

		EntryListEntryHTML(entryEntry, c)
		io.WriteString(c, "\u2029")
		EntryListEntryEditorHTML(entryEntry, c)
	})
}

func Serve(port string) {
	http.HandleFunc("/", WrapperServer(StaticInMemoryServer))
	http.HandleFunc("/static-hello.html", WrapperServer(HelloServer))

	// List urls
	http.HandleFunc("/change-priority", WrapperServer(ChangePriorityServer))
	http.HandleFunc("/get", WrapperServer(GetServer))
	http.HandleFunc("/list", WrapperServer(ListServer))
	http.HandleFunc("/save", WrapperServer(SaveServer))
	http.HandleFunc("/qadd", WrapperServer(QaddServer))
	http.HandleFunc("/remove", WrapperServer(RemoveServer))
	http.HandleFunc("/htmlget", WrapperServer(HtmlGetServer))

	// Calendar urls
	http.HandleFunc("/cal", WrapperServer(CalendarServer))
	http.HandleFunc("/calevents", WrapperServer(CalendarEventServer))
	
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
 		Log(ERROR, "Couldn't serve:", err)
		return
	}
	fmt.Printf("Done serving\n")
}

