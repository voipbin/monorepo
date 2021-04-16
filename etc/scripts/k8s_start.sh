#!/bin/bash

if [ $# -ne 2 ]
then
    echo "Usage: $0 <target interface name> <gcp bucket name >"
    exit
fi

echo "Print inserted parameters-------------------------------------"
echo "Target interface name: $1"
echo "GCP bucket name: $2"

# Defines
VOIPBIN_TARGET_INTERFACE_NAME=$1
VOIPBIN_GCP_BUCKET_NAME=$2
MAC_ADDRESS=$(cat /sys/class/net/$VOIPBIN_TARGET_INTERFACE_NAME/address)
HOSTNAME=$(hostname)

echo "Print env variables-------------------------------------"
echo "Target interface name: $VOIPBIN_TARGET_INTERFACE_NAME"
echo "GCP bucket name: $VOIPBIN_GCP_BUCKET_NAME"
echo "Mac address: $MAC_ADDRESS"
echo "Hostname: $HOSTNAME"

# VOIPBIN_MAC_ADDRESS
sed -i 's/VOIPBIN_MAC_ADDRESS/'$MAC_ADDRESS'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_HOSTNAME/'$HOSTNAME'/g' /etc/asterisk/*

# Start gcsfuse
/usr/bin/gcsfuse --key-file /service_accounts/voipbin-production/service_account.json --implicit-dirs -o rw -o allow_other --file-mode 777 --dir-mode 777 $VOIPBIN_GCP_BUCKET_NAME /mnt

# Set cron - recording move script
/bin/mkdir -p /var/spool/asterisk/recording
crontab -l | { cat; echo "* * * * * /cron_recording_move.sh"; } | crontab -

# Start cron
service cron start

# Start asterisk-exporter
/asterisk-exporter -web_listen_address ":2112" &

# Start asterisk
/usr/sbin/asterisk -fvvvvvvvg
