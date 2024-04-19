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
        echo "Stopped the Asterisk. Moving recording files"
        /cron_recording_move.sh
        break
    fi
done
