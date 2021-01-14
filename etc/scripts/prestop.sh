#!/bin/bash

/usr/sbin/asterisk -rx "core stop gracefully"

while :
do
    if pgrep -x "asterisk" > /dev/null
    then
        echo "Running"
        sleep 1
        continue
    else
        echo "Stopped"
        break
    fi
done
