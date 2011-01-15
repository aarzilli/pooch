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

var CommonHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
  <script>
    thisPage = "{pageName}";
  </script>
  <div class='advmenu'>
    <a href='/opts'>options</a>
    &nbsp;
    <a href="/advanced.html">advanced operations</a><br/>
    Current timezone: {timezone|html}
  </div>
  <h2 style='display: inline;'>{queryForTitle|html}</h2>
  <div class='mainmenu'>
    <div class='mainmenu_item'>
      <a href='javascript:toggle_searchpop()'>[change query]</a>
      <div id='searchpop' class='popup' style='display: none'>
         <form id='searchform' method='get' action='{pageName}'>
           <label for='q'>Query:</label>&nbsp;
           <textarea name='q' id='q' cols='50' rows='10'>{query|html}</textarea>
           <input type='submit' value='search'>
           &nbsp;
           <div class='popbuttons'>
           {.section removeSearch }
             <input type='button' value='Edit query' onClick='javascript:editsearch()'/>
             <input type='button' value='Remove query' onClick='javascript:removesearch()'/>
           {.or}
             <input type='button' value='Save query' onClick='javascript:savesearch()'/>
           {.end}
           </div>
           <div class='keyinfo'>(press alt-enter to search)</div>
         </form>
      </div>
    </div>
    <div class='mainmenu_item'>
      <a href='javascript:toggle_addpop()'>[add entry]</a>
      <div id='addpop' class='popup' style='display: none'>
        <form onsubmit='return add_entry("{query|html}")'>
          <label for='newentry'>New entry:</label>&nbsp;
          <input size='50' type='text' id='newentry' name='text'/>
          <input type='button' value='add' onclick='javascript:add_entry("{query|html}")'/>
        </form>
      </div>
    </div>
    <div class='mainmenu_item'>
      <a href="{otherPageName}?q={query|url}">[see as {otherPageLink}]</a>
    </div>
    <div class='mainmenu_item'>
      <a href="/explain?q={query|url}">[see explanation]</a>
    </div>
    <div class='mainmenu_item'>
      <a href='javascript:toggle_navpop()'>[navigation]</a>
      <div id='navpop' class='popup' style='display: none'>
         <ul class='navlist'>
           <li><a href='{pageName}?q='>index</a></li>
           {.repeated section savedSearches}
           <li><a href="{pageName}?q=%23%25{@|url}">#%{@|html}</a></li>
           {.end}
         </ul>
         <hr/>
         <ul class='navlist'>
           {.repeated section subtags}
           <li><a href="{pageName}?q={@|url}">{@|html}</a></li>
           {.or}
           {.end}
         </ul>
         <hr/>
         <ul class='navlist'>
           {.repeated section toplevel}
           <li><a href="{pageName}?q={@|url}">{@|html}</a></li>
           {.or}
           {.end}
        </ul>
      </div>
    </div>
  </div>

  {.section parseError}
    <div class='screrror'>Error while executing search: {@|html} <a href='/errorlog'>Full error log</a></div>
  {.end}
  {.section retrieveError}
    <div class='screrror'>Error while executing search: {@|html} <a href='/errorlog'>Full error log</a></div>
  {.end}
`)

var EntryListHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
  <table class='maintable' id='maintable'>
`)

var ListHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
<!DOCTYPE html>
<html>
<head>
  <title>Pooch: {queryForTitle|html}</title>
  <link type='text/css' rel='stylesheet' href='listcommon.css'>
  <link type='text/css' rel='stylesheet' href='{theme}'>
  <link type='text/css' rel='stylesheet' href='calendar.css'>
  <script src='/json.js'></script>
  <script src='/jquery.js'></script>
  <script src='/int.js'></script>
  <script src='/calendar.js'></script>
</head>
<body onkeypress='keytable(event)'>
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
      <td class='etitle' onclick='javascript:toggle_editor("{id|html}", event)'><a href='javascript:toggle_editor("{id|html}", event)'>{title|html}</a></td>

      <td class='epr'>
        <input type='button' class='prioritybutton priorityclass_{priority|priority}' id='epr_{id|html}' value='{priority|priority}' onclick='javascript:change_priority("{id|html}", event)'/>
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
          <input name='edtitle' type='text' disabled='disabled'/><br>
          <textarea name='edtext' disabled='disabled' rows=20>
          </textarea>
          <textarea name='edcols' disabled='disabled' rows=20>
          </textarea>

		  <input name='edid' type='hidden'/>
		  <input name='edprio' type='hidden'/>

		  <p>When: <input type='text' id='edat_{id|html}' name='edat' size=10 disabled='disabled'/>
          <script>calendar.set("edat_{id|html}")</script>
		  &nbsp; Sort by: <input type='text' name='edsort' size=10 disabled='disabled'/>
          &nbsp; ID: {id|html}
          &nbsp; Timestamp: <img id='loading_{id|html}' style='display: none' src='loading.gif'/> <span id='ts_{id|html}'>—</span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{id|html}", event)'/>
          <input type='button' name='savebtn' value='save' onclick='javascript:save_editor_by_id("{id|html}", event)' disabled='disabled'/>
          <input type='button' value='reload' onclick='javascript:fill_editor("{id|html}")'/></p>
        </form>
      </td>
    {.end}
    {.section heading}
    </tr>
    {.end}
`)

var ListEnderHTML ExecutableTemplate = MakeExecutableTemplate(`
  </table>
</body></html>
`)

var ErrorLogHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
<!DOCTYPE html>
<html>
<head>
  <title>Pooch: {name|html}</title>
  <link type='text/css' rel='stylesheet' href='{theme}'>
</head>
<body>
  <p><pre class='code' style='font-family: monospace'>{code|html}</pre></p>
  <table width='100%' id='maintable' style='border-collapse: collapse;'>
`)

var ErrorLogEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
  <tr class='{htmlClass}'>
    <td class='etime'>{time}</td>
    <td class='etitle'>{message|html}</td>
  </tr>
`)

var ExplainEntryHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
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

var ExplainEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
  <tr class='{htmlClass}'>
    {.section explain}
    <td>{addr|html}</td>
    <td>{opcode|html}</td>
    <td>{p1|html}</td>
    <td>{p2|html}</td>
    <td>{p3|html}</td>
    <td>{p4|html}</td>
    <td>{p5|html}</td>
    <td>{comment|html}</td>
    {.end}
  </tr>
`)

var ErrorLogEnderHTML ExecutableTemplate = MakeExecutableTemplate(`
  </table>
</body>
</html>
`)

var CalendarHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
<!DOCTYPE html>
<html>
<head>
  <link type='text/css' rel='stylesheet' href='listcommon.css'>
  <link rel='stylesheet' type='text/css' href='fullcalendar.css'/>
  <link rel='stylesheet' type='text/css' href='cal.css'/>
  <script src='jquery.js'></script>
  <script src='jquery-ui-custom.js'></script>
  <script src='fullcalendar.js'></script>
  <script src='int.js'></script>
  <script src='cint.js'></script>
  <title>{query|html} calendar</title>
  <script>
     var query = "{query|html}";
  </script>
</head>
<body onkeypress='keytable(event)'>
`)

var CalendarHTML ExecutableTemplate = MakeExecutableTemplate(`
  <p>
  <div id='calendar'></div>
  <script>
  </script>
</body>
</html>
`)

var RegisterHTML ExecutableTemplate = MakeExecutableTemplate(`
<!DOCTYPE html>
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
<!DOCTYPE html>
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
<!DOCTYPE html>
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
<!DOCTYPE html>
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
<!DOCTYPE html>
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
<!DOCTYPE html>
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
<!DOCTYPE html>
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
      <label for='{name|html}'>{name|html}</label>&nbsp;<input type='text' name='{name|html}' id='{name|html}' value='{value|html}'/></br>
`)

var OptionsLongPageLine ExecutableTemplate = MakeExecutableTemplate(`
      <label for='<name|html}'>{name|html}</label>&nbsp;<textarea name='{name|html}' id='{name|html}' rows='20' cols='80'>{value|html}</textarea></br>
`)

var OptionsPageEnd ExecutableTemplate = MakeExecutableTemplate(`
      <input type='submit' value='save'/>
    </form>
  </body>
</html>
`)


