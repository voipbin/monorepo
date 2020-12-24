#!/bin/bash

# Defines
# $VOIPBIN_MAC_ADDRESS: Container's mac address
# $VOIPBIN_HOSTNAME: 

MAC_ADDRESS=$(cat /sys/class/net/eth0/address)
HOSTNAME=$(hostname)
OUTBOUND_PROXY=sip:10.164.0.20:5060

# VOIPBIN_MAC_ADDRESS
sed -i 's/$VOIPBIN_MAC_ADDRESS/'$MAC_ADDRESS'/g' /etc/asterisk/*
sed -i 's/$VOIPBIN_HOSTNAME/'$HOSTNAME'/g' /etc/asterisk/*
sed -i 's/$VOIPBIN_OUTBOUND_PROXY/'$OUTBOUND_PROXY'/g' /etc/asterisk/*

# Start gcsfuse
/usr/bin/gcsfuse --key-file /service_accounts/voipbin-production/service_account.json --implicit-dirs -o rw -o allow_other --file-mode 777 --dir-mode 777 voipbin-voip-media-bucket-europe-west4 /mnt
/bin/mkdir -p /var/spool/asterisk/recording
/bin/mount --bind /mnt/recording /var/spool/asterisk/recording

# Start asterisk
/usr/sbin/asterisk -fvvvvvvvg
