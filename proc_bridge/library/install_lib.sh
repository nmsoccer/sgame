#!/bin/bash
HEADER_DIR=/usr/local/include/proc_bridge/
LIB_DIR=/usr/local/lib/
WORK_DIR=`pwd`
SRC_FILE="proc_bridge.c"
DYN_FILE="libproc_bridge.so"
LIB_FILE="libproc_bridge.a"

function main()
{
  #make dir
  mkdir -p ${HEADER_DIR}
  if [[ $? -ne 0 ]]
  then
    echo "make dir ${HEADER_DIR} failed!"
    return
  fi

  #compile library
  #gcc -g -Wall -fPIC --shared ${SRC_FILE} -o ${DYN_FILE}

  gcc -g -Wall -c ${SRC_FILE}
  ar rcvs ${LIB_FILE} *.o

  if [[ ! -e ${LIB_FILE} ]]
  then
    echo "genrating library failed!"
    return
  fi

  #copy
  cp -f *.h ${HEADER_DIR}
  if [[ $? -ne 0 ]]
  then
    echo "install header files failed!"
  fi 
  
  #cp -f ${DYN_FILE} ${LIB_FILE} ${LIB_DIR}
  cp -f ${LIB_FILE} ${LIB_DIR}
  if [[ $? -ne 0 ]]
  then
    echo "install library failed!"
  fi

  #ln
  #cd ${LIB_DIR}
  #ln -s ${DYN_FILE} ${DYN_FILE}.0.1

  #clear
  cd ${WORK_DIR}
  rm ${LIB_FILE} *.o

  #finish
  echo "install library sucess!"

}

main
