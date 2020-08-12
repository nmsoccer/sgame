#!/bin/bash

DIR=("conn_serv" "db_serv" "logic_serv" "manage_serv" "disp_serv")
cd ..
WORKING=`pwd`

function build()
{
  for dir in ${DIR[@]}
  do
    #compile
    cd $dir
    rm $dir #remove old
    dst=$dir
    src=$dir.go
    go build $src

    #check
    if [[ ! -e  $dst ]]
    then
      echo "build $dst failed!"
      exit 0
    else
      echo "compile $dst success"
      ls -l $dst
    fi

    cd $WORKING 
  done
}

function clear() 
{
  for dir in ${DIR[@]}
  do
    #clear
    cd $dir
    dst=$dir
    rm $dst

    #check
    if [[ ! -e  $dst ]]
    then
      echo "clear $dst success"
    else
      echo "clear $dst failed"
      ls -l $dst
    fi

    cd $WORKING 
  done
}

while getopts "bc" arg 
do
  case $arg in
    b)
     echo "build"
     build
    ;;
    c)
     echo "clear"
     clear
    ;;
    ?)
     echo "-b build all target"
     echo "-c clear all target"
    ;;
  esac
done
