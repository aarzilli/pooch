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

func PriorityFormatter(w io.Writer, format string, value ...interface{}) {
	v := value[0].(Priority)
	io.WriteString(w, strings.ToUpper(v.String()))
}

func URLFormatter(w io.Writer, format string, value ...interface{}) {
	v := value[0].(string)
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
		err := t.Execute(wr, data)
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
             <input type='button' id='editquerybtn' value='Edit query' onClick='javascript:editsearch()'/>
             <input type='button' value='Remove query' onClick='javascript:removesearch()'/>
           {.or}
             <input type='button' value='Save query' onClick='javascript:savesearch()'/>
           {.end}
           </div>
           <div class='keyinfo'>(press alt-enter to search)</div>
         </form>
      </div>
    </div>
    {.section otherPageName}
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
      <a href="{@}?q={query|url}">[see as {otherPageLink}]</a>
    </div>
    <div class='mainmenu_item'>
      <a href="/explain?q={query|url}">[see explanation]</a>
    </div>
    {.end}
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
  <link type='image/png' rel='icon' href='animals-dog.png'>
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
      <td class='prchange' colspan=4>{Priority|priority}</td>
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
      <td class='etitle' onclick='javascript:toggle_editor("{Id|html}", event)'><a href='javascript:toggle_editor("{Id|html}", event)'>{Title|html}</a></td>

      <td class='epr'>
        <input type='button' class='prioritybutton priorityclass_{Priority|priority}' id='epr_{Id|html}' value='{Priority|priority}' onclick='javascript:change_priority("{Id|html}", event)'/>
      </td>
    {.end}

    <td class='etime'>{etime}</td>

    <td class='ecats'>{ecats}</td>

    {.repeated section cols}
      <td class='ecol'>{@|html}</td>
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
        <form id='ediv_{Id|html}'>
          <input name='edtitle' type='text' disabled='disabled'/><br>
          <textarea name='edtext' disabled='disabled' rows=20>
          </textarea>
          <textarea name='edcols' disabled='disabled' rows=20>
          </textarea>

		  <input name='edid' type='hidden'/>
		  <input name='edprio' type='hidden'/>

		  <p>When: <input type='text' id='edat_{Id|html}' name='edat' size=10 disabled='disabled'/>
          <script>calendar.set("edat_{Id|html}")</script>
		  &nbsp; Sort by: <input type='text' name='edsort' size=10 disabled='disabled'/>
          &nbsp; ID: {Id|html}
          &nbsp; Timestamp: <img id='loading_{Id|html}' style='display: none' src='loading.gif'/> <span id='ts_{Id|html}'>—</span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{Id|html}", event)'/>
          <input type='button' name='savebtn' value='save' onclick='javascript:save_editor_by_id("{Id|html}", event)' disabled='disabled'/>
          <input type='button' value='reload' onclick='javascript:fill_editor("{Id|html}")'/></p>
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
  <link type='image/png' rel='icon' href='animals-dog.png'>
</head>
<body>
  <p><pre class='code'>{code|html}</pre></p>
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

var StatHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
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

var StatEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
  <tr class='{htmlClass}'>
    {.section entry}
      {.section Link}
        <td class='stat_tag'><a href='/list?q={Link|url}'>{Name|html}</a></td>
      {.or}
        <td class='stat_tag'>{Name|html}</td>
      {.end}
      
    <td class='stat_total'>{Total|html}</td>

    <td class='stat_now'>{Now|html}</td>
    <td class='stat_later'>{Later|html}</td>
    <td class='stat_done'>{Done|html}</td>

    <td class='stat_timed'>{Timed|html}</td>

    <td class='stat_notes'>{Notes|html}</td>
    <td class='stat_sticky'>{Sticky|html}</td>
    {.end}
  </tr>
`)

var ExplainEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
  <tr class='{htmlClass}'>
    {.section explain}
    <td>{Addr|html}</td>
    <td>{Opcode|html}</td>
    <td>{P1|html}</td>
    <td>{P2|html}</td>
    <td>{P3|html}</td>
    <td>{P4|html}</td>
    <td>{P5|html}</td>
    <td>{Comment|html}</td>
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
  <link type='image/png' rel='icon' href='animals-dog.png'>
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
    <link type='image/png' rel='icon' href='animals-dog.png'>
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
    <link type='image/png' rel='icon' href='animals-dog.png'>
  </head>
  <body>
    <div>{problem}</div>
    <form method="post" action="/login">
      Username:&nbsp;<input type='text' name='user'/><br/>
      Password:&nbsp;<input type='password' name='password'/><br/>
      <input type='submit' value='login'/>
    </form>
  </body>
</html>
`)


var RegisterOKHTML ExecutableTemplate = MakeExecutableTemplate(`
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

var LoginOKHTML ExecutableTemplate = MakeExecutableTemplate(`
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

var WhoAmIHTML ExecutableTemplate = MakeExecutableTemplate(`
<!DOCTYPE html>
<html>
  <head>
    <title>Who Am I</title>
    <link type='image/png' rel='icon' href='animals-dog.png'>
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
    <link type='image/png' rel='icon' href='animals-dog.png'>
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
    <link type='image/png' rel='icon' href='animals-dog.png'>
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


