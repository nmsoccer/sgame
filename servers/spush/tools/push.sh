#!/bin/bash
#./push.sh <task_name> <proc-name> <target-host> <target-dir> <my_ip> <my_port> <user> <pass> <foot> <cmd>  
task_name=$1
proc_name=$2
t_host=$3
t_dir=$4
m_ip=$5
m_port=$6
t_user=$7
t_pass=$8
t_foot=$9
shift
t_cmd=$9
working_dir=`pwd`
peer_dir="spush_${task_name}_${proc_name}_tools"
peer_exe="peer_exe.sh"
report="report.c"

echo ">>push $task_name $proc_name $t_host $t_dir $t_user *** <$m_ip:$m_port> $t_foot [$t_cmd]"
#exit 0
function show_help()
{
  echo "usage $0 <task> <proc-name> <target-host> <target-dir> <my_ip> <my_port> [user] [pass] <foot> <cmd>"
}
#$1:tarfile $2 md5file 
function local_dispatch()
{
  tar_file=$1
  md5_file=$2

  #mkdir
  mkdir -p $t_dir
  if [[ 0 -ne $? ]] 
  then
    echo "create $t_dir failed! please check"
    return 1
  fi

  #cp
  cp $tar_file $md5_file $t_dir
  if [[ 0 -ne $? ]]
  then
    echo "copy $tar_file $md5_file to $t_dir failed"
    return 1
  fi

  #unzip
  cd $t_dir
  tar -zxvf $tar_file 1>/dev/null 

  #check md5
  cd $peer_dir/ 
  chmod u+x $peer_exe
  ./$peer_exe "l" $proc_name "127.0.0.1" $m_port $t_foot "$t_cmd" $task_name

  cd $working_dir
  echo "dispatch $1 to $t_dir success"
  return 0
}

function remote_dispatch() 
{
  tar_file=$1
  md5_file=$2
  
  
  #scp files
  chmod u+x $working_dir/tools/scp.exp
  echo `pwd`
  echo "$working_dir/tools/scp.exp $t_host $t_user $t_pass $tar_file  $t_dir"
  $working_dir/tools/scp.exp $t_host $t_user $t_pass $tar_file  $t_dir
  echo "$working_dir/tools/scp.exp $t_host $t_user $t_pass $md5_file  $t_dir"
  $working_dir/tools/scp.exp $t_host $t_user $t_pass $md5_file  $t_dir
 
  #unzip and exe peer
  chmod u+x $working_dir/tools/exe_cmd.exp
  remote_cmd="cd $t_dir;tar -zxvf $tar_file;cd $peer_dir;chmod u+x ./$peer_exe;./$peer_exe r $proc_name $m_ip $m_port $t_foot \"$t_cmd\" $task_name"
  $working_dir/tools/exe_cmd.exp $t_host $t_user $t_pass "$remote_cmd" 
 
  echo "done"
}


function main() 
{
  #arg
  if [[ -z $proc_name || -z $t_host || -z $t_dir ]]
  then
    show_help
    return 1
  fi



  #tar file
  cd ./pkg/$task_name/$proc_name/
  mkdir ./$peer_dir
  cp $working_dir/tools/peer_exe.sh ./$peer_dir 
  cp $working_dir/tools/$report ./$peer_dir 

  #tool file
  pkg_name=$proc_name.tar.gz
  rm $pkg_name
  tar -zcvf $pkg_name ./* 1>/dev/null
  if [[ 0 -ne $? ]]
  then
    echo "create tar $pkg_name failed!"
    return 1
  fi

  #md5
  md5_name=$pkg_name.md5
  md5sum $pkg_name > $md5_name
  #ls -lrt

  #choose copy cmd
  if [[ $t_host == "127.0.0.1" || $t_host == "localhost" ]]
  then
    echo "local dispatch"
    local_dispatch $pkg_name $md5_name
  else
    echo "remote dispatch"
    cp_cmd="scp"
    if [[ -z $t_user  || -z $t_pass ]]
    then
      echo "deploy_user or pass not defined"
      show_help
      return 1
    else
      remote_dispatch $pkg_name $md5_name
    fi
  fi

  #


}

main
