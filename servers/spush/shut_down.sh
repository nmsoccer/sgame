#!/bin/bash

LOG="./shut.log"
now=`date +'%F %T'`

echo "--------------$0 $1 $now--------------" >> $LOG
if [[ -z $1 ]]
then
  echo "no server found" >> $LOG
  exit 0
fi

killall -2 $1
sleep 1
result=`ps aux |grep "./$1" | grep -v 'grep'` 
if [[ -z $result ]]
then
  echo "shutdown $1 success!" >> $LOG
else
  echo "$1 still exist!" >> $LOG
fi


