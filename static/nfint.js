var jsonLoading = false;
var curRoot = "";
var curSelected = { id: "", name: "" };
var curCut = { id: "", name: "" };
var completeAction = null;
var completeActionHTML = "";

function loaded() {
	var searchstr = window.location.search.substr(1);
	var v = searchstr.split("&")
	for (var i = 0; i < v.length; i++) {
		var vv = v[i].split("=");
		if (vv[0] == "q") {
			curRoot = decodeURIComponent(vv[1]);
			break;
		}
	}
	setMenu("");
	var mainDiv = document.getElementById("main");
	loadChildrenAnd(mainDiv, curRoot, "0",
		function() {
			toggleExpandedAnd(curRoot, "1", function() { });
		});
}

function setMenu(errStr) {
	var menuDiv = document.getElementById("menu_cl");
	if (jsonLoading) {
		menuDiv.innerHTML = "LOADING&nbsp;<img src='loading.gif'>";
		return;
	}

	if (completeAction != null) {
		menuDiv.innerHTML = completeActionHTML;
		return;
	}

	if (errStr != "") {
		console.debug("ERROR:", errStr);
		menuDiv.innerHTML = "<span class='menu_err'>ERROR: " + errStr + "</span>";
		return;
	}

	if ((curCut.id != "") && (curSelected.id != "")) {
		menuDiv.innerHTML = "CUTTING " + curCut.id + " to " + curSelected.id + ": (v) as sibling, (V) as child, (escape) cancel";
		return;
	}

	if (curCut.id != "") {
		menuDiv.innerHTML = "CUTTING " + curCut.id + ": select destination or (escape)";
		return;
	}

	if (curSelected.id != "") {
		menuDiv.innerHTML = "SELECTED " + curSelected.id + ": (A) add children, (e) edit content, (o|a) add sibling object, (x) cut, (r) refresh, (d) delete, (g) goto, (escape) cancel";
		return
	}

	menuDiv.innerHTML = "READY: (r) refresh, (g) goto";
}

function complete_action_click() {
	completeAction();
	completeAction = null;
}

function cancel_action_click() {
	completeAction = null;
	setMenu("");
}

function makeHTTPRequest2(method, urlstr, content, fn) {
	jsonLoading = true;
	setMenu("");
	var m = new XMLHttpRequest();
	m.onreadystatechange = function() {
		if (m.readyState != 4) {
			return;
		}
		jsonLoading = false;
		if (m.status != 200) {
			setMenu("Request returned " + m.stauts);
			return;
		}
		setMenu("");
		fn(m.responseText);
	}
	m.open(method, urlstr, true);
	if (content != null) {
		m.setRequestHeader("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	}
	m.send(content);
}

function makeHTTPRequest(method, urlstr, content, fn) {
	makeHTTPRequest2(method, urlstr, content, function(data) {
		var obj = JSON.parse(data);
		if (obj.Error != "") {
			setMenu(obj.Error);
			return;
		}
		fn(obj);
	});
}

function loadChildrenAnd(adiv, rootName, c, fn) {
	adiv.innerHTML = "";
	makeHTTPRequest("GET", "/nf/list.json?id=" + encodeURIComponent(rootName) + "&c=" + c, null,
		function(responseObj) {
			fillDiv(adiv, responseObj);
			fn();
		});
}

function insertObjectRow(datable, id, n) {
	var r = datable.insertRow(n);
	r.classList.add("objectrow");
	r.id = "row_" + id;
	if (n < 0) {
		return r;
	}
	renumberRows(datable, n+1);
	return r;
}

function renumberRows(datable, start) {
	for (var i = start; i < datable.rows.length; ++i) {
		var cr = datable.rows[i];
		var idcolx = cr.getElementsByClassName("idcol")[0].getElementsByTagName("a")[0];
		if (idcolx.innerHTML.indexOf("-") < 0) {
			idcolx.innerHTML = "" + i;
		}
	}
}

function fillDiv(adiv, obj) {
	if (obj.Objects == null) {
		return;
	}
	var datable = document.createElement("table");
	datable.classList.add("objlist");
	adiv.appendChild(datable);

	for (var i = 0; i < obj.Objects.length; ++i) {
		var o = obj.Objects[i];
		var r = insertObjectRow(datable, o.Id, -1);
		fillObjectRow(r, o, false);
		if ((o.Children != null) && (o.Children.length > 0)) {
			var childscell = insertChildrenRowAndCell(datable, o.Id, -1);
			fillDiv(childscell, { Objects: o.Children });
		}
	}
}

function fillObjectRow(r, o, editor) {
	var z = ""
	if (o.ChildrenCount == 0) {
		z += "<td class='foldercol'>\uf096</td>";
	} else if ((o.Children != null) && (o.Children.length > 0)) {
		z += "<td class='foldercol'><a href='javascript:void(0)' onclick='click_folder(this);'>\uf147</a></td>";
	} else {
		z += "<td class='foldercol'><a href='javascript:void(0)' onclick='click_folder(this);'>\uf196</a></td>";
	}

	var name = o.Name;

	z += "<td class='idcol'><a href='/home?q=" + encodeURIComponent(o.Id) + "'>" + name + "</a></td>";

	if (editor) {
		z += "<td class='contentcol'><textarea id='contentform_" + o.Id + "'></textarea><input type='hidden' name='objname' value='" + o.Name + "'><br><input onclick='saveobject_click(this, false)' type='button' value='Save (C-s)'>&nbsp;<input type='button' value='Save and Close (escape)' onclick='saveobject_click(this, true)'>&nbsp;<input type='button' value='Rename' onclick='renameobject_click(this)'></td>";
	} else {
		if (o.Editable) {
			z += "<td class='contentcol'"
		} else {
			z += "<td class='contentcol uneditable'"
		}
		z += "onclick='click_content(this)'><div class='formattedtext'>" + o.FormattedText + "</div></td>";
	}

	if (o.Priority != "unknown") {
		z += "<td class='prioritycol'><input type='button' value='" + o.Priority.toUpperCase() + "' onclick='change_priority_click(this, event)'></div>";
	} else {
		z += "<td class='prioritycol'></div>";
	}

	r.innerHTML = z;

	if (editor) {
		var ta = document.getElementById("contentform_" + o.Id);
		ta.value = o.Body;
		ta.focus();
	}
}

function click_content(e) {
	var darow = getContainingRow(e);
	toggleSelected(getIdFromRow(darow));
}

function click_folder(e) {
	var darow = getContainingRow(e);
	toggleExpandedAnd(getIdFromRow(darow), "1", function() {});
	return false;
}

function saveobject_click(e, close) {
	var dacell = getContainingCell(e);
	var darow = getContainingRow(dacell);
	var daid = getIdFromRow(darow);
	var ta = dacell.getElementsByTagName("textarea")[0];
	makeHTTPRequest("POST", "/nf/update.json", "id=" + encodeURIComponent(daid) + "&body=" + encodeURIComponent(ta.value),
		function(returnObj) {
			fillObjectRow(darow, returnObj.Objects[0], !close);
		});
}

function renameobject_click(e) {
	var dacell = getContainingCell(e);
	var darow = getContainingRow(dacell);
	var daid = getIdFromRow(darow);

	var inputs = dacell.getElementsByTagName("input");
	var objname = null;
	for (var i = 0; i < inputs.length; ++i) {
		if (inputs[i].name == "objname") {
			objname = inputs[i].value;
			break;
		}
	}

	if (objname == null) {
		return;
	}

	objname = prompt("Rename object", objname);
	if (objname == null) {
		return;
	}

	makeHTTPRequest("GET", "/nf/update.json?id=" + encodeURIComponent(daid) + "&name=" + encodeURIComponent(objname), null,
		function(returnObj) {
			fillObjectRow(darow, returnObj.Objects[0], true);
		});
}

function getContainingElement(e, tagName) {
	tagName = tagName.toLowerCase();
	for (var el = e; el != null; el = el.parentNode) {
		if (el.tagName.toLowerCase() == tagName) {
			return el;
		}
	}
	return null;

}

function getContainingRow(e) {
	return getContainingElement(e, "tr");
}

function getContainingTable(e) {
	return getContainingElement(e, "table");
}

function getContainingCell(e) {
	return getContainingElement(e, "td");
}

function getIdFromRow(darow) {
	if (darow.id.indexOf("row_") == 0) {
		return darow.id.substring("row_".length);
	}
	if (darow.id.indexOf("childs_") == 0) {
		return darow.id.substring("childs_".length);
	}
	return "";
}

function getRowFromId(daid) {
	return document.getElementById("row_" + daid);
}

function toggleSelected(daid) {
	if (curSelected.id != daid) {
		switchSelected(daid);
	}

	setMenu("");
	return
}

function switchSelected(daid) {
	if (curSelected.id != "") {
		var row = document.getElementById("row_" + curSelected.id);
		if (row != null) {
			row.classList.remove("selected");
		}
	}
	curSelected.id = daid;
	if (curSelected.id != "") {
		var row = document.getElementById("row_" + curSelected.id);
		if (row != null) {
			row.classList.add("selected");
		}
	}
}

function setCut(daid) {
	if (curCut.id != "") {
		var row = document.getElementById("row_" + curCut.id);
		if (row != null) {
			row.classList.remove("cutting");
		}
	}
	curCut.id = daid;
	if (curCut.id != "") {
		var row = document.getElementById("row_" + curCut.id);
		if (row != null) {
			row.classList.add("cutting");
		}
	}
}

function getChildrenRow(daid) {
	return document.getElementById("childs_" + daid);
}

function insertChildrenRowAndCell(datable, daid, i) {
	childsrow = datable.insertRow(i);
	childsrow.id = "childs_" + daid;
	childsrow.classList.add("childrow");
	var childscell = childsrow.insertCell(-1);
	childscell.colSpan = 4;
	return childscell;
}

function toggleExpandedAnd(daid, cparam, fn) {
	var childsrow = getChildrenRow(daid);
	if (childsrow == null) {
		var darow = getRowFromId(daid);
		var folderx = darow.getElementsByClassName("foldercol")[0].getElementsByTagName("a")[0];
		var datable = getContainingTable(darow);
		for (var i = 0; i < datable.rows.length; ++i) {
			if (datable.rows.item(i).id == darow.id) {
				var childscell = insertChildrenRowAndCell(datable, daid, i+1)
				loadChildrenAnd(childscell, daid, cparam, fn);
				break;
			}
		}
		if (folderx != null) {
			folderx.innerHTML = "\uf147";
		}
	} else {
		var darow = getRowFromId(daid);
		var folderx = darow.getElementsByClassName("foldercol")[0].getElementsByTagName("a")[0];
		var datable = getContainingTable(childsrow);
		for (var i = 0; i < datable.rows.length; ++i) {
			if (datable.rows.item(i).id == childsrow.id) {
				datable.deleteRow(i);
				break;
			}
		}
		if (folderx != null) {
			folderx.innerHTML = "\uf196";
		}
	}
}

function keypressfn(e) {
	var ae = document.activeElement;

	if (ae != null) {
		var n = ae.tagName.toLowerCase();
		if (n == "textarea") {
			if (e.ctrlKey && (e.key == "s")) {
				saveobject_click(ae, false);
				return false;
			} else if (e.key == "Esc") {
				saveobject_click(ae, true);
				return false
			}
			return;
		} else if ((n == "input") && (ae.getAttribute("type") != "button")) {
			return;
		}
	}

	if (e.key == "Esc") {
		switchSelected("");
		setCut("");
		makeHTTPRequest("PUT", "/nf/curcut?id=", null, function(obj) { });
		setMenu("");
		return;
	}

	if (e.key == "r") {
		reloadEverything()
		return;
	}

	if ((curCut.id != "") && (curSelected.id != "")) {
		if (e.key == "v") {
			pasteSibling(curSelected.id);
			return;
		} else if (e.key == "V") {
			pasteChildren(curSelected.id);
			return;
		}
		return;
	}

	if (e.key == "g") {
		if (curSelected.id != "") {
			curRoot = curSelected.id;
			curSelected.id = "";
			history.pushState("/home?q=" + encodeURIComponent(curRoot));
			toggleExpandedAnd(curRoot, "1", function() { });
			reloadEverything();
		} else {
			completeActionHTML = "Go To: <form onsubmit='complete_action_click()'><input id='gotobox'/>&nbsp;<input type='submit' value='Go'></form><a id='classiclink' href='#'>classic</a>";
			completeAction = function() {
				var gotobox = document.getElementById('gotobox');
				curRoot = idConvertSpaces(gotobox.value);
				history.pushState(null, "", "/home?q=" + encodeURIComponent(curRoot));
				//toggleExpandedAnd(curRoot, "1", function() { });
				reloadEverything();
			}
			setMenu("");
			var gotobox = document.getElementById('gotobox');
			gotobox.value = idRevertSpaces(curRoot);
			gotobox.focus();

			var classiclink = document.getElementById('classiclink');
			classiclink.href = "/list?q=" + encodeURIComponent(gotobox.value)
		}
		return;
	}

	if (curSelected.id != "") {
		if (e.key == "A") {
			addChildren(curSelected.id);
			return;
		} else if (e.key == "e") {
			openEditor(curSelected.id);
			return;
		} else if ((e.key == "o") || (e.key == "a")) {
			var daid = curSelected.id;
			switchSelected("");
			setMenu("");
			addSibling(daid);
			return;
		} else if (e.key == "x") {
			var daid = curSelected.id;
			switchSelected("")
			setCut(daid);
			setMenu("");
			makeHTTPRequest("PUT", "/nf/curcut?id=" + encodeURIComponent(daid), null, function(obj) { });
			return;
		} else if (e.key == "d") {
			completeAction = function() {
				var daid = curSelected.id;
				switchSelected("");
				setMenu("");
				deleteObject(daid);
			};
			completeActionHTML = "<span class='menu_complete'>Delete object " + curSelected.id + "?&nbsp;</span><input type='button' onclick='complete_action_click()' value='Yes'/><input type='button' onclick='cancel_action_click()' value='No'/>";
			setMenu("");

			return;
		} else if (e.key == " ") {
			toggleExpandedAnd(curSelected.id, "1", function() { });
			return;
		}
	}
}

function reloadObjectAnd(daid, fn) {
	makeHTTPRequest("GET", "/nf/list.json?id=" + encodeURIComponent(daid) + "&c=0", null,
		function(responseObj) {
			if (responseObj.Objects.length != 1) {
				setMenu("Wrong return value from list.json");
				return;
			}
			fn(responseObj);
		});
}

function openEditor(daid) {
	reloadObjectAnd(daid,
		function(responseObj) {
			var o = responseObj.Objects[0];
			if (!o.Editable) {
				setMenu("Item can not be edited");
				return;
			}

			var darow = getRowFromId(daid);
			fillObjectRow(darow, o, true);
			toggleSelected(daid);
		});
}

function deleteObjectRow(daid) {
	var darow = getRowFromId(daid);
	var datable = null;
	if (darow != null) {
		var datable = getContainingTable(darow);
		var i = 0;
		for (; i < datable.rows.length; ++i) {
			if (darow.id == datable.rows[i].id) {
				datable.deleteRow(i);
				break;
			}
		}
		renumberRows(datable, 0);
	}
	var childrow = getChildrenRow(daid);
	if ((childrow != null) && (datable != null)) {
		for (var i = 0; i < datable.rows.length; ++i) {
			if (childrow.id == datable.rows[i].id) {
				datable.deleteRow(i);
				break;
			}
		}
	}
}

function deleteObject(daid) {
	makeHTTPRequest("GET", "/nf/remove.json?id=" + encodeURIComponent(daid), null,
		function(responseObj) {
			deleteObjectRow(daid);
		});
}

function expandOrAddObjectRow(daid, o, n, editor) {
	reloadObjectAnd(daid,
		function(obj) {
			var darow = getRowFromId(daid);
			fillObjectRow(darow, obj.Objects[0], false);
		});
	var contentrow = getChildrenRow(daid);
	if (contentrow == null) {
		toggleExpandedAnd(daid, "1",
			function() {
				if (editor) {
					openEditor(o.Id);
				}
			});
	} else {
		var datable = contentrow.getElementsByTagName("td")[0].getElementsByTagName("table")[0];
		var r = insertObjectRow(datable, o.Id, n);
		switchSelected(o.Id);
		setMenu("");
		fillObjectRow(r, o, editor);
	}
}

function addChildren(daid) {
	makeHTTPRequest("GET", "/nf/new.json?id=" + encodeURIComponent(daid) + "&n=0", null,
		function(responseObj) {
			expandOrAddObjectRow(daid, responseObj.Objects[0], 0, true);
		});
}

function getParentOf(daid) {
	var darow = getRowFromId(daid);
	var datable = getContainingTable(darow);
	var prow = getContainingRow(datable);
	if (prow == null) {
		return null
	}
	var n = -1;
	for (var i = 0; i < datable.rows.length; ++i) {
		if (datable.rows[i].id == darow.id) {
			n = i;
			break;
		}
	}
	return [ getIdFromRow(prow), datable, n ];
}

function addSibling(daid) {
	p = getParentOf(daid);
	if (p == null) {
		setMenu("Can't add siblings to current root");
		return;
	}

	pid = p[0]
	datable = p[1]
	n = p[2]+1

	makeHTTPRequest("GET", "/nf/new.json?id=" + encodeURIComponent(pid) + "&n=" + n, null,
		function(responseObj) {
			var r = insertObjectRow(datable, responseObj.Objects[0].Id, n);
			switchSelected(responseObj.Objects[0].Id);
			fillObjectRow(r, responseObj.Objects[0], true);
		});
}

function pasteChildren(daid) {
	if (curCut.id == "") {
		setMenu("No selected cut");
		return;
	}

	makeHTTPRequest("GET", "/nf/move.json?id=" + encodeURIComponent(curCut.id) + "&p=" + encodeURIComponent(daid) + "&n=0", null,
		function(responseObj) {
			deleteObjectRow(curCut.id);
			setCut("");
			makeHTTPRequest("PUT", "/nf/curcut?id=", null, function(obj) { });
			expandOrAddObjectRow(daid, responseObj.Objects[0], 0, false);
		});
}

function pasteSibling(daid) {
	if (curCut.id == "") {
		setMenu("No selected cut");
		return;
	}

	p = getParentOf(daid);
	if (p == null) {
		setMenu("Can't add siblings to current root");
		return;
	}

	pid = p[0]
	datable = p[1]
	n = p[2]+1

	makeHTTPRequest("GET", "/nf/move.json?id=" + encodeURIComponent(curCut.id) + "&p=" + encodeURIComponent(pid) + "&n=" + n, null,
		function(responseObj) {
			deleteObjectRow(curCut.id);
			setCut("");
			makeHTTPRequest("PUT", "/nf/curcut?id=", null, function(obj) { });
			expandOrAddObjectRow(pid, responseObj.Objects[0], parseInt(responseObj.Objects[0].Name), false);
		});
}

function change_priority_click(e, event) {
	var dacell = getContainingCell(e);
	var darow = getContainingRow(dacell);
	var daid = getIdFromRow(darow);
	if (daid.indexOf("#id=") != 0) {
		setMenu("Internal error");
		return;
	}
	var realId = stripToRealId(daid);
	guess_next_priority(daid, e, event.shiftKey);
	makeHTTPRequest2("GET", "/change-priority?id=" + encodeURIComponent(realId) + "&special=" + event.shiftKey, null,
		function(data) {
			if (data.match(/^priority-change-to: /)) {
				priority = data.substr("priority-change-to: ".length);
				priorityNum = priority[0];
				priority = priority.substr(2);
				change_priority_to(daid, e, priorityNum, priority);
			} else {
				setMenu(data);
			}
		});
}

function stripToRealId(id) {
	return id.substr("#id=".length);
}

function guess_next_priority(daid, e, special) {
	current = e.value;
	if (current == "NOTES") {
		if (special) {
			change_priority_to(daid, e, 1, "NOW");
		} else {
			change_priority_to(daid, e, 3, "STICKY");
		}
	} else if (current == "STICKY") {
		if (special) {
			change_priority_to(daid, e, 1, "NOW");
		} else {
			change_priority_to(daid, e, 3, "NOTES");
		}
	} else {
		if (special) {
			change_priority_to(daid, e, 3, "NOTES");
		} else {
			if (current == "LATER") {
				change_priority_to(daid, e, 1, "NOW");
			} else if (current == "NOW") {
				change_priority_to(daid, e, 5, "DONE");
			} else {
				change_priority_to(daid, e, 2, "LATER");
			}
		}
	}
}

function change_priority_to(daid, e, priorityNum, priority) {
	e.value = priority;
}

function reloadEverything() {
	makeHTTPRequest("GET", "/nf/curcut", null,
		function(responseObj) {
			curCut.id = responseObj.Objects[0].Id;
			var allTrs = document.getElementsByTagName("tr");
			var childs = [ curRoot ];
			for (var i = 0; i < allTrs.length; i++) {
				if (allTrs[i].id.indexOf("childs_") == 0) {
					childs.push(allTrs[i].id.substr("childs_".length))
				}
			}
			function cont() {
				for (var i = 0; i < childs.length; i++) {
					if (childs[i] == null) {
						continue;
					}
					var darow = getRowFromId(childs[i]);
					if (darow != null) {
						var dachildrow = getChildrenRow(childs[i]);
						if (dachildrow != null) {
							childs[i] = null;
							continue;
						}
						var daid = childs[i];
						childs[i] = null;
						toggleExpandedAnd(daid, "2", cont);
						return;
					}
				}
				setCut(curCut.id);
				if (curSelected.id != "") {
					var row = document.getElementById("row_" + curSelected.id);
					if (row != null) {
						row.classList.add("selected");
					}
				}
			}
			var mainDiv = document.getElementById("main");
			loadChildrenAnd(mainDiv, curRoot, "0", cont);
		})
}

function idConvertSpaces(a) {
	return a.replace(" ", "·");
}

function idRevertSpaces(a) {
	return a.replace("·", " ");
}

document.addEventListener('DOMContentLoaded', loaded, false);
document.onkeypress = keypressfn;
