#!/bin/bash

# This script checks the Asterisk's recording directory and handle the recording files.

ASTERISK_RECORDING_DIRECTORY=/var/spool/asterisk/recording
GCSFUSE_RECORDING_DIRECTORY=/mnt/recording

for RECORDING in $ASTERISK_RECORDING_DIRECTORY/*
do
  # check filesize. we don't do anything if the filesize is 0
  if [[ ! -s $RECORDING ]]
  then
    # the asterisk is working on this file.
    continue
  fi

  # check the filename. if the filename starts with tmp-, then delete it.
  if [[ $RECORDING == $ASTERISK_RECORDING_DIRECTORY/tmp-* ]] ;
  then
    echo "Found temp recording file. Deleting the recording. recording: $RECORDING"
    rm $RECORDING $GCSFUSE_RECORDING_DIRECTORY
    continue
  fi

  echo "Found asterisk recording. Moving to cloude storage. recording: $RECORDING"
  mv $RECORDING $GCSFUSE_RECORDING_DIRECTORY

done
