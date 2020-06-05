#!/bin/bash

#BIN_FILES=("manager" "creater" "deleter" "carrier" "tester")
BIN_FILES=("manager" "creater" "deleter" "carrier")
LIB_FILE="./carrier_base.c ./carrier_lib.c ../library/proc_bridge.c"
HEADER_FILE="../library/"
DYN_LIBS="-Wl,-Bstatic -lslog -lstlv -Wl,-Bdynamic -lm -lrt"

function make()
{
  echo "compiling..."
  gcc -g -Wall -I${HEADER_FILE} bridge_manager.c ${LIB_FILE} ${DYN_LIBS} -o manager
  gcc -g -w -I${HEADER_FILE} create_bridge.c ${LIB_FILE} ${DYN_LIBS} -o creater
  gcc -g -w -I${HEADER_FILE} delete_bridge.c ${LIB_FILE} ${DYN_LIBS} -o deleter

  #if using debug
  #gcc -g -D_TRACE_DEBUG -Wall -I${HEADER_FILE} bridge_carrier.c ${LIB_FILE} ${DYN_LIBS} -o carrier 
  gcc -g -Wall -I${HEADER_FILE} bridge_carrier.c ${LIB_FILE} ${DYN_LIBS} -o carrier
  echo "compiling finish"
}

function clean()
{
  echo "cleaning..."
  for file in ${BIN_FILES[@]}
  do
    rm ${file}
  done
  echo "done"
}

function show_help()
{
  echo "usage: ./make.sh <make|clean>"
  echo "  make: compile and create binary files"
  echo "  clean: clean binary files"
}

function main()
{
  if [[ $# -lt 1 ]]
  then
    show_help
    exit 0
  fi 

  if [[ $1 == "make" ]]
  then
    make
  elif [[ $1 == "clean" ]]
  then
    clean
  else
    show_help
  fi 

  return
}

main $@
