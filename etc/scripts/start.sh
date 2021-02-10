#!/bin/bash

if [ $# -ne 6 ]
then
    echo "Usage: $0 <target interface name> <database hostname> <database port> <database name> <database username> <database password>"
    exit
fi

echo "Print inserted parameters-------------------------------------"
echo "Target interface name: $1"
echo "Database hostname: $2"
echo "Database port: $3"
echo "Database name: $4"
echo "Database username: $5"
echo "Database password: $6"

# Defines
VOIPBIN_TARGET_INTERFACE_NAME=$1
VOIPBIN_DATABASE_HOSTNAME=$2
VOIPBIN_DATABASE_PORT=$3
VOIPBIN_DATABASE_NAME=$4
VOIPBIN_DATABASE_USERNAME=$5
VOIPBIN_DATABASE_PASSWORD=$6

MAC_ADDRESS=$(cat /sys/class/net/$VOIPBIN_TARGET_INTERFACE_NAME/address)
HOSTNAME=$(hostname)

echo "Print env variables-------------------------------------"
echo "Target interface name: $VOIPBIN_TARGET_INTERFACE_NAME"
echo "Mac address: $MAC_ADDRESS"
echo "Hostname: $HOSTNAME"
echo "Database hostname: $VOIPBIN_DATABASE_HOSTNAME"
echo "Database port: $VOIPBIN_DATABASE_PORT"
echo "Database name: $VOIPBIN_DATABASE_NAME"
echo "Database username: $VOIPBIN_DATABASE_USERNAME"
echo "Database password: $VOIPBIN_DATABASE_PASSWORD"

# Substitute the asterisk variables
sed -i 's/VOIPBIN_MAC_ADDRESS/'$MAC_ADDRESS'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_HOSTNAME/'$HOSTNAME'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_DATABASE_HOSTNAME/'$VOIPBIN_DATABASE_HOSTNAME'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_DATABASE_PORT/'$VOIPBIN_DATABASE_PORT'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_DATABASE_NAME/'$VOIPBIN_DATABASE_NAME'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_DATABASE_USERNAME/'$VOIPBIN_DATABASE_USERNAME'/g' /etc/asterisk/*
sed -i 's/VOIPBIN_DATABASE_PASSWORD/'$VOIPBIN_DATABASE_PASSWORD'/g' /etc/asterisk/*


# Start asterisk
/usr/sbin/asterisk -fvvvvvvvg
