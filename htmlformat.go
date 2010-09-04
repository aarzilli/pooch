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
  <title>Pooch: {tlname|html}</title>
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
<body onload='javascript:setup("{tlname|html}")'>
`)

var TasklistLinkCellHTML ExecutableTemplate = MakeExecutableTemplate(`
    <td style='width: 1%;' class='{class|html}'>
      {.section tlink} <a href='/{baselink}?tl={tlink|html}&theme={theme}'> {.end}
      {tlname|html}
      {.section tlink} </a> {.or} &nbsp;<small><a href='/{baselink}?theme={theme}&tl={donelink|html}{donelinkextra|html}'>{donelinkname}</a></small>
      {.end}
    </td>
`)

var EntryListHeaderHTML ExecutableTemplate = MakeExecutableTemplate(`
  <h2>{tlname|html} <span style='font-size: small'><a href='cal?tl={tlname|html}'>calendar</a></span></h2>
  <p><form onsubmit='return add_entry("{tlname|html}")'>
  <label for='text'>New entry:</label>&nbsp;<input size='50' type='newentry' id='newentry' name='text'/><input type='button' value='add' onclick='javascript:add_entry("{tlname|html}")'/>
  </form>

  <p><form method='get' action='/list'>
  <input type='hidden' id='tl' name='tl' value='{tlname|html}'/>
  <input type='hidden' id='theme' name='theme' value='{theme}'/>
  <label for='query'>Query:</label>&nbsp;<input size='50' type='text' id='q' name='q' value=''/><input type='submit' value='search'/>
  </form>
`)

var EntryListHeaderForSearchHTML ExecutableTemplate = MakeExecutableTemplate(`
  <h1>{tlname|html}</h1>

  <p><form method='get' action='/list'>
  <input type='hidden' id='tl' name='tl' value='{tlname|html}'/>
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
      <td class='etitle' onclick='javascript:toggle_editor("{tlname|html}", "{id|html}", event)'>{title|html}</td>

      <td class='epr' onclick='javascript:change_priority("{tlname|html}", "{id|html}", event)'>
        <div class='priorityclass_{priority|priority}' id='epr_{id|html}'>{priority|priority}</div>
      </td>

      <td class='etime'>{etime}</td>
   {.end}
`)

var EntryListEntryEditorHTML ExecutableTemplate = MakeExecutableTemplate(`
    {.section entry}
      <td colspan=4>
        <form id='ediv_{id|html}'>
          <p><input name='edtilte' id='edtitle' type='text' size='80'/><br>
          <textarea name='edtext' id='edtext' rows=20 cols=70>
          </textarea></p>

		  <input name='edid' id='edid' type='hidden'/>
		  <input name='edprio' id='edprio' type='hidden'/>

		  <p>Trigger at: <input type='text' id='edat_{id|html}' name='edat' size=10/>
          <script>calendar.set("edat_{id|html}")</script>
		  &nbsp Repeat: <input type='text' id='edfreq' name='edfreq' size=10/>
		  &nbsp; Sort by: <input type='text' id='edsort' name='edsort' size=10/>
          &nbsp; Timestamp: <span id='ts_{id|html}'>-------</span></p>

          <p><input type='button' style='float: right' value='remove' onclick='javascript:remove_entry("{tlname|html}", "{id|html}", event)'/>
          <input type='button' value='save' onclick='javascript:save_editor_by_id("{tlname|html}", "{id|html}", event)'/>
          <input type='button' value='reload', onclick='javascript:fill_editor("{tlname|html}", "{id|html}")'/></p>
        </form>
      </td>
    {.end}
`)

var EntryListFooterHTML ExecutableTemplate = MakeExecutableTemplate(`
  </table>
</body>
</html>
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
  <title>{tlname|html} calendar</title>
  <script>
     var tlname = "{tlname|html}";
  </script>
</head>
<body>
`)

var CalendarHTML ExecutableTemplate = MakeExecutableTemplate(`
  <h2>{tlname|html} <span style='font-size: small'><a href='list?tl={tlname|html}'>list</a></span></h2>

  <p><form onsubmit='return add_entry("{tlname|html}")'>
  <label for='text'>New entry:</label>&nbsp;<input size='50' type='newentry' id='newentry' name='text'/><input type='button' value='add' onclick='javascript:add_entry("{tlname|html}")'/>
  </form>

  <div id='calendar'></div>
  <script>
  </script>
</body>
</html>
`)

