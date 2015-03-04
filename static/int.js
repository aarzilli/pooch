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

function toggle_addpop_sub(id) {
    toggle("#addpop");
    var ne = $("#newentry");
    ne.val("#sub/" + id + " ");
    ne.get(0).setSelectionRange(6 + id.length, 6 + id.length);
    ne.get(0).focus();
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
                cancelNextKeypress = true;
                e.preventDefault();
                return false;
            }
        }
        return true;
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
  var tbl = $("#editor_"+name).parent().parent().first().get(0);
  $.ajax({ url: "remove?id=" + encodeURIComponent(name), success: function(data, textStatus, req) {
	if (data.match(/^removed/)) {
	  //var maintable = $("#maintable").get(0);
	  for (var i in tbl.rows) {
	    if (tbl.rows[i] == null) continue; // deleted rows
	    if (tbl.rows[i].id == null) continue; // rows without id

	    if (tbl.rows[i].id == "editor_" + name) {
	      tbl.deleteRow(i);
	      tbl.deleteRow(i-1); // this is the title
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

function add_row(id, parent) {
  $.ajax({ url: "htmlget?type=add&id=" + encodeURIComponent(id), success: function(data, textStatus, req) {
    var newrows = data.split("\u2029", 2);

    var tbl = (parent == undefined) ?
       $("#maintable").get(0) :
       $("#subs_" + parent).get(0);

    var newrow1 = tbl.insertRow(0);
    newrow1.setAttribute("class", "entry");
    newrow1.innerHTML = newrows[0];

    var newrow2 = tbl.insertRow(1);
    newrow2.setAttribute("id", "editor_"+encodeURIComponent(id));
    newrow2.setAttribute("class", "editor");
    newrow2.setAttribute("style", "display: none");
    newrow2.innerHTML = newrows[1];

    if (parent == undefined) {
      save_open_editor(true);
    }
  }});
}

function add_entry(query) {
    var netext = $('#newentry').val();
    $.ajax({ url: "qadd?q=" + encodeURIComponent(query) + "&text=" + encodeURIComponent(netext), success: function(data, textStatus, req) {
                if (data.match(/^added: /)) {
                    resp = data.substr("added: ".length).split(/ /);

                    add_row(resp[0], resp[1]);

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

function fill_editor(name, contfn) {
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
        $.ajax({url: "list?guts=1&q=" + encodeURIComponent("#:sub #:w/done #:ssort #sub/" + v.Id), success: function(data, textStatus, req) {
            var tbl = $("#subs_" + v.Id).first().get(0);
            tbl.innerHTML = data
            if (data != "") {
                show_subs(v.Id);
                if (contfn != null) {
                    contfn();
                }
            } else {
                show_editor(v.Id);
            }
        }});

        if (contfn == null) {
            window.location.hash = window.location.hash + "#" + encodeURIComponent(v.Id);
        }
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
    var removeid = ed.elements['edid'].value;
    var vs = window.location.hash.split("#");
    var vr = [];
    for (var i = 0; i < vs.length; ++i) {
        if (vs[i] == "") continue;
        if (vs[i] == removeid) continue;
        vr.push(vs[i]);
    }
    window.location.hash = "#" + vr.join("#");
}

function toggle_editor(name, event, contfn) {
    var row = $("#editor_"+name).get(0);
    if (row.style['display'] == 'none') {
        var tbl = $("#editor_"+name).parent().parent().first().get(0);
        orows = tbl.getElementsByTagName("tr");
        for (var i in tbl.getElementsByTagName("tr")) {
            orow = orows[i];

            if (orow == null) continue;
            if (orow.style == null) continue;
            if (orow.id == null) continue;

            if ((orow.id.match(/^editor_/)) && (orow.style['display'] != 'none')) {
                close_editor(orow)
            }
        }

        row.style['display'] = '';

        fill_editor(name, contfn);
    } else {
        close_editor(row);
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

function save_ontology() {
  var o = JSON.stringify($("#ontonav").jstree("get_json"));
  $("#ontosaving").get(0).style.visibility = "visible";
  $.ajax({ type: "POST", url: "/ontologysave?save=1", data: o, success: function(data, textStatus, req) {
    if (data != "ok") {
      alert(data);
    }
    $("#ontosaving").get(0).style.visibility = "hidden";
  }});
}

function click_ontology(event) {
  var path = "";
  var c = $(event.target).parents().get(0);

  for (;;) {
    var n = $(c).children("a").get(0).text.replace(/\s+/g, "");
    path = n + " " + path;

    c = $(c).parents().get(0);
    c = $(c).parents().get(0);

    if (c.tagName != "LI") break;
  }

  window.location = "list?q=" + encodeURIComponent(path);
}

function show_editor(id) {
  $("#subs_" + id).first().get(0).style["display"] = "none";
  $("#ediv_" + id).first().get(0).style["display"] = "block";
}

function show_subs(id) {
  $("#subs_" + id).first().get(0).style["display"] = "block";
  $("#ediv_" + id).first().get(0).style["display"] = "none";
}

function onload_open_function(v, start) {
  return function() {
    for (var i = start; i < v.length; ++i) {
      if (v[i] == "") continue;
      toggle_editor(v[i], null, onload_open_function(v, i+1));
      return;
    }
  }
}

function ontonav_click_folder(ev) {
	var thea = ev.target;

	if (ev.target.innerHTML == '\uf096') {
		return;
	}

	var childul = $(ev.target).parents('li').children('ul').get(0);

	if (ev.target.innerHTML == '\uf147') {
		childul.style['display'] = 'none';
		ev.target.innerHTML = '\uf196';
	} else {
		childul.style['display'] = 'block';
		ev.target.innerHTML = '\uf147'
	}
}

function ontonav_drop(ev) {
	console.debug(ev);

	var dsttype = ($(ev.target).parents('span').get(0).classList.contains('tree_folder')) ? 'chlidren' : 'sibling';
	var dst = $(ev.target).parents('li').children('.tree_content').children('a').get(0).innerHTML;
	var src = ev.dataTransfer.getData('text/text');

	console.debug(dsttype, dst, src);

	$.ajax({ type: "GET", url: "/ontology?move=1&src=" + encodeURIComponent(src) + "&dst=" + encodeURIComponent(dst) + "&mty=" + dsttype, success: function(data, textStatus, req) {
		load_ontonav();
	}})
}

function cancel_shit(ev) {
	ev.preventDefault();
	return false;
}

function ontonav_start_drag(ev, source) {
	ev.dataTransfer.setData("text/text", source);
}

function load_ontonav_rec(onl, t) {
	for (var i = 0; i < t.length; i++) {
		var li = document.createElement("li");
		onl.appendChild(li);

		function add_d(ftype, content) {
			var d = document.createElement("span");
			d.classList.add("tree_folder");
			d.innerHTML = "<a href='javascript:void(0)' onclick='ontonav_click_folder(event)' draggable='true' ondrop='ontonav_drop(event)' ondragenter='cancel_shit(event)' ondragover='cancel_shit(event)' ondragleave='cancel_shit(event)' ondragstart='ontonav_start_drag(event, \"" + content + "\")'>" + ftype + "</a>";
			li.appendChild(d);

			d = document.createElement("span");
			d.classList.add("tree_content");

			d.innerHTML = "<a href='list?q=" + encodeURIComponent(content) + "' ondrop='ontonav_drop(event)' ondragenter='cancel_shit(event)' ondragover='cancel_shit(event)' ondragleave='cancel_shit(event)' ondragstart='ontonav_start_drag(event, \"" + content + "\")'>" + content + "</a>";
			li.appendChild(d);
		}


		if (typeof(t[i]) == "string") {
			add_d('\uf096', t[i]);
		} else {
			add_d('\uf147', t[i]['data'])

			d = document.createElement('ul');
			li.appendChild(d);
			load_ontonav_rec(d, t[i]['children']);
		}
	}
}

function load_ontonav() {
	$.ajax({ url: "ontology", success: function(data, textStatus, req) {
  		var t = JSON.parse(data);
  		var on = $("#ontonav").first().get(0);
  		on.innerHTML = "";
  		var onl = document.createElement("ul");
  		on.appendChild(onl);
  		load_ontonav_rec(onl, t);
	}})
}

window.onload = function() {
  load_ontonav();
  var v = window.location.hash.split("#")
  var f = onload_open_function(v, 0);
  f();
};
