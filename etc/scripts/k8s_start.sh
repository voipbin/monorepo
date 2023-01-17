#!/bin/bash

if [ $# -ne 4 ]
then
    echo "Usage: $0 <outbound proxy address> <target interface name> <gcp bucket name >"
    exit
fi

echo "Print status ------------------------------------"
echo "Public address: $(curl ifconfig.me)"

echo "Print inserted parameters -------------------------------------"
echo "Outbound proxy address: $1"
echo "Target interface name: $2"
echo "GCP bucket name media: $3"
echo "GCP bucket name temp: $4"

# Defines
VOIPBIN_OUTBOUND_PROXY_ADDR=$1
VOIPBIN_TARGET_INTERFACE_NAME=$2
VOIPBIN_GCP_BUCKET_NAME_MEDIA=$3
VOIPBIN_GCP_BUCKET_NAME_TEMP=$4
MAC_ADDRESS=$(cat /sys/class/net/$VOIPBIN_TARGET_INTERFACE_NAME/address)
HOSTNAME=$(hostname)

echo "Print env variables -------------------------------------"
echo "Outbound proxy address: $VOIPBIN_OUTBOUND_PROXY_ADDR"
echo "Target interface name: $VOIPBIN_TARGET_INTERFACE_NAME"
echo "GCP bucket name media: $VOIPBIN_GCP_BUCKET_NAME_MEDIA"
echo "GCP bucket name temp: $VOIPBIN_GCP_BUCKET_NAME_TEMP"
echo "Mac address: $MAC_ADDRESS"
echo "Hostname: $HOSTNAME"

# VOIPBIN_MAC_ADDRESS
sed -i 's/VOIPBIN_MAC_ADDRESS/'$MAC_ADDRESS'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_HOSTNAME/'$HOSTNAME'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_OUTBOUND_PROXY/'$VOIPBIN_OUTBOUND_PROXY_ADDR'/g' /etc/asterisk/*

# Start gcsfuse
mkdir /mnt/media
/usr/bin/gcsfuse --key-file /service_accounts/voipbin-production/service_account.json --implicit-dirs -o rw -o allow_other --file-mode 777 --dir-mode 777 $VOIPBIN_GCP_BUCKET_NAME_MEDIA /mnt/media

mkdir /mnt/temp
/usr/bin/gcsfuse --key-file /service_accounts/voipbin-production/service_account.json --implicit-dirs -o rw -o allow_other --file-mode 777 --dir-mode 777 $VOIPBIN_GCP_BUCKET_NAME_TEMP /mnt/temp


# Set cron - recording move script
/bin/mkdir -p /var/spool/asterisk/recording
crontab -l | { cat; echo "* * * * * /cron_recording_move.sh"; } | crontab -

# Start cron
service cron start

# Start http server for local file get
cd /mnt
/python_http.sh &

# Start asterisk-exporter
/asterisk-exporter -web_listen_address ":2112" &

# Start asterisk
/usr/sbin/asterisk -fvvvvvvvg
