#!/bin/bash

if [ $# -ne 3 ]
then
    echo "Usage: $0 <outbound proxy address> <target interface name> <gcp bucket name >"
    exit
fi

echo "Print status ------------------------------------"
echo "Public address: $(curl ifconfig.me)"

echo "Print inserted parameters -------------------------------------"
echo "Outbound proxy address: $1"
echo "Target interface name: $2"
echo "GCP bucket name: $3"

# Defines
VOIPBIN_OUTBOUND_PROXY_ADDR=$1
VOIPBIN_TARGET_INTERFACE_NAME=$2
VOIPBIN_GCP_BUCKET_NAME=$3
MAC_ADDRESS=$(cat /sys/class/net/$VOIPBIN_TARGET_INTERFACE_NAME/address)
HOSTNAME=$(hostname)

echo "Print env variables -------------------------------------"
echo "Outbound proxy address: $VOIPBIN_OUTBOUND_PROXY_ADDR"
echo "Target interface name: $VOIPBIN_TARGET_INTERFACE_NAME"
echo "GCP bucket name: $VOIPBIN_GCP_BUCKET_NAME"
echo "Mac address: $MAC_ADDRESS"
echo "Hostname: $HOSTNAME"

# VOIPBIN_MAC_ADDRESS
sed -i 's/VOIPBIN_MAC_ADDRESS/'$MAC_ADDRESS'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_HOSTNAME/'$HOSTNAME'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_OUTBOUND_PROXY/'$VOIPBIN_OUTBOUND_PROXY_ADDR'/g' /etc/asterisk/*

# Start gcsfuse
/usr/bin/gcsfuse --key-file /service_accounts/voipbin-production/service_account.json --implicit-dirs -o rw -o allow_other --file-mode 777 --dir-mode 777 $VOIPBIN_GCP_BUCKET_NAME /mnt
/bin/mkdir -p /var/spool/asterisk/recording
/bin/mount --bind /mnt/recording /var/spool/asterisk/recording

# Start http server for local file get
cd /mnt && python3 -m http.server 8000 &

# Start asterisk-exporter
/asterisk-exporter -web_listen_address ":2112" &

# Start asterisk
/usr/sbin/asterisk -fvvvvvvvg
