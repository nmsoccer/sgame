/*
 * manager_lib.h
 *
 *  Created on: 2019年3月8日
 *      Author: nmsoccer
 */

#ifndef PROC_BRIDGE_MANAGER_LIB_H_
#define PROC_BRIDGE_MANAGER_LIB_H_

#include "proc_bridge_base.h"
#include "carrier_lib.h"


#define MANAGER_STAT_MAX_ITEM 10

/*
 * PIPE PKG
 */
typedef struct _manager_pipe_pkg_t
{
	char cmd[MANAGER_CMD_NAME_LEN];
	//char arg[MANAGER_CMD_ARG_LEN];
	char arg1[PROC_ENTRY_NAME_LEN];
	char arg2[PROC_ENTRY_NAME_LEN];
}manager_pipe_pkg_t;


/*
 * Manage-Cmd
 */
#define MANAGER_CMD_MIN 1
#define MANAGER_CMD_STAT 1	// 检索信息
#define MANAGER_CMD_ERR 2	//检索错误
#define MANAGER_CMD_PROTO 3	//传递给其他服务器的内部协议
#define MANAGER_CMD_MAX 3

#define CMD_STAT_T_PART 1	//检索部分carrier状态
#define CMD_STAT_T_ALL 2 //检索全部carrier状态

#define CMD_ERR_T_CONN 1	//检索链接错误
#define CMD_ERR_T_SYS 2	//检索系统错误
#define CMD_ERR_T_ALL 3	//检索所有错误

#define CMD_PROTO_T_PING 1 //ping
#define CMD_PROTO_T_TRAFFIC 2 //conn-traffic
#define CMD_PROTO_T_LOG_DEGREE 3 //log-degree
#define CMD_PROTO_T_LOG_LEVEL 4 //log-level

	//STAT
typedef struct _cmd_stat_req_t
{
	short type;	//refer CMD_STAT_T_*
	char cmd[MANAGER_CMD_NAME_LEN];
	char arg[MANAGER_CMD_ARG_LEN];
}cmd_stat_req_t;
typedef struct _cmd_stat_rsp_t
{
	cmd_stat_req_t req;
	int seq;
	int total_count;
	long last_update;
	int count;
	manage_item_t item_list[MANAGER_STAT_MAX_ITEM];
}cmd_stat_rsp_t;

	//ERR
typedef struct _cmd_err_req_t
{
	short type;	//refer CMD_ERR_T_xx
	char cmd[MANAGER_CMD_NAME_LEN];
	char arg[MANAGER_CMD_ARG_LEN];
}cmd_err_req_t;
typedef struct _cmd_err_rsp_t
{
	cmd_err_req_t req;
	char type;
	int seq;
	int total_count;
	long last_update;
	int count;
	manage_item_t item_list[MANAGER_STAT_MAX_ITEM];
}cmd_err_rsp_t;

	//INNER_PROTO
typedef struct _cmd_proto_req_t
{
	short type;
	char value;
	//char arg[MANAGER_CMD_ARG_LEN];
	char arg1[PROC_ENTRY_NAME_LEN];
	char arg2[PROC_ENTRY_NAME_LEN];
}cmd_proto_req_t;
typedef struct _cmd_proto_rsp_t
{
	int value;
	short type;
	char result;
	//char arg[MANAGER_CMD_ARG_LEN];
	char arg1[PROC_ENTRY_NAME_LEN];
	char arg2[PROC_ENTRY_NAME_LEN];
	traffic_list_t traffic_list;
}cmd_proto_rsp_t;

	//MANAGER_CMD
typedef struct _manager_cmd_req_t
{
	short type;	//refer MANAGER_CMD_xx
	union
	{
		cmd_stat_req_t stat;
		cmd_err_req_t err;
		cmd_proto_req_t proto;
	}data;
}manager_cmd_req_t;

typedef struct _manager_cmd_rsp_t
{
	char manage_stat;
	short type; //refer MANAGER_CMD_xx
	union
	{
		cmd_stat_rsp_t stat;
		cmd_err_rsp_t err;
		cmd_proto_rsp_t proto;
	}data;
}manager_cmd_rsp_t;


#endif /* PROC_BRIDGE_MANAGER_LIB_H_ */
