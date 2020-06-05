#!/bin/bash

#arg list
operation=""
name_space=""
proc_name=""
proc_id=""

function show_help()
{
  echo "usage:./remote_tool.sh <opt> <name_space> <proc_name> <proc_id>"
}


function reload_cfg()
{
  echo "try to reload cfg of ${name_space}:${proc_name}"
  #reload cfg
  pid=`ps -Ao "pid","command" |grep carrier | grep "\<${name_space}\>" | grep "\<${proc_name}\>" | grep "\<${proc_id}\>" |grep -v 'grep' | grep -v 'bash' |awk '{print $1}'`
  if [[ -z ${pid} ]]
  then
    echo "carrier with ${proc_name} is not running"
  else
    echo "reload to ${pid}"
    kill -s SIGUSR1 ${pid}
  fi
}

function shut_carrier()
{
  pid=`cat "/tmp/.proc_bridge.${name_space}/carrier.${proc_id}.lock"`
  ret=`ps aux | grep carrier | grep ${name_space} | grep "\<${proc_name}\>" | grep ${pid} | grep -v grep | grep -v 'bash'`
  if [[ -z ${ret} ]]
  then
    echo "${proc_name} not runnable!"
    return
  fi

  #kill proc
  kill -s SIGINT ${pid}

  sleep 1
  #rm shm
  ./deleter -i ${proc_id} -N ${name_space}


  #check
  ret=`ps aux |grep carrier | grep ${name_space} | grep '\<${pid}\>' | grep -v grep`
  if [[ -z ${ret} ]]
  then
    echo "shut ${proc_name} success!"
  else
    echo "shut ${proc_name} failed!"
  fi 
}


function main
{
  echo "remote_tool exe..."
  if [[ $# -ne 4 ]]
  then
    echo "arg num error."
    show_help
    exit 1
  fi

  operation=$1
  name_space=$2
  proc_name=$3
  proc_id=$4
  echo "[${proc_name}:${proc_id}] in ${name_space}"

  #switch
  case ${operation} in
  1)
    reload_cfg   
  ;;
  2)
    shut_carrier
  ;;
  *)
    echo "fuck"
  ;;
  esac


}

function other()
{
  my_opt=$1

  #switch
  case $1 in
  101)
    echo "check_md5..."
    tar_file=$2
    md5file=$3
    md5_a=`md5sum ${tar_file} | awk '{print $1}'`
    md5_b=`cat ${md5file}`
    if [[ ${md5_a} != ${md5_b} ]]
    then
      echo "${tar_file} not matched!"
      exit 1
    else
      echo "md5 check pass"
    fi
  ;;
  *)
  ;;
  esac

}

if [[ $1 -gt 100 ]]
then
  other $@
else
  main $@
fi
