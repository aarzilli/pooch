/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch

import (
	"text/template"
	"io"
	"strings"
	"net/url"
)

func PriorityFormatter(input Priority) string {
	return strings.ToUpper(input.String())
}

var formatters template.FuncMap = template.FuncMap{
	"priority": PriorityFormatter,
	"url": url.QueryEscape,
}

type ExecutableTemplate func(interface{}, io.Writer)

func WrapTemplate(t *template.Template) ExecutableTemplate {
	return func(data interface{}, wr io.Writer) {
		Must(t.Execute(wr, data))
	}
}

func MakeExecutableTemplate(name string, rawTemplate string) ExecutableTemplate {
	t := template.New(name)
	t.Funcs(formatters)
	template.Must(t.Parse(rawTemplate))
	return WrapTemplate(t)
}

var CommonHeaderHTML ExecutableTemplate = MakeExecutableTemplate("CommonHeader", `
  <script>
    thisPage = "{{.pageName}}";
  </script>
  <div class='advmenu'>
    <a href='/opts'>options</a>
    &nbsp;
    <a href="/advanced.html">advanced operations</a><br/>
    Current timezone: {{.timezone|html}}
  </div>
  <h2 style='display: inline;'>{{.queryForTitle|html}}</h2>
  <div class='mainmenu'>
    <div class='mainmenu_item'>
      <a href='javascript:toggle_searchpop()'>[change query]</a>
      <div id='searchpop' class='popup' style='display: none'>
         <form id='searchform' method='get' action='{{.pageName}}'>
           <label for='q'>Query:</label>&nbsp;
           <textarea name='q' id='q' cols='50' rows='10'>{{.query|html}}</textarea>
           <input type='submit' value='search'>
           <input type='button' value='cancel' onclick='javascript:toggle_searchpop()'/>
           &nbsp;
           <div class='popbuttons'>
           {{if .removeSearch }}
             <input type='button' id='editquerybtn' value='Edit query' onClick='javascript:editsearch()'/>
             <input type='button' value='Remove query' onClick='javascript:removesearch()'/>
           {{else}}
             <input type='button' value='Save query' onClick='javascript:savesearch()'/>
           {{end}}
           </div>
           <div class='keyinfo'>(press alt-enter to search)</div>
         </form>
      </div>
    </div>
    {{if .otherPageName}}
	    <div class='mainmenu_item'>
	      <a href='javascript:toggle_addpop()'>[add entry]</a>
	      <div id='addpop' class='popup' style='display: none'>
	        <form onsubmit='return add_entry("{{.query|html}}")'>
	          <label for='newentry'>New entry:</label>&nbsp;
	          <input size='50' type='text' id='newentry' name='text'/>
	          <input type='button' value='add' onclick='javascript:add_entry("{{.query|html}}")'/>
	          <input type='button' value='cancel' onclick='javascript:toggle_addpop()'/>
	        </form>
	      </div>
	    </div>
	    <div class='mainmenu_item'>
	      <a href="{{.otherPageName}}?q={{.query|url}}">[see as {{.otherPageLink}}]</a>
	    </div>
	    <div class='mainmenu_item'>
	      <a href="/explain?q={{.query|url}}">[see explanation]</a>
	    </div>
    {{end}}
    <div class='mainmenu_item'>
      <a href='javascript:toggle_navpop()'>[navigation]</a>
      <div id='navpop' class='popup' style='display: none'>
         <ul class='navlist'>
           <li><a href='{{.pageName}}?q='>index</a></li>
           {{with $parent := .}}
              {{range .savedSearches}}
              <li><a href="{{$parent.pageName}}?q=%23%25{{.|url}}">#%{{.|html}}</a></li>
              {{end}}
           {{end}}
         </ul>
         <hr/>
         <ul class='navlist'>
           {{with $parent := .}}
             {{range .subtags}}
                <li><a href="{{$parent.pageName}}?q={{.|url}}">{{.|html}}</a></li>
             {{else}}
             {{end}}
           {{end}}
         </ul>
         <hr/>
         <ul class='navlist'>
           {{with $parent := .}}
             {{range .toplevel}}
               <li><a href="{{$parent.pageName}}?q={{.|url}}">{{.|html}}</a></li>
             {{else}}
             {{end}}
           {{end}}
        </ul>
      </div>
    </div>
    <div class='mainmenu_item'>
      <a href='javascript:toggle_runpop()'>[run]</a>
      <div id='runpop' class='popup' style='display: none'>
        <form method='get' action='/run'>
          <label for='runcmd'>Command:</label>&nbsp;
          <input size='50' type='text' id='runcmd' name='text'/>
          <input type='submit' value='run'/>
        </form>
      </div>
    </div>
    <div class='mainmenu_item'>
       <a href='/stat'>[statistics]</a>
    </div>
  </div>

  {{if .parseError}}
    <div class='screrror'>Error while executing search: {{.parseError|html}} <a href='/errorlog'>Full error log</a></div>
  {{end}}
  {{if .retrieveError}}
    <div class='screrror'>Error while executing search: {{.retrieveError|html}} <a href='/errorlog'>Full error log</a></div>
  {{end}}
`)

var EntryListHeaderHTML ExecutableTemplate = MakeExecutableTemplate("EntryListHeader", `
  <table class='maintable' id='maintable'>
`)

var ListHeaderHTML ExecutableTemplate = MakeExecutableTemplate("ListHeader", `
<!DOCTYPE html>
<html>
<head>
  <title>Pooch: {{.queryForTitle|html}}</title>
  <link type='text/css' rel='stylesheet' href='listcommon.css'>
  <link type='text/css' rel='stylesheet' href='{{.theme}}'>
  <link type='text/css' rel='stylesheet' href='calendar.css'>
  <link type='image/png' rel='icon' href='animals-dog.png'>
  <script src='/jquery.js'></script>
  <script src='/int.js'></script>
  <script src='/calendar.js'></script>
  <style>
     {{if .hide_eid}}
        .eid {
           display: none;
		}
     {{end}}
     {{if .hide_etime}}
        .etime {
            display: none;
        }
     {{end}}
     {{if .hide_epr}}
        .epr {
            display: none;
        }
     {{end}}
     {{if .hide_prchange}}
        .prchange {
            visibility: hidden;
        }
     {{end}}
     {{if .hide_ecats}}
        .ecats {
            display: none;
        }
     {{end}}
  </style>
</head>
<body onkeydown='keytable(event)' onkeypress='keypress(event)'>
`)

var EntryListPriorityChangeHTML ExecutableTemplate = MakeExecutableTemplate("EntryListPriorityChange", `
    <tr>
      <td class='prchange' colspan='{{.PrioritySize|html}}'>{{.entry.Priority|priority}}</td>
      {{range .colNames}}
      <td class='colname'>{{.|html}}</td>
      {{else}}
      {{end}}
    </tr>
`)

var EntryListEntryHTML ExecutableTemplate = MakeExecutableTemplate("EntryListEntry", `
   {{if .heading}}
    <tr class='{{.htmlClass}}'>
   {{end}}
    {{with .entry}}
      <td class='eid'>{{.Id|html}}</td>

      <td class='etitle' onclick='javascript:toggle_editor("{{.Id|html}}", event)'><a href='javascript:toggle_editor("{{.Id|html}}", event)'>{{.Title|html}}</a></td>

      <td class='epr'>
        <input type='button' class='prioritybutton priorityclass_{{.Priority|priority}}' id='epr_{{.Id|html}}' value='{{.Priority|priority}}' onclick='javascript:change_priority("{{.Id|html}}", event)'/>
      </td>

      <td class='eloading'><img id='ploading_{{.Id|html}}' style='visibility: hidden' src='loading.gif'/></td>

      <td class='etime' id='etime_{{.Id|html}}'>{{end}}{{.etime}}</td>

    <td class='ecats'>{{.ecats}}</td>

    {{range .cols}}
      <td class='ecol'>{{.|html}}</td>
    {{end}}
   {{with .heading}}
   </tr>
   {{end}}
`)

var EntryListEntryEditorHTML ExecutableTemplate = MakeExecutableTemplate("EntryListEntryEditor", `
    {{with .heading}}
    <tr id='editor_{{.|html}}' class='editor' style='display: none'>
    {{end}}
    {{with .entry}}
      <td colspan=4>
        <form id='ediv_{{.Id|html}}'>
          <input name='edtitle' type='text' disabled='disabled'/><br>
          <textarea name='edtext' disabled='disabled' rows=20>
          </textarea>

		  <input name='edid' type='hidden'/>
		  <input name='edprio' type='hidden'/>

		  <p>When: <input type='text' id='edat_{{.Id|html}}' name='edat' size=10 disabled='disabled'/>
          <script>calendar.set("edat_{{.Id|html}}")</script>
		  &nbsp; Sort by: <input type='text' name='edsort' size=10 disabled='disabled'/>
          &nbsp; ID: {{.Id|html}}
          &nbsp; Timestamp: <img id='loading_{{.Id|html}}' style='display: none' src='loading.gif'/> <span id='ts_{{.Id|html}}'> </span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{{.Id|html}}", event)'/>
          <input type='button' name='savebtn' value='save' onclick='javascript:save_editor_by_id("{{.Id|html}}", event)' disabled='disabled'/>
          <input type='button' value='reload' onclick='javascript:fill_editor("{{.Id|html}}")'/></p>
        </form>
      </td>
    {{end}}
    {{with .heading}}
    </tr>
    {{end}}
`)

var ListEnderHTML ExecutableTemplate = MakeExecutableTemplate("ListEnder", `
  </table>
</body></html>
`)

var ErrorLogHeaderHTML ExecutableTemplate = MakeExecutableTemplate("ErrorLogHeader", `
<!DOCTYPE html>
<html>
<head>
  <title>Pooch: {{.name|html}}</title>
  <link type='text/css' rel='stylesheet' href='{{.theme}}'>
  <link type='image/png' rel='icon' href='animals-dog.png'>
</head>
<body>
  <p><pre class='code'>{{.code|html}}</pre></p>
  <table width='100%' id='maintable' style='border-collapse: collapse;'>
`)

var ErrorLogEntryHTML ExecutableTemplate = MakeExecutableTemplate("ErrorLogEntry", `
  <tr class='{{.htmlClass}}'>
    <td class='etime'>{{.time}}</td>
    <td class='etitle'>{{.message|html}}</td>
  </tr>
`)

var ExplainEntryHeaderHTML ExecutableTemplate = MakeExecutableTemplate("ExplainEntryHeader", `
  <th>
    <tr class='entry'>
      <td>addr</td>
      <td>opcode</td>
      <td>p1</td>
      <td>p2</td>
      <td>p3</td>
      <td>p4</td>
      <td>p5</td>
      <td>comment</td>
    </tr>
  </th>
`)

var StatHeaderHTML ExecutableTemplate = MakeExecutableTemplate("StatHeader", `
  <table class='maintable statstable' id='maintable'>
  <th>
    <tr class='entry'>
      <td class='stat_tag'>Tag</td>
      <td class='stat_total'>Total</td>

      <td class='stat_now'>NOW</td>
      <td class='stat_later'>LATER</td>
      <td class='stat_done'>DONE</td>

      <td class='stat_timed'>TIMED</td>

      <td class='stat_notes'>NOTES</td>
      <td class='stat_sticky'>STICKY</td>
    </tr>
  </th>
`)

var StatEntryHTML ExecutableTemplate = MakeExecutableTemplate("StatEntry", `
  <tr class='{{.htmlClass}}'>
    {{with .entry}}
      {{if .Link}}
        <td class='stat_tag'><a href='/list?q={{.Link|url}}'>{{.Name|html}}</a></td>
      {{else}}
        <td class='stat_tag'>{{.Name|html}}</td>
      {{end}}

    <td class='stat_total'>{{.Total|html}}</td>

    <td class='stat_now'>{{.Now|html}}</td>
    <td class='stat_later'>{{.Later|html}}</td>
    <td class='stat_done'>{{.Done|html}}</td>

    <td class='stat_timed'>{{.Timed|html}}</td>

    <td class='stat_notes'>{{.Notes|html}}</td>
    <td class='stat_sticky'>{{.Sticky|html}}</td>
    {{end}}
  </tr>
`)

var ExplainEntryHTML ExecutableTemplate = MakeExecutableTemplate("ExplainEntry", `
  <tr class='{{.htmlClass}}'>
    {{with .explain}}
    <td>{{.Addr|html}}</td>
    <td>{{.Opcode|html}}</td>
    <td>{{.P1|html}}</td>
    <td>{{.P2|html}}</td>
    <td>{{.P3|html}}</td>
    <td>{{.P4|html}}</td>
    <td>{{.P5|html}}</td>
    <td>{{.Comment|html}}</td>
    {{end}}
  </tr>
`)

var ErrorLogEnderHTML ExecutableTemplate = MakeExecutableTemplate("ErrorLogEnder", `
  </table>
</body>
</html>
`)

var CalendarHeaderHTML ExecutableTemplate = MakeExecutableTemplate("CalendarHeader", `
<!DOCTYPE html>
<html>
<head>
  <link rel='stylesheet' type='text/css' href='dot-luv/jquery-ui.custom.css'>
  <link type='text/css' rel='stylesheet' href='listcommon.css'>
  <link rel='stylesheet' type='text/css' href='fullcalendar.css'/>
  <link rel='stylesheet' type='text/css' href='cal.css'/>
  <link type='image/png' rel='icon' href='animals-dog.png'>
  <link rel='stylesheet' type='text/css' href='{{.theme}}'/>
  <script src='jquery.js'></script>
  <script src='jquery-ui-custom.js'></script>
  <script src='fullcalendar.js'></script>
  <script src='int.js'></script>
  <script src='cint.js'></script>
  <title>{{.query|html}} calendar</title>
  <script>
     var query = "{{.query|html}}";
  </script>
</head>
<body onkeydown='keytable(event)' onkeypress='keypress(event)'>
`)

var CalendarHTML ExecutableTemplate = MakeExecutableTemplate("Calendar", `
  <p>
  <div id='calendar'></div>
  <script>
  </script>
</body>
</html>
`)

var RegisterHTML ExecutableTemplate = MakeExecutableTemplate("Register", `
<!DOCTYPE html>
<html>
  <head>
    <title>Register with pooch2</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    <div>{{.problem}}</div>
    <form method="post" action="/register">
      Username:&nbsp;<input type='text' name='user'/><br/>
      Password:&nbsp;<input type='password' name='password'/><br/>
      <input type='submit' value='register'/>
    </form>
  </body>
</html>
`)

var LoginHTML ExecutableTemplate = MakeExecutableTemplate("Login", `
<!DOCTYPE html>
<html>
  <head>
    <title>Login with pooch2</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    <div>{{.problem}}</div>
    <form method="post" action="/login">
      Username:&nbsp;<input type='text' name='user'/><br/>
      Password:&nbsp;<input type='password' name='password'/><br/>
      <input type='submit' value='login'/>
    </form>
  </body>
</html>
`)


var RegisterOKHTML ExecutableTemplate = MakeExecutableTemplate("RegisterOK", `
<!DOCTYPE html>
<html>
  <head>
    <title>Register with pooch2</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    Registration successful. <a href="/login">Login</a>.
  </body>
</html>
`)

var LoginOKHTML ExecutableTemplate = MakeExecutableTemplate("LoginOK", `
<!DOCTYPE html>
<html>
  <head>
    <title>Login with pooch2</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    Login successful, go to <a href="/list">index</a>.
  </body>
</html>
`)

var WhoAmIHTML ExecutableTemplate = MakeExecutableTemplate("WhoAmI", `
<!DOCTYPE html>
<html>
  <head>
    <title>Who Am I</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    You are: {{.username|html}}
  </body>
</html>
`)

var MustLogInHTML ExecutableTemplate = MakeExecutableTemplate("MustLogIn", `
<!DOCTYPE html>
<html>
  <head>
    <title>Login needed</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    You must log in first: <a href="/login">login</a> or <a href="/register">register</a>.
  </body>
</html>
`)

var OptionsPageHeader ExecutableTemplate = MakeExecutableTemplate("OptionsPageHeader", `
<!DOCTYPE html>
<html>
  <head>
    <title>Options</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
    <script src='/jquery.js'></script>
    <script src='/opts.js'></script>
  </head>
  <body>
    <h2>Options</h2>
    <form action="/opts" method="get">
      <input type='hidden' name='save' value='save'/>
`)

var OptionsPageLine ExecutableTemplate = MakeExecutableTemplate("OptionsPageLine", `
      <label for='{{.name|html}}'>{{.name|html}}</label>&nbsp;<input type='text' name='{{.name|html}}' id='{{.name|html}}' value='{{.value|html}}'/></br>
`)

var OptionsLongPageLine ExecutableTemplate = MakeExecutableTemplate("OptionsLongPage", `
      <label for='<name|html}'>{{.name|html}}</label>&nbsp;<textarea name='{{.name|html}}' id='{{.name|html}}' rows='20' cols='80'>{{.value|html}}</textarea></br>
`)

var OptionsPageAPITokens ExecutableTemplate = MakeExecutableTemplate("OptionsPageAPITokens", `
	<hr>
	Released API tokens:
	<ul>
		{{range .}}
		<li>{{.|html}} <input type='button' value='Revoke' onclick='tokremove("{{.|html}}")'></li>
		{{end}}
	</ul>
	<input type='button' value="Add a token" onclick='tokadd()'>
`)

var OptionsPageEnd ExecutableTemplate = MakeExecutableTemplate("OptionsPageEnd", `
      <input type='submit' value='save'/>
    </form>
  </body>
</html>
`)


