#!/bin/bash

dest=$1
src=$2

sqlite3 $dest <<EOF
ATTACH '$src' AS src;
INSERT INTO tasks SELECT * FROM src.tasks;
INSERT INTO ridx SELECT * FROM src.ridx;
INSERT INTO columns SELECT * FROM src.columns;
DETACH src;
EOF
