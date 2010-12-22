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
  <title>Pooch: {queryForTitle|html}</title>
  <link type='text/css' rel='stylesheet' href='{theme}'>
  <link type='text/css' rel='stylesheet' href='calendar.css'>
  <script src='/json.js'></script>
  <script src='/jquery.js'></script>
  <script src='/int.js'></script>
  <script src='/calendar.js'></script>
</head>
<body onkeypress='keytable(event)'>
  <div style='float: right'><p align='right'><small><a href='/opts'>options</a>&nbsp;<a href="/advanced.html">advanced operations</a></small><br/><small>Current timezone: {timezone|html}</small></div>
  <h2>{queryForTitle|html} <span style='font-size: small'>
    &nbsp;
    <span style='position: relative;'>
      <a href='javascript:toggle_searchpop()'>[change query]</a>
      <div id='searchpop' class='popup' style='display: none; position: absolute; top: 20px; z-index: 9000'>
         <form method='get' action='/list'>
           <label for='query'>Query:</label>&nbsp;
           <textarea name='q' id='q' cols='50' rows='10'>{query|html}</textarea>
           <input type='submit' value='search'>
           &nbsp;
           <input type='checkbox' name='done' value='1' {includeDone|html}> include done
           {.section removeSearch }
             <input type='button' style='float: right' value='edit query' onClick='javascript:editsearch()'/>
             <input type='button' style='float: right' value='remove query' onClick='javascript:removesearch()'/>
           {.or}
             <input type='button' style='float: right' value='save query' onClick='javascript:savesearch()'/>
           {.end}
         </form>
      </div>
    </span>
    &nbsp;
    <span style='position: relative;'>
      <a href='javascript:toggle_addpop()'>[add entry]</a>
      <div id='addpop' class='popup' style='display: none; position: absolute; top: 20px; z-index: 9000'>
        <form onsubmit='return add_entry("{query|html}")'>
          <label for='text'>New entry:</label>&nbsp;
          <input size='50' type='newentry' id='newentry' name='text'/>
          <input type='button' value='add' onclick='javascript:add_entry("{query|html}")'/>
        </form>
      </div>
    </span>
    &nbsp;
    <a href='cal?q={query|url}'>[see as calendar]</a>
  </span></h2>

  {.section error}
    <div class='screrror'>Error while executing search: {@|html} <a href='/errorlog'>Full error log</a></div>
  {.end}

  <table width='100%' style='border-collapse: collapse;'><tr>
  <td valign='top' style='width: 10%'><div style='padding-top: 30px'>
`)

var SubcolsEnder ExecutableTemplate = MakeExecutableTemplate(`
  </div></td>
  <td valign='top'><table width='100%' id='maintable' style='border-collapse: collapse;'>
`)

var EntryListPriorityChangeHTML ExecutableTemplate = MakeExecutableTemplate(`
    <tr>
      {.section entry}
      <td class='prchange' colspan=4>{priority|priority}</td>
      {.end}
      {.repeated section colNames}
      <td class='colname'>{@|html}</td>
      {.end}
    </tr>
`)


var EntryListEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
    {.section heading}
    <tr class='{htmlClass}'>
    {.end}
    {.section entry}
      <td class='etitle' onclick='javascript:toggle_editor("{id|html}", event)'><a href="javascript:toggle_editor("{id|html}", event)'>{title|html}</a></td>

      <td class='epr'>
        <input type='button' class='priorityclass_{priority|priority}' id='epr_{id|html}' value='{priority|priority}' onclick='javascript:change_priority("{id|html}", event)'/>
      </td>

      <td class='etime'>{etime}</td>

      <td class='ecats'>{ecats}</td>

      {.repeated section cols}
      <td class='ecol'>{@|html}</td>
      {.end}
   {.end}
   {.section heading}
   </tr>
   {.end}
`)

var EntryListEntryEditorHTML ExecutableTemplate = MakeExecutableTemplate(`
    {.section heading}
    <tr id='editor_{@|html}' class='editor' style='display: none'>
    {.end}
    {.section entry}
      <td colspan=4>
        <form id='ediv_{id|html}'>
          <p><input name='edtilte' id='edtitle' type='text' style='width: 99%; padding-bottom: 5px' disabled='yes'/><br>
          <textarea style='width: 65%; margin-right: 1%' name='edtext' id='edtext' disabled='yes' rows=20>
          </textarea>
          <textarea style='width: 33%;' float: right' name='edcols' id='edcols' disabled='yes' rows=20>
          </textarea>
          </p>

		  <input name='edid' id='edid' type='hidden'/>
		  <input name='edprio' id='edprio' type='hidden'/>

		  <p>When: <input type='text' id='edat_{id|html}' name='edat' size=10 disabled='yes'/>
          <script>calendar.set("edat_{id|html}")</script>
		  &nbsp; Sort by: <input type='text' id='edsort' name='edsort' size=10 disabled='yes'/>
          &nbsp; ID: {id|html}
          &nbsp; Timestamp: <img id='loading_{id|html}' style='display: none' src='loading.gif'/> <span id='ts_{id|html}'>-------</span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{id|html}", event)'/>
          <input type='button' id='savebtn' name='savebtn' value='save' onclick='javascript:save_editor_by_id("{id|html}", event)' disabled='yes'/>
          <input type='button' value='reload', onclick='javascript:fill_editor("{id|html}")'/></p>
        </form>
      </td>
    {.end}
    {.section heading}
    </tr>
    {.end}
`)

var ListEnderHTML ExecutableTemplate = MakeExecutableTemplate(`
</table></td>
</tr></table></body></html>
`)

var ErrorLogHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
<html>
<head>
  <title>Pooch: error log</title>
  <link type='text/css' rel='stylesheet' href='{theme}'>
</head>
<body>
  <table width='100%' id='maintable' style='border-collapse: collapse;'>
`)

var ErrorLogEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
  <tr class='{htmlClass}'>
    <td class='etime'>{time}</td>
    <td class='etitle'>{message|html}</td>
  </tr>
`)

var ErrorLogEnderHTML ExecutableTemplate = MakeExecutableTemplate(`
  </table>
</body>
</html>
`)

var SubcolEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
<a href='list?q={dst|url}'>{name|html}</a><br>
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

var OptionsPageHeader ExecutableTemplate = MakeExecutableTemplate(`
<html>
  <head>
    <title>Options</title>
  </head>
  <body>
    <h2>Options</h2>
    <form action="/opts" method="get">
      <input type='hidden' name='save' value='save'/>
`)

var OptionsPageLine ExecutableTemplate = MakeExecutableTemplate(`
      <label for='{name|html}'>{name|html}</label><input type='text' name='{name|html}' id='{name|html}' value='{value|html}'/></br>
`)

var OptionsPageEnd ExecutableTemplate = MakeExecutableTemplate(`
      <input type='submit' value='save'/>
    </form>
  </body>
</html>
`)


