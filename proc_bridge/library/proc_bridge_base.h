/*
 * proc_bridge_base.h
 *
 *  Created on: 2019年2月21日
 *      Author: nmsoccer
 */

#ifndef PROC_BRIDGE_PROC_BRIDGE_BASE_H_
#define PROC_BRIDGE_PROC_BRIDGE_BASE_H_

#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <slog/slog.h>

//一个bridge系统所能支撑的最多节点数目 实际应小于单机的MAX(port)=65535
//该字段也用于控制发送位图标记已标明是否某管道已满
#define MAX_PROC_BRIDGE_SCALE 80000

#define PROC_BRIDGE_NAME_SPACE_LEN 128	//name space len
#define BRIDGE_PROC_NAME_LEN 64 //proc_name len
#define BRIDGE_PROC_CONN_VERIFY_KEY_LEN 64	//链接验证码长度

#define PROC_BRIDGE_HIDDEN_DIR_FORMAT "/tmp/.proc_bridge.%s"
#define PROC_BRIDGE_HIDDEN_PID_FILE "carrier.%d.lock"
#define PROC_BRIDGE_HIDDEN_KEY_FILE "carrier.%d.key"

#define MANAGER_PROC_ID_MIN 1	//保留给manager的proc_id min
#define MANAGER_PROC_ID_MAX 1000	//保留给manager的proc_id max
#define SHM_CREATE_MAGIC	17908	/*创建共享内存时的魔数*/
#define BRIDGE_MODE_FLAG        S_IRWXU | S_IRWXG | S_IRWXO
#define CARRIER_PORT_ADD	 0 //10086	/*基于proc端口的增量*/
#define CARRIER_REAL_PORT(n) (n+CARRIER_PORT_ADD)	//carrier进程监听的实际端口(加上了偏移)

#define MANAGER_CMD_NAME_LEN (32)
#define MANAGER_CMD_ARG_LEN (128)

#define _TRACE_DEBUG_BUFF_LEN 2046 //宏_TRACE_DEBUG的缓冲区
/*##########DATA STRUCT##########*/
/******Bridge Package*/
#define BRIDGE_PACKAGE_DATA_LEN	(1024*100) //100k

#define BRIDGE_PKG_TYPE_NORMAL 0
#define BRIDGE_PKG_TYPE_CR_MSG 1
#define BRIDGE_PKG_TYPE_INNER_PROTO 2

struct _bridge_package_head
{
	int data_len;	/*数据区长度*/
	int sender_id;	/*发射者ID*/
	int recver_id;	/*接收者ID*/
	//long send_ts;	/*发送的时间戳*/
	long long send_ms; /*发送的时间戳(ms)*/
	char pkg_type;	/*包类型:refer BRIDGE_PKG_TYPE_xx*/
} __attribute__((packed));
typedef struct _bridge_package_head bridge_package_head_t;

struct _bridge_package
{
	bridge_package_head_t pack_head;
	char pack_data[0];
};
typedef struct _bridge_package bridge_package_t;

#define BRIDGE_PACK_HEAD_LEN (sizeof(bridge_package_head_t))
#define BRIDGE_PACK_LEN	(sizeof(bridge_package_head_t) + BRIDGE_PACKAGE_DATA_LEN)
#define GET_PACK_LEN(data_len) (sizeof(bridge_package_head_t) + data_len)

/******Bridge Hub*/
struct _bridge_hub
{
	char proc_name[BRIDGE_PROC_NAME_LEN];
	int proc_id;	/*使用的进程 proc_id*/
	int shm_id;
	short attached;	//使用的进程数目 正常为2个

	unsigned int send_buff_size;	/*发送缓冲区大小*/
	//char send_full;	/*发送区是否满 会有同步问题，取消该标记。队列full的情况下永远不会head==tail*/
	unsigned int sended_count;	/*发送过的包数目*/
	int sending_count;
	int  send_tail;	/*send队列尾，用于写数据*/
	int send_head;	/*send队列头，用于读数据*/

	unsigned int recv_buff_size;	/*接收缓冲区大小*/
	//char recv_full;	/*接收区是否满 取消标记*/
	unsigned int recved_count;	/*接受过的包数目*/
	int recving_count;
	int  recv_tail;	/*recv队列尾，用于写数据*/
	int recv_head;	/*recv队列头，用于读数据*/

	char snd_bitmap[MAX_PROC_BRIDGE_SCALE/8]; /*发送通道的位图 用于向上级进程标记是否该通道已满*/
	/*缓冲区起始地址*/
	char all_buff[0];
}__attribute__((packed));
typedef struct _bridge_hub bridge_hub_t;

#define CHANNEL_SAFE_AREA	(4)
#define GET_SEND_CHANNEL(p) (&p->all_buff[0])
#define GET_RECV_CHANNEL(p) (&p->all_buff[p->send_buff_size + CHANNEL_SAFE_AREA])


#ifdef __cplusplus
extern "C"{
#endif


#ifdef __cplusplus
}
#endif

#endif /*PROC_BRIDGE_PROC_BRIDGE_BASE_H_ */
