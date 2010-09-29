#!/bin/bash

for i in $(pooch search | cut -d' ' -f1 | grep '^task'); do
    pooch rename $i
done