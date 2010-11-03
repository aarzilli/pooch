/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

$(document).ready(function() {
    $('#calendar').fullCalendar({
      firstDay: 1, // start with monday
	  editable: false,
      events: "/calevents?q="+encodeURIComponent(query),
	  header: {
	left: 'prev,next today',
	    center: 'title',
	    right: 'month,basicWeek'
	}
      })
      });

function add_entry() {
    var netext = document.getElementById('newentry').value;
    var cal = document.getElementById('calendar');

    var req = XMLHttpRequest()
    req.open("GET", "qadd?text=" + escape(netext), true);
    req.onreadystatechange = function() {
        if (req.readyState == 4) {
            if (req.responseText.match(/^added: /)) {
	      window.location.reload()
            } else {
	      alert("ADD FAILED: " + req.responseText);
            }
            
        }
    };
    req.send(null);
    return false;
}
