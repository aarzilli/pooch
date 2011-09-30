# Introduction

Pooch is a system to manage todo lists, calendar events and small text notes uniformly that allows the user to categorize all this entries by using a flexible tagging system.

It was inspired by the Chandler Project and Microsoft Entourage.

# Usage (web interface)

See the command line description for how to start your own private pooch server.

To use the online pooch you must register and then login. No email is required to register an account at the moment. Once you are logged in you can proceed to the index: [Task list](https://ddzuk.dyndns.info/list).

To add a todo entry click on "add entry", type some text and press enter. New entries will be triaged as "NOW" by default, by clicking on the priority column you can rotate through the triage states: NOW, LATER, DONE. A "DONE" entry will disappear when you refresh the page (but it will not be deleted, it can be retrieved, see below).

If you want to make your entry a NOTE (i.e. just a small text fragment, not something that needs to be done) shift-click on the priority button.

If you click on the title of a new entry and enter a date on the "When" field, and then click on the priority button your entry will become "TIMED", once the time specified in the "When" field arrives the entry will be automatically moved to "NOW".

When you add an entry you can add any tags you want, just like on twitter. Pooch will remember about those, you can restrict the current view to only show you entries tagged in some way by using the "change query" button.
You can also search for arbitrary text.


## Special tags

If when you add an entry you type this `#<date>` substituting a date for `<date>` the date will be saved in the when fields and the entry will be timed. For example:

    `Dentist appointment #2012-03-03`

Will make an entry named "Dentist appointment" that will go from "TIMED" to "NOW" the 3rd of March 2012. The other date format understood by the application is this `#3/10` which is the next october 3rd (be it this year or the next).

Another useful "special tag" is `#l`, when you type this the default triage of the entry will be "LATER" instead of "NOW".


## Calendar

By clicking "see as calendar" you will see all the entries matching the current query that have a "When" field set as a calendar.

## Seeing "DONE" entries

If you want your search to return "DONE" entries too add to the query the special tag `#:w/done`
