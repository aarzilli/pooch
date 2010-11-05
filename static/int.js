/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

function setup() {
    shortcut.add("Alt+s", function() { save_open_editor(false); });
}

function remove_entry(name) {
    var req = new XMLHttpRequest();
    req.open("GET", "remove?id=" + encodeURIComponent(name), true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            if (req.responseText.match(/^removed/)) {
                var maintable = document.getElementById("maintable");
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
                var ts = document.getElementById("ts_" + v.id);
                ts.innerHTML = "Remove failed: " + req.responseText;
            }
        }
    };
    req.send(null);
}

function save_editor(form) {
    obj = new Object();
    obj.title = form.elements['edtitle'].value;
    obj.text = form.elements['edtext'].value;
    obj.triggerAt = form.elements['edat'].value;
    obj.sort = form.elements['edsort'].value;
    obj.id = form.elements['edid'].value;
    obj.priority = parseInt(form.elements['edprio'].value);
    obj.freq = form.elements['edfreq'].value;
    obj.cols = form.elements['edcols'].value;
    
    var ts = document.getElementById("ts_"+obj.id);
    var req = new XMLHttpRequest();
    req.open("POST", "/save", true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            if (req.responseText.match(/^saved-at-timestamp: /)) {
                ts.innerHTML = req.responseText.substr("saved-at-timestamp: ".length)
            } else {
                ts.innerHTML = " LAST SAVE FAILED: " + req.responseText;
            }
        }
    };
    req.send(obj.toJSONString())
}

function add_row(id) {
    var req = new XMLHttpRequest();
    req.open("GET", "htmlget?type=add&id=" + encodeURIComponent(id), true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            var newrows = req.responseText.split("\u2029", 2);

            var maintable = document.getElementById("maintable");

            var newrow1 = maintable.insertRow(0)
            newrow1.setAttribute("class", "entry")
            newrow1.innerHTML = newrows[0]

            var newrow2 = maintable.insertRow(1)
            newrow2.setAttribute("id", "editor_"+encodeURIComponent(id))
            newrow2.setAttribute("class", "editor")
            newrow2.setAttribute("style", "display: none")
            newrow2.innerHTML = newrows[1]

            save_open_editor(true);
        }
    }
    req.send(null)
}

function add_entry(query) {
    var netext = document.getElementById('newentry').value;

    var req = new XMLHttpRequest();
    req.open("GET", "qadd?q=" + encodeURIComponent(query) + "&text=" + encodeURIComponent(netext), true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            if (req.responseText.match(/^added: /)) {
                newid = req.responseText.substr("added: ".length);
                add_row(newid)
                
                document.getElementById('newentry').value = "";
                
            } else {
                alert("ADD FAILED: " + req.responseText)
            }
            
        }
    };
    req.send(null);
    return false;
}

function fill_editor(name) {
    var ed = document.getElementById("ediv_"+name);
    var req = new XMLHttpRequest();
    req.open("GET", "get?id=" + encodeURIComponent(name), true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            var timestamp = req.responseText.split("\n", 2)[0];
            var jsonObj = req.responseText.substr(timestamp.length);
            v = jsonObj.parseJSON();
            ed.elements['edtitle'].value = v.Title;
            ed.elements['edtext'].value = v.Text;
            ed.elements['edat'].value = v.TriggerAt;
            ed.elements['edsort'].value = v.Sort;
            ed.elements['edid'].value = v.Id;
            ed.elements['edprio'].value = v.Priority;
            ed.elements['edfreq'].value = v.Freq;
            ed.elements['edcols'].value = v.Cols;
            var ts = document.getElementById("ts_" + v.Id);
            ts.innerHTML = timestamp;
        }
    }
    req.send(null)
}

function editor_from_row(row) {
    return document.getElementById("ediv_"+row.id.substr("editor_".length))
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
    var form = document.getElementById("ediv_"+name);
    save_editor(form);
}

function close_editor(row) {
    save_editor(editor_from_row(row));
    row.style['display'] = 'none';
}

function toggle_editor(name, event) {
    var row = document.getElementById("editor_"+name);
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

function change_priority(name, event) {
    var req = new XMLHttpRequest();
    req.open("GET", "change-priority?id=" + encodeURIComponent(name) + "&special=" + event.shiftKey, true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            if (req.responseText.match(/^priority-change-to: /)) {
                priority = req.responseText.substr("priority-change-to: ".length);
                priorityNum = priority[0]
                priority = priority.substr(2)
                var epr = document.getElementById('epr_'+name);
                epr.value = priority;
                epr.setAttribute("class", "priorityclass_" + priority);

                // changes the value saved inside the editor div so that saving the editor contents doesn't revert a changed priority
                var ed = document.getElementById("ediv_"+name);
                ed.elements["edprio"].value = priorityNum;
            } else {
                alert(req.responseText)
            }
        }
    }
    req.send(null)
}

function savesearch() {
    var name = prompt("save search to:");
    var req = new XMLHttpRequest();
    req.open("GET", "save-search?name=" + encodeURIComponent(name) + "&query=" + encodeURIComponent(document.getElementById('q').value), true);
    req.onreadystatechange = function() {
        if (req.readyState != 4) return;
        if (!req.responseText.match(/^query-saved: /)) {
            alert(req.responseText);
        }
    };
    req.send(null)
}