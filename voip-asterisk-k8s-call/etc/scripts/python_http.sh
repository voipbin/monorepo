#!/bin/sh

while true
do
    echo "Starting python-http for asterisk media handle."
    /usr/bin/python3 -m http.server 8000
    sleep 5
done
