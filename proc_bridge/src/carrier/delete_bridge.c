/*
 * delete_bridge.c
 *
 *  Created on: 2013-12-22
 *      Author: Administrator
 */

#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <getopt.h>
#include <errno.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include "proc_bridge.h"

extern int errno;
#define MY_LOG_NAME "delete_bridge.log"
static int slogd = -1; //slog
static char name_space[PROC_BRIDGE_NAME_SPACE_LEN] = {0};

static int show_help(void)
{
	printf("-i: proc id!\n");
	printf("-N:name_space\n");
	return 0;
}

int main(int argc , char **argv)
{
	int opt;
	int proc_id = 0;
	int shm_key;
	int shm_id = -1;

	bridge_hub_t *pbridge_hub = NULL;
	int ret;
	SLOG_OPTION slog_option;

	/***Open Log*/
	memset(&slog_option , 0 , sizeof(slog_option));
	strncpy(slog_option.type_value._local.log_name , MY_LOG_NAME , sizeof(slog_option.type_value._local.log_name));
	slogd = slog_open(SLT_LOCAL , SL_DEBUG , &slog_option , NULL);
	if(slogd < 0)
	{
		printf("open slog %s failed!\n" , MY_LOG_NAME);
		return -1;
	}

	/***Arg Check*/
	if(argc <= 0)
	{
		slog_log(slogd , SL_ERR , "Error,argc <= 0");
		return -1;
	}

	/*获取参数*/
	while((opt = getopt(argc , argv , "N:i:h")) != -1)
	{
		switch(opt)
		{
		case 'i':
			proc_id = atoi(optarg);
			break;
		case 'N':
			if(optarg)
				strncpy(name_space , optarg , sizeof(name_space));
			break;
		case 'h':
			show_help();
			return 0;
		}
	}
	if(proc_id <= 0)
	{
		slog_log(slogd , SL_ERR , "Error:Bad proc id arg!");
		return -1;
	}
	if(strlen(name_space) <= 0)
	{
		slog_log(slogd , SL_ERR , "Error:Del Shm Failed! name_space is NUll!");
		return -1;
	}
	slog_log(slogd , SL_INFO , "name_space:%s proc_id:%d" , name_space , proc_id);

	/*以PROC_ID为KEY取得共享内存*/
	shm_key = get_bridge_shm_key(name_space , proc_id , 0 , slogd);
	if(shm_key < 0)
	{
		slog_log(slogd , SL_ERR , "create shm failed for get_bridge_shm_key error.name_space:%s proc_id:%d" , name_space , proc_id);
		return -1;
	}
	//shm_key = SHM_CREATE_MAGIC + proc_id;

	//CREATE
	shm_id = shmget(shm_key , 0 , 0);
	if(shm_id < 0)
	{
		slog_log(slogd , SL_ERR , "Error: delete bridge,shmget bridge of %d  failed! err:%s" , proc_id , strerror(errno));
		return -1;
	}

	//attach
	pbridge_hub = (bridge_hub_t *)shmat(shm_id , NULL , 0);
	if(!pbridge_hub)
	{
		slog_log(slogd , SL_ERR , "Error: delete bridge,shmat bridge of %d failed! err:%s" , proc_id , strerror(errno));
		return -1;
	}

	slog_log(slogd , SL_INFO , "try to del bridge:%d attach:%d" , pbridge_hub->proc_id , pbridge_hub->attached);
	//delete shm
	ret = shmctl(shm_id , IPC_RMID , NULL);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "Error:delete bridge:%d failed! err:%s" , pbridge_hub->proc_id , strerror(errno));
	}
	else
	{
		slog_log(slogd , SL_INFO , "delete bridge:%d success!" , pbridge_hub->proc_id);
	}

	//return
	slog_close(slogd);
	return 0;
}
