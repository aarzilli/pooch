/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

function toggle(query) {
    var div = $(query).get(0);
    if (div.style['display'] == 'none') {
        div.style['display'] = 'block';
        return true
    } else {
        div.style['display'] = 'none';
        return false
    }
}

function toggle_searchpop() {
    toggle("#searchpop");
    $("#q").get(0).focus();
    $("#q").get(0).select();
}

function toggle_addpop() {
    toggle("#addpop");
    $("#newentry").get(0).focus();
}

function toggle_navpop() {
    toggle("#navpop");
}

function toggle_runpop() {
    toggle("#runpop");
}

var cancelNextKeypress = false;

function keypress(e) {
	if (cancelNextKeypress) {
		cancelNextKeypress = false;
		e.preventDefault();
		return false;
	}
	return true;
}

function perform_toggle_on_keyevent(e) {
    switch(e.which) {
    case 65: // a key
        toggle_addpop();
        e.preventDefault();
        return false;
    case 83: // s key
        toggle_searchpop();
        e.preventDefault();
        return false;
    default:
        return true;
    }
}

function keytable(e) {
    switch (e.keyCode) {
    case 27:
        $("#searchpop").get(0).style['display'] = 'none';
        $("#addpop").get(0).style['display'] = 'none';
        cancelNextKeypress = true;
        e.preventDefault();
        return false;
    case 13:
        if (e.altKey) {
            if ($("#searchpop").get(0).style['display'] != 'none') {
                $("#searchform").get(0).submit();
            }
        }
        cancelNextKeypress = true;
        e.preventDefault();
        return false;
    }

    if (document.activeElement.type == null) {
        var r = perform_toggle_on_keyevent(e);
        cancelNextKeypress = !r;
        return r;
    }
    nothing_enabled = ($("#searchpop").get(0).style['display'] == 'none') && ($("#addpop").get(0).style['display'] == 'none');

    if (!nothing_enabled) {
        return true;
    }

    if ((document.activeElement.id == "q") || (document.activeElement.id == "newentry")) {
        var r = perform_toggle_on_keyevent(e);
        cancelNextKeypress = !r;
        return r;
    }
}

function remove_entry(name) {
  $.ajax({ url: "remove?id=" + encodeURIComponent(name), success: function(data, textStatus, req) {
	if (data.match(/^removed/)) {
	  var maintable = $("#maintable").get(0);
	  for (var i in maintable.rows) {
	    if (maintable.rows[i] == null) continue; // deleted rows
	    if (maintable.rows[i].id == null) continue; // rows without id

	    if (maintable.rows[i].id == "editor_" + name) {
	      maintable.deleteRow(i);
	      maintable.deleteRow(i-1); // this is the title
	      break;
	    }
	  }
	} else {
	  $("#ts_"+v.id).html("Remove failed: " + data);
	}
    }});
}

function save_editor(form) {
    if (form == null) return;
    if (form.elements['edtitle'].disabled != "") return;

    obj = new Object();
    obj.title = form.elements['edtitle'].value;
    obj.text = form.elements['edtext'].value;
    obj.triggerAt = form.elements['edat'].value;
    obj.sort = form.elements['edsort'].value;
    obj.id = form.elements['edid'].value;
    obj.priority = parseInt(form.elements['edprio'].value);
    $("#loading_"+obj.id).get(0).style.display = "inline";

    $.ajax({ type: "POST", url: "/save", data: JSON.stringify(obj), success: function(data, textStatus, req) {
	  if (data.match(/^saved-at-timestamp: /)) {
	    $("#ts_"+obj.id).html(data.substr("saved-at-timestamp: ".length));
	  } else {
	    $("#ts_"+obj.id).html(" LAST SAVE FAILED: " + data);
	  }
	  $("#loading_"+obj.id).get(0).style.display = "none";
	}});
}

function add_row(id) {
  $.ajax({ url: "htmlget?type=add&id=" + encodeURIComponent(id), success: function(data, textStatus, req) {
	var newrows = data.split("\u2029", 2);

	var maintable = $("#maintable").get(0);

	var newrow1 = maintable.insertRow(0);
	newrow1.setAttribute("class", "entry");
	newrow1.innerHTML = newrows[0];

	var newrow2 = maintable.insertRow(1);
	newrow2.setAttribute("id", "editor_"+encodeURIComponent(id));
	newrow2.setAttribute("class", "editor");
	newrow2.setAttribute("style", "display: none");
	newrow2.innerHTML = newrows[1];

	save_open_editor(true);
      }});
}

function add_entry(query) {
    var netext = $('#newentry').val();
    $.ajax({ url: "qadd?q=" + encodeURIComponent(query) + "&text=" + encodeURIComponent(netext), success: function(data, textStatus, req) {
                if (data.match(/^added: /)) {
                    newid = data.substr("added: ".length);
                    add_row(newid);

                    $('#newentry').val("");
                    $("#addpop").get(0).style["display"] = "none";
                } else {
                    alert("ADD FAILED: " + data);
                }
            }});
    return false;
}

function change_editor_disabled(ed, disabledStatus) {
    if (ed == null) return;
    ed.elements['edtitle'].disabled = disabledStatus;
    ed.elements['edtext'].disabled = disabledStatus;
    ed.elements['edat'].disabled = disabledStatus;
    ed.elements['edsort'].disabled = disabledStatus;
    ed.elements['edid'].disabled = disabledStatus;
    ed.elements['edprio'].disabled = disabledStatus;
    ed.elements['savebtn'].disabled = disabledStatus;
}

function fill_editor(name) {
  $("#loading_"+name).get(0).style.display = "inline";
  $.ajax({url: "get?id=" + encodeURIComponent(name), success: function(data, textStatus, req) {
	var timestamp = data.split("\n", 2)[0];
	var jsonObj = data.substr(timestamp.length);
	v = $.parseJSON(jsonObj);
	var ed = $("#ediv_" + name).first().get(0);
	ed.elements['edtitle'].value = v.Title;
	ed.elements['edtext'].value = v.Text;
	ed.elements['edat'].value = v.TriggerAt;
	ed.elements['edsort'].value = v.Sort;
	ed.elements['edid'].value = v.Id;
	ed.elements['edprio'].value = v.Priority;

	$("#ts_" + v.Id).html(timestamp);
	change_editor_disabled(ed, "");
	$("#loading_"+name).get(0).style.display = "none";
      }});
}

function editor_from_row(row) {
  return $("#ediv_"+row.id.substr("editor_".length)).get(0);
}

function save_open_editor(should_close_editor) {
    orows = document.getElementsByTagName("tr");
    for (var i in document.getElementsByTagName("tr")) {
        orow = orows[i];

        if (orow == null) continue; // deleted rows
        if (orow.style == null) continue;
        if (orow.id == null) continue;

        if ((orow.id.match(/^editor_/)) && (orow.style['display'] != 'none')) {
            if (should_close_editor) {
                close_editor(orow)
            } else {
                save_editor(editor_from_row(orow));
            }
        }
    }
}

function save_editor_by_id(name, event) {
    save_editor($("#ediv_"+name).get(0));
}

function close_editor(row) {
    var ed = editor_from_row(row);
    save_editor(ed);
    change_editor_disabled(ed, "disabled");
    row.style['display'] = 'none';
}

function toggle_editor(name, event) {
    var row = $("#editor_"+name).get(0);
    if (row.style['display'] == 'none') {
        orows = document.getElementsByTagName("tr");
        for (var i in document.getElementsByTagName("tr")) {
            orow = orows[i];

            if (orow == null) continue;
            if (orow.style == null) continue;
            if (orow.id == null) continue;

            if ((orow.id.match(/^editor_/)) && (orow.style['display'] != 'none')) {
                close_editor(orow)
            }
        }

        row.style['display'] = '';

        fill_editor(name);
    } else {
        close_editor(row)
    }
}

function change_priority_to(name, priorityNum, priority) {
    var epr = $('#epr_'+name);
    epr.val(priority);
    epr.attr("class", "prioritybutton priorityclass_" + priority);

    // changes the value saved inside the editor div so that saving the editor contents doesn't revert a changed priority
    var ed = $("#ediv_"+name).get(0);
    ed.elements["edprio"].value = priorityNum;
}

function guess_next_priority(name, special) {
    var current = $('#epr_'+name).val();
    var etime_initial_char = $('#etime_'+name).get(0).innerHTML[0];

    if (etime_initial_char == "@") {
        if (current == "NOW") {
            change_priority_to(name, 5, "DONE")
        } else if (current == "TIMED") {
            change_priority_to(name, 5, "DONE");
        } else {
            change_priority_to(name, 6, "UNKW");
        }
    } else if (current == "NOTES") {
        if (special) {
            change_priority_to(name, 1, "NOW");
        } else {
            change_priority_to(name, 0, "STICKY");
        }
    } else if (current == "STICKY") {
        if (special) {
            change_priority_to(name, 1, "NOW");
        } else {
            change_priority_to(name, 3, "NOTES");
        }
    } else {
        if (special) {
            change_priority_to(name, 3, "NOTES");
        } else {
            if (current == "TIMED") {
                change_priority_to(name, 2, "LATER");
            } else if (current == "DONE") {
                change_priority_to(name, 2, "LATER");
            } else if (current == "LATER") {
                change_priority_to(name, 1, "NOW");
            } else if (current == "NOW") {
                change_priority_to(name, 5, "DONE");
            }
        }
    }

    return "";
}

function change_priority(name, event) {
    $("#ploading_"+name).get(0).style['visibility'] = 'visible';

    guess_next_priority(name, event.shiftKey);

    $.ajax({ url: "change-priority?id=" + encodeURIComponent(name) + "&special=" + event.shiftKey, success:
            function(data, textStatus, req) {
                if (data.match(/^priority-change-to: /)) {
                    priority = data.substr("priority-change-to: ".length);
                    priorityNum = priority[0];
                    priority = priority.substr(2);
                    change_priority_to(name, priorityNum, priority);
                    $("#ploading_"+name).get(0).style['visibility'] = 'hidden';
                } else {
                    alert(data);
                }
            }});
}

function savesearch_ex(name, query) {
  $.ajax({ url: "save-search?name=" + encodeURIComponent(name) + "&query=" + encodeURIComponent(query), success: function(data, textStatus, req) {
	if (!data.match(/^query-saved: /)) {
	  alert(data);
        }
      }});
}

function savesearch() {
    var name = prompt("save search to:");
    savesearch_ex(name, $('#q').val())
}

var editingSavedQueryMode = null;

function editsearch() {
    if (editingSavedQueryMode == null) {
        name = $('#q').val();
        $.ajax({ url: "save-search?name=" + encodeURIComponent(name),
                    success: function(data, textStatus, req) {
                    if (data.match(/^query-saved: /)) {
                        query = data.substring(13);
                        $("#editquerybtn").val("Save query");
                        editingSavedQueryMode = name;
                        $('#q').val(query);
                    }
                }});
    } else {
        savesearch_ex(editingSavedQueryMode, $("#q").val());
    }
}

function removesearch() {
  $.ajax({ url: "remove-search?query=" + encodeURIComponent($('#q').val()) });
}
