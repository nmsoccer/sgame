/*
 * create_bridge.c
 *
 *  Created on: 2013-12-22
 *      Author: Administrator
 *      为每个进程创建其发射通道
 */
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <getopt.h>

#include <sys/ipc.h>
#include <sys/shm.h>
#include <errno.h>
#include "proc_bridge.h"

extern int errno;

#define MY_LOG_NAME "create_bridge.log"
static int slogd = -1; //slog

static int show_help(void)
{
	printf("-r <recv size> recv_size of channel.(by max_pkg count)\n");
	printf("-s <send size> send_size of channel.(by max_pkg count)\n");
	printf("-i <proc id>\n");
	printf("-N <name space>\n");
	return 0;
}

static int recv_buff_size = 0;
static int send_buff_size = 0;
static char name_space[PROC_BRIDGE_NAME_SPACE_LEN] = {0};

int main(int argc , char **argv)
{
	int opt;
	int proc_id = 0;
	int shm_key;
	int value = 0;

	unsigned shm_size = 0;
	int shm_id = -1;
	bridge_hub_t *pbridge_hub = NULL;
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
		slog_log(slogd , SL_ERR , "argc <=0");
		return -1;
	}

	/*获取参数*/
	while((opt = getopt(argc , argv , "r:s:i:N:h")) != -1)
	{
		switch(opt)
		{
		case 's':
			value = atoi(optarg);
			if(value <= 0)
			{
				slog_log(slogd  , SL_ERR , "Error:Bad send_size:%d\n" , value);
				return -1;
			}
			slog_log(slogd , SL_INFO , "send:%d" , value);
			send_buff_size = value * BRIDGE_PACK_LEN;
			break;
		case 'r':
			value = atoi(optarg);
			if(value <= 0)
			{
				slog_log(slogd  , SL_ERR , "Error:Bad recv_size:%d\n" , value);
				return -1;
			}
			slog_log(slogd , SL_INFO , "recv:%d" , value);
			recv_buff_size = value * BRIDGE_PACK_LEN;
			break;
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
		slog_log(slogd , SL_ERR , "Error:Create Shm Failed! Bad proc id arg!");
		return -1;
	}
	if(send_buff_size<=0 || recv_buff_size<=0)
	{
		slog_log(slogd , SL_ERR , "Error:Create Shm Failed! Bad send buff:%d or recv buff:%d" , send_buff_size , recv_buff_size);
		return -1;
	}
	if(strlen(name_space) <= 0)
	{
		slog_log(slogd , SL_ERR , "Error:Create Shm Failed! name_space is NUll!");
		return -1;
	}

	slog_log(slogd , SL_INFO , "name_space:%s proc_id: %d send_size:%d recv_size:%d" , name_space , proc_id , send_buff_size , recv_buff_size);

	/*计算共享内存总大小*/
	shm_size = sizeof(bridge_hub_t) + recv_buff_size + send_buff_size + CHANNEL_SAFE_AREA;	//send_buff <--4B--> recv_buff

	/*以PROC_ID为KEY创建共享内存*/
	shm_key = get_bridge_shm_key(name_space , proc_id , 1 , slogd);
	if(shm_key < 0)
	{
		slog_log(slogd , SL_ERR , "create shm failed for get_bridge_shm_key error.name_space:%s proc_id:%s" , name_space , proc_id);
		return -1;
	}
	//shm_key = SHM_CREATE_MAGIC + proc_id;

	//CREATE
	shm_id = shmget(shm_key , shm_size , IPC_CREAT | IPC_EXCL | BRIDGE_MODE_FLAG);
	if(shm_id < 0)
	{
		slog_log(slogd , SL_ERR , "Error: shmget bridge of %d  failed! err:%s" , proc_id , strerror(errno));
		return -1;
	}

	//attach
	pbridge_hub = (bridge_hub_t *)shmat(shm_id , NULL , 0);
	if(!pbridge_hub)
	{
		slog_log(slogd , SL_ERR , "Error: shmat bridge of %d failed! err:%s" , proc_id , strerror(errno));
		return -1;
	}

	//init
	memset(pbridge_hub , 0 , sizeof(bridge_hub_t));
	pbridge_hub->proc_id = proc_id;
	pbridge_hub->shm_id = shm_id;
	pbridge_hub->send_buff_size = send_buff_size;
	pbridge_hub->recv_buff_size = recv_buff_size;
	slog_log(slogd , SL_INFO , "create bridge:%d success! recv_size:%ld send_size:%ld shm_id:%d" , pbridge_hub->proc_id , pbridge_hub->recv_buff_size ,
			pbridge_hub->send_buff_size , pbridge_hub->shm_id);

	//return
	slog_close(slogd);
	return 0;
}
