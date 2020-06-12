/*
 * carrier_lib.h
 *
 *  Created on: 2019年2月21日
 *      Author: nmsoccer
 */

#ifndef CARRIER_LIB_H_
#define CARRIER_LIB_H_

#include "proc_bridge_base.h"
#include "carrier_base.h"


#define TARGET_DATA_LEN(p) ((p->snd_head<=p->snd_tail)?(p->snd_tail-p->snd_head):(p->snd_buff_len-p->snd_head+p->snd_tail))
//empty space len (sub 1byte)
#define TARGET_EMPTY_SPACE(p) ((p->snd_head<=p->snd_tail)?(p->snd_buff_len-p->snd_tail+p->snd_head-1):(p->snd_head-p->snd_tail-1))
#define TARGET_IS_EMPTY(p) (p->snd_head==p->snd_tail)


/*
 * carrier_msg
 */
#define CR_MSG_MIN	     	1
#define CR_MSG_EVENT 	1	//事件类消息
#define CR_MSG_ERROR 	2	//错误类消息
#define CR_MSG_MAX		2

#define EVENT_PRINT_PREFFIX "[EVENT]"
#define ERROR_PRINT_PREFIX "[ERROR]"

#define MSG_EVENT_T_MIN			1
#define MSG_EVENT_T_START 	1	//进程拉起
#define MSG_EVENT_T_CONNECT_ALL			2	//连接OK
#define MSG_EVENT_T_RELOAD 	4	//重加载配置
#define MSG_EVENT_T_SHUTDOWN 5	//进程关闭
#define MSG_EVENT_T_CONNECTING 6	//连接中
#define MSG_EVENT_T_UPPER_RUNNING 7 //上层业务进程正常运行
#define MSG_EVENT_T_REPORT_STATISTICS 8	//报告数据
#define MSG_EVENT_T_MAX		8

#define MSG_ERR_T_MIN			1
#define MSG_ERR_T_START 		1	//进程拉起失败
#define MSG_ERR_T_RELOAD 	2	//重加载配置失败
#define MSG_ERR_T_CONNECT	3	//连接失败
#define MSG_ERR_T_LOST_CONN 4 //丢失连接
#define MSG_ERR_T_UPPER_LOSE 5 //上层业务进程丢失
#define MSG_ERR_T_MAX				5

//msg-event
typedef struct _msg_event_stat_t
{
	bridge_info_t bridge_info;	//bridge statistics
}msg_event_stat_t;
typedef struct _msg_event_t
{
	int type;	//refer MSG_EVENT_T_xx
	union
	{
		int value;
		long lvalue;
		proc_entry_t one_proc;
		msg_event_stat_t stat;
	}data;
}msg_event_t;

//msg-error
typedef struct _msg_error_t
{
	int type;	//refer MSG_ERR_T_xx
	union
	{
		proc_entry_t one_proc;	//one proc
	}data;
}msg_error_t;

typedef struct _carrier_msg_t
{
	int msg;	//refer CR_MSG_xx
	long ts;
	union
	{
		msg_event_t event;
		msg_error_t error;
	}data;
}carrier_msg_t;

/***INNER_PROTO*/
#define INNER_PROTO_MIN 1
#define INNER_PROTO_PING 1	//ping
#define INNER_PROTO_PONG 2 //pong
#define INNER_PROTO_VERIFY_REQ 3 //链接验证
#define INNER_PROTO_VERIFY_RSP  4
#define INNER_PROTO_TRAFFIC_REQ 5
#define INNER_PROTO_TRAFFIC_RSP 6
#define INNER_PROTO_LOG_DEGREE_REQ 7
#define INNER_PROTO_LOG_DEGREE_RSP 8
#define INNER_PROTO_LOG_LEVEL_REQ 9
#define INNER_PROTO_LOG_LEVEL_RSP 10
#define INNER_PROTO_MAX 10

typedef struct _inner_proto_t
{
	int type;	//refer INNER_PROTO_**
	char arg[MANAGER_CMD_ARG_LEN];
	union
	{
		char result;
		long long time_ms;
		char proc_name[PROC_ENTRY_NAME_LEN];
		char verify_key[BRIDGE_PROC_CONN_VERIFY_KEY_LEN];
		traffic_list_t traffic_list;
	}data;
}inner_proto_t;


/***********API***********/
extern char *format_time_stamp(long ts);
extern target_detail_t *proc_id2_target(carrier_env_t *penv , target_info_t *ptarget_info , int proc_id);
extern target_detail_t *fd_2_target(carrier_env_t *penv , int fd);
extern client_info_t *fd_2_client(carrier_env_t *penv , int fd);
extern int parse_proc_info(char *proc_info , proc_entry_t *pentry , int slogd);
extern int send_carrier_msg(carrier_env_t *penv , int msg , int type , void *arg1 , void *arg2);
extern int send_inner_proto(carrier_env_t *penv , target_detail_t *ptarget , int proto ,  void *arg1 , void*arg2);
extern int recv_inner_proto(carrier_env_t *penv , client_info_t *pclient , char *package);
extern int manager_handle(manager_info_t *pmanager , char *package , int slogd);
/*
 * 清空某个channel的target发送缓冲区
 * -1:错误
 *  0:成功
 */
extern int flush_target(carrier_env_t *penv , target_detail_t *ptarget);

/*
 * 直接发送数据[这种情况实在target缓冲区为空的情况下进行]
 * @stlv_buff:打包好的缓冲区
 * @stlv_len:包长
 * @return
 * -1:错误
 *  >=0:发送的字节数
 */
extern int direct_send(carrier_env_t *penv , target_detail_t *ptarget , char *stlv_buff , int stlv_len);
/*
 * 初始化manager的管理列表
 */
extern int init_manager_item_list(carrier_env_t *penv);
/*
 * 重建manager的管理列表
 * 用于在动态加载配置文件之后
 */
extern int rebuild_manager_item_list(carrier_env_t *penv);
/*
 * 打印当前报表
 */
extern int print_manage_info(carrier_env_t *penv);
extern manage_item_t *get_manage_item_by_id(carrier_env_t *penv , int proc_id);
extern int print_manage_item_list(int starts , manage_item_t *item_list , int count , FILE *fp);

/*
 * 处理来自manager tool 的包
 */
extern int handle_manager_cmd(carrier_env_t *penv , void *preq);
//投递网络包到本地recv共享内存
extern int append_recv_channel(bridge_hub_t *phub , char *pstpack , int slogd);

//生成校验key
extern int gen_verify_key(carrier_env_t *penv , char *key , int key_len);
//校验key
extern int do_verify_key(carrier_env_t *penv , char *key , int  key_len);
//扩展发送缓冲区
extern int expand_target_buff(carrier_env_t *penv , target_detail_t *ptarget);
//关闭target
extern int close_target_fd(carrier_env_t *penv , target_detail_t *ptarget , const char *reason , int epoll_fd , char del_from_epoll);
//正发送节点
extern int append_sending_node(carrier_env_t *penv , target_detail_t *ptarget);
extern int del_sending_node(carrier_env_t *penv , target_detail_t *ptarget);
//遍历sending node
extern int iter_sending_list(carrier_env_t *penv , char del);
extern int del_sending_list(carrier_env_t *penv);
//将数据放入target
extern int pkg_2_target(carrier_env_t *penv , target_detail_t *ptarget , char *pkg , int pkg_len);
extern int pkg_2_target_stlv(carrier_env_t *penv , target_detail_t *ptarget , char *stlv_buff , int stlv_len);

#endif /* CARRIER_LIB_H_*/
