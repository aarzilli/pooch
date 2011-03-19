/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

$(document).ready(function() {
        $('#calendar').fullCalendar({
                firstDay: 1, // start with monday
                    editable: false,
                    events: "/calevents?q="+encodeURIComponent(query),
                    theme: true,
                    header: {
                    left: 'prev,next today',
                        center: 'title',
                        right: 'month,basicWeek'
                        }
            })
            });

