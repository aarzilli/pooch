# Introduction

Pooch is a system to organize todo lists, calendar events and small text notes uniformly. I wrote it because I think that calendar applications and todo list manager help you organize basically the same kind of information but the radically different approach they take force you to either force todo items into a calendar (or live with whatever, usually poor, management of non-timed tasks your calendar applications provides) or to manually manage the time aspect of todo items.

It is interesting how this dicotomy (between todo managers and calendar applications) mirrors somehow the [Maker's Schedule / Manager's Scheduler dicotomy](http://www.paulgraham.com/makersschedule.html).

The concept is not new both Microsoft Entourage and Chandler provide features similar to what is implemented in Pooch.

# History

I was a [Chandler](http://chandlerproject.org/) user for over a year. I really liked the ability to mix "todo" items with a triage status (now/later/done) with calendar events that would automatically move from LATER to NOW at the set time. Unfortunately the Chandler Project is mostly abandoned: it crashed frequently and in two occasions corrupted its database irreparably (had backups). In addition the binaries they ship don't work on the last two versions of Ubuntu and on the last version of Mac OSX.

So, instead of dealing with all the features of chandler I didn't really use or want, I decided to reimplement the small subset of features of Chandler that I actually use fixing some of the things I miss in Chandler along the way, namely:
1. Distinguishing "timed" events (that can not be done before a specific time) from todo items that are simply postponed
2. Management of small text notes that are not todo items nor calendar events
3. Scriptability (yes, in theory Chandler should be completely extensible, yet I dare you to actually do it)

# Usage (web interface)

Pooch is available online at [https://ddzuk.dyndns.info/list](https://ddzuk.dyndns.info/list). See the command line description for how to start your own private pooch server.

To use the online pooch you must register and then login. No email is required to register an account at the moment. Once you are logged in you can proceed to the index: [Task list](https://ddzuk.dyndns.info/list).

## Normal entries

You can add new entries by typing any text in the "New entry" field and clicking "add". By defult new tasks will be assigned the priority "NOW", this priority is for tasks that should be done immediately. By clicking on the priority button you can move your task into the "LATER" priority (things that you want to do some time later) or "DONE" when the task has been executed.

## Timed entries

If you have to add a task that can not be done before a certain date you should enter a text like this in the "New entry" field:

   end of the world @2012-12-21

this will add a task called "end of the world" with priority "TIMED", this task will automatically trigger to "NOW" at midnight on 21th december 2012. If the task is scheduled for some time this year you can use the shorthand format: @21/12 (note the european order for day/month).
If you don't like that the trigger happens at midnight you can change it like this:

   end of the world @2012-12-21,10:00

this means 21st december 2012 at 10am.

## Notes

You can also add to pooch entries that are not tasks, in pooch-speak this entries are called NOTES, to change a task into a note shift-click its priority button. You can also write this text into the "New entry" field:

   this is a note #$

## Hashtags

We have seen that the special hashtag #$ creates NOTEs, well you can use hashtags to tag any note/task into categories, for example:

   buy breakfast #home

will create a new task "buy breakfast" and assign it to the "home" category. If the "home" category doesn't exist it will be created automatically and displayed to the right of the task list. If no category is specified the category #uncat will be used.

## Editing entries

To edit existing entries you can click on their title, a series of forms will appear to let you change title, text and categories assigned to the entry.

## Search

You can both search with a free text query or by entering an hashtag. If you are viewing the results of a search by hashtag and enter a new entry the new entry will be automatically associated with the searched hashtag.

At any moment you can click the "as calendar" link and view the results of the search as calendar entries (it will only show TIMED entries).

## Hashtags format

* `#l` or `#later` -> LATER
* `#d` or `#done` -> DONE
* `#$`, `#N` or `#Notes` -> NOTE
* `#$$`, `#StickyNotes` -> STICKY note
* `#<date>` -> TIMED (will trigger when <date> is reached)
* `#<date>,<time>` -> TIMED (also has time)
* `#<date>+<number>` -> TIMED like `#<date>` but the event will repeat every <number> days
* `#<category>` -> assigns the entry to <category>, creates <category> if it doesn't exist.

# Usage (command line)

The rest of this README will explain how to start http servers. Pooch can be used fully from the command line but if you want to use the command line you should be smart enough to figure out how it works by yourself.

  pooch help

to see a list of commands and:

  pooch help <command>

to see a short description of how <command> works.

To create a new pooch file use

  pooch create <path to pooch file>

To start the http server type:

  POOCHDB=<path to pooch file> pooch serve <port>

Now you can use your browser (at the moment only firefox has been tested) to go to this URLs:

  http://127.0.0.1:<port>/list?tl=<mytasklist>

which shows you the tasks of <mytasklist>, and:

  http://127.0.0.1:<port>/cal?tl=*

which shows you all your events (switch a tasklist name with the "*" to see only the events in that tasklist).

To start the multiuser http server create a directory where the users information will be stored and then run:

  pooch multiserve <port> <multiserve-directory> <path to logfile>

Inside the serv/ directory there are some configuration files that could be useful if you want to replicate the installation of pooch at [https://ddzuk.dyndns.info/](https://ddzuk.dyndns.info/).

# Installation

1. Download and install [google go](http://golang.org/).

2. Download and install [gosqlite](http://code.google.com/p/gosqlite/).

3. Download and make pooch, copy the executable somewhere you like
   At the moment google go has serious deployment problems (the executable is dynamically linked to shared objects referred to with absolute paths). Once this problem is fixed (or someone explains to me how to circumvent it) I will make a precompiled stand-alone binary available

4. Create a directory where you want to store the pooch tasklist files and point the POOCHPATH environment variable to it.

