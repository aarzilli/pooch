/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"template"
	"fmt"
	"io"
	"strings"
	"http"
)

func PriorityFormatter(w io.Writer, value interface{}, format string) {
	v := value.(Priority)
	io.WriteString(w, strings.ToUpper(v.String()))
}

func URLFormatter(w io.Writer, value interface{}, format string) {
	v := value.(string)
	io.WriteString(w, http.URLEscape(v))
}

var formatters template.FormatterMap = template.FormatterMap{
	"html": template.HTMLFormatter,
	"priority": PriorityFormatter,
	"url": URLFormatter,
}

type ExecutableTemplate func(interface{}, io.Writer)

func WrapTemplate(t *template.Template) ExecutableTemplate {
	return func(data interface{}, wr io.Writer) {
		err := t.Execute(data, wr)
		if err != nil {
			panic(fmt.Sprintf("Error while formatting: %s\n", err))
		}
	}
}

func MakeExecutableTemplate(t string) ExecutableTemplate {
	return WrapTemplate(template.MustParse(t, formatters))
}

var ListHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
<head>
  <title>Pooch: {query|html}</title>
  <link type='text/css' rel='stylesheet' href='{theme}'>
  <link type='text/css' rel='stylesheet' href='calendar.css'>
`)

var JavascriptIncludeHTML ExecutableTemplate = MakeExecutableTemplate(`
  <script src='{fname|url}'>
  </script>
`)

func JavascriptInclude(c io.Writer, name string) {
	JavascriptIncludeHTML(map[string]string{"fname": name}, c)
}

var ListHeaderCloseHTML ExecutableTemplate = MakeExecutableTemplate(`
</head>
<body onload='javascript:setup()'>
`)

var EntryListHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
  <h2>{query|html} <span style='font-size: small'><a href='cal?q={query|url}'>as calendar</a></span></h2>
  <p><form onsubmit='return add_entry("{query|html}")'>
  <label for='text'>New entry:</label>&nbsp;<input size='50' type='newentry' id='newentry' name='text'/><input type='button' value='add' onclick='javascript:add_entry("{query|html}")'/>
  </form>

  <p><form method='get' action='/list'>
  <input type='hidden' id='theme' name='theme' value='{theme}'/>
  <label for='query'>Query:</label>&nbsp;<input size='50' type='text' id='q' name='q' value='{query|html}'/> <input type='submit' value='search'/> &nbsp; <input type='checkbox' name='done' value='1' {includeDone|html}> include done <input type='button' style='float: right' value='save query' onClick='javascript:savesearch()'/>
  </form>
`)

var EntryListPriorityChangeHTML ExecutableTemplate = MakeExecutableTemplate(`
    <tr class='prchange'>
      <td colspan=3>{priority|priority}</td>
    </tr>
`)

var EntryListEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
    {.section entry}
      <td class='etitle' onclick='javascript:toggle_editor("{id|html}", event)'>{title|html}</td>

      <td class='epr'>
        <input type='button' class='priorityclass_{priority|priority}' id='epr_{id|html}' value='{priority|priority}' onclick='javascript:change_priority("{id|html}", event)'/>
      </td>

      <td class='etime'>{etime}</td>

      <td class='ecats'>{ecats}</td>
   {.end}
`)

var EntryListEntryEditorHTML ExecutableTemplate = MakeExecutableTemplate(`
    {.section entry}
      <td colspan=4>
        <form id='ediv_{id|html}'>
          <p><input name='edtilte' id='edtitle' type='text' style='width: 99%; padding-bottom: 5px'/><br>
          <textarea style='width: 65%; margin-right: 1%' name='edtext' id='edtext' rows=20>
          </textarea>
          <textarea style='width: 33%;' float: right' name='edcols' id='edcols' rows=20>
          </textarea>
          </p>

		  <input name='edid' id='edid' type='hidden'/>
		  <input name='edprio' id='edprio' type='hidden'/>

		  <p>When: <input type='text' id='edat_{id|html}' name='edat' size=10/>
          <script>calendar.set("edat_{id|html}")</script>
		  &nbsp Repeat: <input type='text' id='edfreq' name='edfreq' size=10/>
		  &nbsp; Sort by: <input type='text' id='edsort' name='edsort' size=10/>
          &nbsp; ID: {id|html}
          &nbsp; Timestamp: <span id='ts_{id|html}'>-------</span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{id|html}", event)'/>
          <input type='button' value='save' onclick='javascript:save_editor_by_id("{id|html}", event)'/>
          <input type='button' value='reload', onclick='javascript:fill_editor("{id|html}")'/></p>
        </form>
      </td>
    {.end}
`)

var SubcolEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
<a href='list?theme={theme|url}&q={dst|url}'>{name|html}</a><br>
`)

var CalendarHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
<head>
  <link rel='stylesheet' type='text/css' href='fullcalendar.css'/>
  <link rel='stylesheet' type='text/css' href='cal.css'/>
  <script type='text/javascript' src='jquery.js'></script>
  <script type='text/javascript' src='jquery-ui-custom.js'></script>
  <script type='text/javascript' src='fullcalendar.js'></script>
  <script type='text/javascript' src='cint.js'></script>
  <title>{query|html} calendar</title>
  <script>
     var query = "{query|html}";
  </script>
</head>
<body>
`)

var CalendarHTML ExecutableTemplate = MakeExecutableTemplate(`
  <h2>{query|html} <span style='font-size: small'><a href='list?q={query|url}'>as list</a></span></h2>

  <p><form onsubmit='return add_entry("{query|html}")'>
  <label for='text'>New entry:</label>&nbsp;<input size='50' type='newentry' id='newentry' name='text'/><input type='button' value='add' onclick='javascript:add_entry("{query|html}")'/>
  </form>

  <p><form method='get' action='/cal'>
  <label for='query'>Query:</label>&nbsp;<input size='50' type='text' id='q' name='q' value='{query|html}'/><input type='submit' value='search'/>

  <p>
  <div id='calendar'></div>
  <script>
  </script>
</body>
</html>
`)

var RegisterHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Register with pooch2</title>
  </head>
  <body>
    <div>{problem}</div>
    <form method="post" action="/register">
      Username:&nbsp;<input type='text' name='user'/><br/>
      Password:&nbsp;<input type='password' name='password'/><br/>
      <input type='submit' value='register'/>
    </form>
  </body>
</html>
`)

var LoginHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Login with pooch2</title>
  </head>
  <body>
    <div>{problem}</div>
    <form method="post" action="/login">
      Username:&nbsp;<input type='text' name='user'/><br/>
      Password:&nbsp;<input type='password' name='password'/><br/>
      <input type='submit' value='register'/>
    </form>
  </body>
</html>
`)


var RegisterOKHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Register with pooch2</title>
  </head>
  <body>
    Registration successful. <a href="/login">Login</a>.
  </body>
</html>
`)

var LoginOKHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Login with pooch2</title>
  </head>
  <body>
    Login successful, go to <a href="/list">index</a>.
  </body>
</html>
`)

var WhoAmIHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Who Am I</title>
  </head>
  <body>
    You are: {username|html}
  </body>
</html>
`)

var MustLogInHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Login needed</title>
  </head>
  <body>
    You must log in first: <a href="/login">login</a> or <a href="/register">register</a>.
  </body>
</html>
`)