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
)

func PriorityFormatter(w io.Writer, value interface{}, format string) {
	v := value.(Priority)
	io.WriteString(w, strings.ToUpper(v.String()))
}

var formatters template.FormatterMap = template.FormatterMap{
	"html": template.HTMLFormatter,
	"priority": PriorityFormatter,
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
  <script src='{fname|html}'>
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
  <h2>{query|html}</h2>
  <p><form onsubmit='return add_entry("{query|html}")'>
  <label for='text'>New entry:</label>&nbsp;<input size='50' type='newentry' id='newentry' name='text'/><input type='button' value='add' onclick='javascript:add_entry("{query|html}")'/>
  </form>

  <p><form method='get' action='/list'>
  <input type='hidden' id='theme' name='theme' value='{theme}'/>
  <label for='query'>Query:</label>&nbsp;<input size='50' type='text' id='q' name='q' value='{query|html}'/><input type='submit' value='search'/>
  </form>
`)

var EntryListPriorityChangeHTML ExecutableTemplate = MakeExecutableTemplate(`
    <tr class='prchange'>
      <td colspan=4>{priority|priority}</td>
    </tr>
`)

var EntryListEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
    {.section entry}
      <td class='eid'>{id|html}</td>
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
          &nbsp; Timestamp: <span id='ts_{id|html}'>-------</span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{id|html}", event)'/>
          <input type='button' value='save' onclick='javascript:save_editor_by_id("{id|html}", event)'/>
          <input type='button' value='reload', onclick='javascript:fill_editor("{id|html}")'/></p>
        </form>
      </td>
    {.end}
`)

var SubcolEntryHTML ExecutableTemplate = MakeExecutableTemplate(`
<a href='list?theme={theme|html}&q={dst|html}'>{name|html}</a><br>
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
  <h2>{query|html} <span style='font-size: small'><a href='list?q={query|html}'>list</a></span></h2>

  <p><form onsubmit='return add_entry("{query|html}")'>
  <label for='text'>New entry:</label>&nbsp;<input size='50' type='newentry' id='newentry' name='text'/><input type='button' value='add' onclick='javascript:add_entry("{query|html}")'/>
  </form>

  <div id='calendar'></div>
  <script>
  </script>
</body>
</html>
`)

