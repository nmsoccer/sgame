#!/bin/bash
#<l/r> <proc_name> <center-ip> <center-port> <foot> <cmd> <task_name>
p_type=$1
proc_name=$2
center_ip=$3
center_port=$4
foot=$5
cmd=$6
task_name=$7
ts=`date +"%F %T"`
report_src="./report.c"
report_bin="./report"
log_dir="/tmp/spush/$task_name/$proc_name"
my_tool_dir="spush_${task_name}_${proc_name}_tools"
log="$log_dir/log"


mkdir -p $log_dir
if [[ ! -e $log_dir ]]
then
  echo "create $log_dir at $ts failed!" >> ../spush_fail
fi

echo -e "\n" >> $log
echo "-----------------------------" >> $log
echo ">>running on $ts" >> $log
echo "$0 $p_type $proc_name $center_ip $center_port $foot [$cmd] $task_name " >> $log

function check_md5() 
{
  tar_file=../$proc_name.tar.gz
  md5_file=../$proc_name.tar.gz.md5

  #check exit
  if [[ ! -e $tar_file || ! -e $md5_file ]]
  then
    echo "$tar_file or $md5_file not exist!" >> $log
    return 1
  fi
 
  #check md5
  my_md5=`md5sum $tar_file | awk '{print $1}'`
  file_md5=`cat $md5_file | awk '{print $1}'`
  if [[ $my_md5 != $file_md5 ]]
  then
    echo "$tar_file md5 not matched!" >> $log
    return 1
  fi

  return 0
}

function report()
{
  if [[ ! -e $report_src ]]
  then
    echo "$report_src not exist!" >> $log
    return 1
  fi

  gcc -g $report_src -o $report_bin
  if [[ ! -e $report_bin ]]
  then
    echo "$report_bin not exist!" >> $log
    return 1
  fi
  echo "try to run $report_bin" >> $log
  ./$report_bin -p $proc_name -s 1 -A $center_ip -P $center_port >> $log 
}

function clean_footprint()
{
  #cd ..
  #clear tar files
  rm ./$proc_name.tar.gz
  rm ./$proc_name.tar.gz.md5

  #clear dir
  rm -rf ./${my_tool_dir}/
}

function main()
{
  #check arg
  if [[ $p_type == "r" ]]
  then
    if [[ -z $center_ip || -z $center_port ]]
    then
      echo "report failed! dispatcher ip or port not set!" >> $log
      return 1
    fi
  fi


  check_md5
  if [[ 0 -ne $? ]]
  then
    return 1
  fi
  echo "check md5 success" >> $log

  report
  echo "deploy finish" >> $log

  cd ..
  #command $cmd 
  #nohup $cmd &

  if [[ $foot != "y" ]]
  then
    clean_footprint 
  fi

  #run cmd
  echo "run cmd:$cmd" >> $log
  real_cmd=`echo $cmd | awk '{print $1}'`
  if [[ -e $real_cmd ]]
  then
   #echo "chmod $real_cmd" >> $log
   chmod u+x $real_cmd
  fi
  command $cmd 1>/dev/null 2>/dev/null 

  return 0
}

main
