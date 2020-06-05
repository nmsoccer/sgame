#!/bin/bash
BIN_FILES=("manager" "creater" "deleter" "carrier")
TOOL_FILES=("bridge_build.py" "deploy_proc.sh" "remote_tool.sh" "scp.exp" "exe_cmd.exp")
CARRIER_DIR="../carrier"
TOOL_DIR=${CARRIER_DIR}/tools
TESTER="tester"

function install()
{
  killall ${TESTER} 

  echo "try to install..."
  curr_dir=`pwd`

  #compiling carrier 
  echo "compiling carrier..."
  cd ${CARRIER_DIR}
  chmod u+x ./make.sh
  ./make.sh clean 
  ./make.sh make  
  
  #mv 
  cd ${curr_dir}
  for file in ${BIN_FILES[@]}
  do
    mv ${CARRIER_DIR}/${file} .
    if [[ $? -ne 0 ]]
    then
      echo "mv ${file} failed!"
      exit 1
    fi
  done  

  #cp
  for file in ${TOOL_FILES[@]}
  do
   cp ${TOOL_DIR}/${file} .
   if [[ ! -e ${file} ]]
   then
     echo "${file} not exist!"
     exit 1 
   fi
  done

  chmod a+x deploy_proc.sh
  #create cfg
  python ./bridge_build.py -c
  sleep 1

  if [[ ! -e "./cfg" ]]
  then
    echo "create cfg failed!"
    exit 1
  fi

  #deploy carrier
  python ./bridge_build.py -a
  sleep 2
 
  #python ./bridge_build.py -c
  #sleep 2
  #python ./bridge_build.py -a

  #rm files
  #rm bridge_build.py
  #for file in ${BIN_FILES[@]}
  #do
  #  rm ${file}
  #done

  #finish
  echo "========================"
  echo "install complete!"
  echo "you may use './manager -i 1 -N <name_space>' to start manager to check info."
  echo "you may use './bridge_build.py xx' to restart/shutdown/reload carrier"
  echo "good luck"
  echo "========================"
}

function clean()
{
  #shutdown carrier
  python bridge_build.py -s
  sleep 2

  #clear files
  for file in ${BIN_FILES[@]}
  do
    if [[ -e ${file} ]]
    then
      rm ${file}
    fi
  done

  for file in ${TOOL_FILES[@]}
  do
    if [[ -e ${file} ]]
    then
      rm ${file}
    fi
  done

  if [[ -e "./cfg" ]]
  then
    rm -rf "./cfg"
  fi

  rm ./*.log*
  killall ${TESTER}
  rm ${TESTER}

  echo "========================"
  echo "clear done"
  echo "========================"
}

if [[ $1 == "install" ]]
then
  install
elif [[ $1 == "clear" ]]
then
  clean
else
  echo "usage $0 <install>|<clear>"
fi
