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

# Installation

1. Download and install [google go](http://golang.org/).

2. Download and install [gosqlite](http://code.google.com/p/gosqlite/).

3. Download and make pooch, copy the executable somewhere you like
   At the moment google go has serious deployment problems (the executable is dynamically linked to shared objects referred to with absolute paths). Once this problem is fixed (or someone explains to me how to circumvent it) I will make a precompiled stand-alone binary available

4. Create a directory where you want to store the pooch tasklist files and point the POOCHPATH environment variable to it.

# Usage

The rest of this README will explain how to use the GUI of pooch, there is also a command line interface that you can use but if you can use a command line interface you can probably figure out how to use it by yourself, start with:

   pooch help

to see a list of commands and:

  pooch help <command>

to see a short description of how <command> works.

To create a new tasklist type:

  pooch create $POOCHPATH/<mytasklist>

(at the moment it is not possible to create new tasklists from the http interface)

To start the http server type:

  pooch serve <port>

Now you can use your browser (at the moment only firefox has been tested) to go to this URLs:

  http://127.0.0.1:<port>/list?tl=<mytasklist>

which shows you the tasks of <mytasklist>, and:

  http://127.0.0.1:<port>/cal?tl=*

which shows you all your events (switch a tasklist name with the "*" to see only the events in that tasklist).

Each tasklist entry is associated with a 'priority' or 'triage status', available priorities are:

1. NOW
   New entries are created by default with this priority, it means it should be addressed "soon". 
2. LATER
   This means the entry is a todo item that will need to be addressed at some point.
3. DONE
   This means the entry has been addressed, DONE entries are not shown by default
4. TIMED
   This means the entry is a calendar item, it will move from TIMED to NOW when its "trigger" time is passed
5. NOTE / STICKY
   This means the entry is a text note. The difference between NOTE and STICKY is just in the order they are shown (STICKY notes are put before NOW entries, simple NOTEs are put after LATER entries)

You can change the priority of an entry by clicking AND shift clicking the priority label. For example clicking on a NOW entry will cycle through NOW / LATER / DONE, while shift clicking will go to NOTE and STICKY NOTE.

You can also create an entry with a specific priority by using a quick tag in the "New entry" field, as described in the following section.

# Quick tags format

* @l or @later -> LATER
* @d or @done -> DONE
* @$, @N or @Notes -> NOTE
* @$$, @StickyNotes -> STICKY note
* @<date> -> TIMED (will trigger when <date> is reached)
* @<date>+<number> -> TIMED like @<date> but the event will repeat every <number> days

