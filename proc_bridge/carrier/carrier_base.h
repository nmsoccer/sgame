/*
 * carrier_base.h
 *
 *  Created on: 2019年3月25日
 *      Author: nmsoccer
 */

#ifndef _CARRIER_BASE_H_
#define _CARRIER_BASE_H_


//发送缓冲区上限为10M.因为其上游是内存下游是网络端，受到对端进程和网络性能有关，可能会造成包堆积，所以需要较大缓冲区
//该字段已经改进为carrier_env.max_expand_size根据发送总长度自动调整
//接收缓冲区上限为1M.比发送缓冲区小的原因是接收缓冲区的上游是网络下游是内存，上游应该远慢于下游处理速度，所以不易造成包堆积
//#define MAX_TARGET_BUFF_SIZE (10*1024*1024) //单个target可扩展到的最大缓冲区长度[可配置]
#define MAX_CLIENT_BUFF_SIZE (1*1024*1024) //单个client可扩展到的最大缓冲区长度
//#define SND_BIT_MAP_ALERT  (500*1024) //在通道可扩展缓冲区到达上限时，剩余空间不足该参数时进行send_bit_map置位(一般应大于package_len防止丢包)

#define MAX_CHECK_SUM_BYTES 64 //发送时最多进行checksum校验的字节数目

//设置socket的高一端缓存(recv的接收端 send的发送端)
//缓存的大小和包大小的收发效率有关 缓存越大(1M)对大包(10K-100K)表现越好 缓存较小对小包(100-1k)表现更好
//ps:缓存最好要大于2*PKG_LEN
#define CARRIER_SOCKET_BUFF_BIG (1024*512)

/*********DATA STRUCT*/
//获得m所在结构的地址 t:结构类型 m:m所在结构的成员名 m_a:m的地址
#define GET_STRUCT_ADDR(t , m , m_a) ((unsigned long)m_a - (unsigned long)&(((t *)0)->m))
/*
 * link_list
 */
typedef struct _link_list {
	struct _link_list *next;
}link_list_t;

/*
 * proc_entry
 */
#define PROC_ENTRY_NAME_LEN BRIDGE_PROC_NAME_LEN
#define PROC_ENTRY_IP_LEN	64
typedef struct _proc_entry_t
{
	int proc_id;
	char name[PROC_ENTRY_NAME_LEN];
	char ip_addr[PROC_ENTRY_IP_LEN];
	short port;
}proc_entry_t;

/*
 * time_ticker
 */
#define TIME_TICKER_T_SINGLE_SHOT 1
#define TIME_TICKER_T_CIRCLE 2

	//ms
#define TICK_CONNECT_TO_REMOTE 3500
#define TICK_MANAGE_PRINT 5000
#define TICK_CHECK_BRIDGE 3000
#define TICK_CHECK_CLIENT_INFO 4000
#define TICK_CHECK_RUN_STATISTICS 6500
#define TICK_CHECK_SIG_STAT 2000
#define TICK_ITER_SENDING_LIST 500//1500
#define TICK_CHECK_HASH_MAP 15000
#define TICK_CHECK_SND_BUFF_MEMROY 30000
#define TICK_CHECK_TMP_FILE (3600000 * 6) //6h

typedef int (* CARRIER_TICK) (void *arg); //return 0:single-shot; >0:next-expire-ms
typedef struct _time_ticker_t
{
	char type;	//1:single-short 2:circle
	long long tick_period;	//如果是周期性的[毫秒]
	long long expire_ms;	//过期时间(毫秒)
	char ticker_name[64];
	void *arg;
	CARRIER_TICK func;
	struct _time_ticker_t *prev;
	struct _time_ticker_t *next;
}time_ticker_t;
typedef struct _tick_list_t
{
	long long latest_expire_ms;	//最近的过期时间(毫秒)
	int count;
	time_ticker_t head;
}tick_list_t;

/*
 * channel_info
 */
typedef struct _half_bridge_info_t
{
	int total_size;
	int head;
	int tail;
	unsigned int handled;	//处理过的包数
	int handing;	//正在处理的包数

	int	min_pkg_size;	//最小包长
	int 	max_pkg_size;	//最大包长
	int ave_pkg_size;	//平均包长

	unsigned int dropped;	//丢弃包数目
	long latest_drop;	//最近一次丢包时间

	unsigned int reset_connect;	//重置链接次数
	long latest_reset;
}half_bridge_info_t;

typedef struct _bridge_info_t
{
	half_bridge_info_t send;
	half_bridge_info_t recv;
}bridge_info_t;

/*
 * 每链接的信息
 */
typedef struct _conn_traffic_t
{
	char type;	//0:send 1:recv only set on transfer
	unsigned int handled;	//处理过的包
	unsigned int handing; //正在处理的包
	int min_size;
	int max_size;
	int ave_size;	//平均包长
	unsigned int dropped; 	//丢包数
	long latest_drop;	//最近一次丢包
	unsigned int reset; //链接重置数
	long latest_reset;	//最近一次重置
	int buff_len;	//buff size
	int delay_time;//	包滞留于发送缓冲区的平均时间,如果有(ms)
	int delay_count;
	int buffering;	//正在缓冲区里的数据
	int max_buffered;	//缓冲区里滞留的最长数据
	int ave_buffer;	//平均的缓冲数据长度 通过定时采样
}conn_traffic_t;
#define MAX_CONN_TRAFFIC_PER_PKG 50

typedef struct _traffic_list_t
{
	int count;
	char owner[PROC_ENTRY_NAME_LEN];
	char names[MAX_CONN_TRAFFIC_PER_PKG][PROC_ENTRY_NAME_LEN];
	conn_traffic_t lists[MAX_CONN_TRAFFIC_PER_PKG];
}traffic_list_t;

/*
 * target info
 */
#define TARGET_CONN_NONE 0	//未连接
#define TARGET_CONN_DONE 1 //已连接
#define TARGET_CONN_PROC 2	//正在连接


typedef struct _target_detail
{
	char connected;	/*是否已经链接*/
	char target_name[PROC_ENTRY_NAME_LEN];
	char ip_addr[PROC_ENTRY_NAME_LEN];	/*目标IP*/
	short port;	/*目标端口*/
	int proc_id;	/*目标proc_id*/
	int fd;	/*使用的fd*/
	struct _target_detail *next;
	long long delay_starts_ms;	//缓冲区滞留数据的起始时间
	long latest_send_ts;	//最近一次完成数据发送的时间戳
	int latest_send_bytes; //最近一次完成发送的长度
	//int ready_count;		//缓冲区里待发送的包数目 只是粗略计数
	//int tail;	// tail of valid data in buff
	int max_tail;	//max resident data in history
	//char *buff;
	//int buff_len;
	conn_traffic_t traffic;
	char snd_block; //发送阻塞
	//循环缓冲区
	unsigned long snd_buff_len;	//缓冲区大小
	unsigned long snd_head; //缓冲区头
	unsigned long snd_tail;	//缓冲区尾
	char *snd_buff;
	char in_append; //是否正在投递
	link_list_t appending; //正在投递的链表
}target_detail_t;

typedef struct _target_
{
	int target_count;
	target_detail_t head;		//dummy head
}target_info_t;


/*
 * manager_info
 */
#define MANAGER_MSG_LOG "msg.log"
#define MANAGER_REPORT_LOG "report.log"

#define MANAGE_STAT_INIT 0
#define MANAGE_STAT_OK 1
#define MANAGE_STAT_BAD 2

#define MANAGE_UPPER_UNKNOWN 0
#define MANAGE_UPPER_RUNNING 1
#define MANAGE_UPPER_LOSE 2

	//为每个carrier维护一个item
#define MANAGE_ITEM_FLG_NONE 0
#define MANAGE_ITEM_FLG_VALID 1
#define MANAGE_ITEM_STAT_LEN 128
typedef struct _manage_item_t
{
	char flag;	//refer MANAGE_ITEM_FLG_xx
	char my_conn_stat;	//connected to remote status.refer TARGET_CONN_xx
	long latest_update;	//latest update time stamp
	proc_entry_t proc;
	struct
	{
#define REMOTE_CONNECT_ALL 1
#define REMOTE_CONNECTING 2
		char stat;	//1:connect-all 2:connecting
		long ts;
		union
		{
			struct
			{
				proc_entry_t proc;
			}connecting;
		}data;

	}conn_stat;
	struct
	{
		struct
		{
			long start_time;	//拉起时间
			long shut_time;	//关闭时间
		}power;
		struct
		{
			long reload_time;	//最近一次加载cfg
			int result;	//结果
		}reload_info;
		struct
		{
			long check_time;
			char running;	//0:no 1:running
		}upper_stat;
		struct
		{
			long check_time;
			bridge_info_t info;	//bridge_info
		}bridge_stat;
	}run_stat;
}manage_item_t;

typedef struct _manager_info_t
{
	struct _carrier_env_t *penv;
	int msg_slogd;	//msg log
	//int item_len;	//max item
	char stat;	//refer manage_stat_xx
	int item_count;	//item count
	manage_item_t *item_list;

}manager_info_t;

/*
 * client_info
 */
typedef struct _client_info
{
	int fd; //sock fd
	char *buff;
	int tail;
	char main_buff[1024*1024];
	char back_buff[1024*1024];
	//char main_buff[BRIDGE_PACK_LEN + 128];
	//char back_buff[BRIDGE_PACK_LEN + 128];
	struct
	{
		int buff_len;		//total buff len
		int tail;	//recving data len
		int max_tail;	//max in history
		char *buff;	//暂时缓存已接收但因为业务层shm缓冲区满而未投递成功的包
		long delay_starts;	//最早非空的时间戳
	}recv_buffer;
	char client_ip[64];
	unsigned short client_port;
	long connect_time;	//建立链接的时间
	char verify;	//是否经过验证
	int proc_id;	//set after verify
	char proc_name[PROC_ENTRY_NAME_LEN]; //set after verify
	conn_traffic_t traffic;
	struct _client_info *next;
}client_info_t;
typedef struct _client_list
{
	int total_count;
	client_info_t *list;
}client_list_t;

/*
 * sending_node
 * 缓冲区有待发送数据的target
 * 保留长度为target_count/5
 */
typedef struct _sending_node_t
{
	int proc_id;
	target_detail_t *ptarget;
	struct _sending_node_t *prev;
	struct _sending_node_t *next;
}sending_node_t;

typedef struct _sending_list_t
{
	int total;
	//int valid;
	link_list_t head;
	//sending_node_t head_node;
}sending_list_t;

/*
 * hash_map用来快速寻址目标的fd或者proc_id
 * [type+value]=primary key
 */
#define CR_HASH_ENTRY_T_FD  1 //类型为fd
#define CR_HASH_ENTRY_T_PROCID 2 //类型为proc_id
typedef struct _cr_hash_entry_t
{
	char type; //CR_HASH_ENTRY_T_XX
	unsigned int value; //fd or proc_id
	void *refer;
	struct _cr_hash_entry_t *next;
}cr_hash_entry_t;

typedef struct _cr_hash_map_t
{
	char flag; //1:inited
	int len;
	int entry_count;
	cr_hash_entry_t *plist;
}cr_hash_map_t;



/*
 * carrier_env
 */
typedef struct _carrier_env_t
{
	char proc_name[BRIDGE_PROC_NAME_LEN];
	char name_space[PROC_BRIDGE_NAME_SPACE_LEN];
	char lock_file_name[128];
	int lock_file_fd;
	char cfg_file_name[128];
	char key_file_name[128];
	int shm_key;
	char file_back_buff;	//在内存不足时进行文件缓存 理论上可以无限缓存数据
	int proc_id;
	int slogd;	//slog descriptor
	int epoll_fd;
	long started_ts;	//启动时间
	manager_info_t *pmanager;	//only for manager proc
	target_info_t *ptarget_info;
	client_list_t *pclient_list;
	bridge_hub_t *phub;
	bridge_info_t bridge_info;
	struct
	{
		char sig_exit;	//信号退出
		char sig_reload;	//信号重载
	}sig_map;
	tick_list_t tick_list;
	sending_list_t sending_list;
	cr_hash_map_t target_hash_map;
	cr_hash_map_t client_hash_map;
	unsigned int max_expand_size;	//单channel最大可扩充的发送缓冲区上限(2*phub->send_size)
	unsigned int block_snd_size; //阻塞单个channel继续发送的条件(phub->send_size) 这两个字段都是为了保证缓冲区充裕,防止丢包
}carrier_env_t;




/*****FUNC********/
//get curr ms
extern long long get_curr_ms();

//添加一个ticker
extern int append_carrier_ticker(carrier_env_t *penv , CARRIER_TICK func_tick , char type , long long tick_period ,
		char *ticker_name , void *arg);

//遍历ticker并执行
extern int iter_time_ticker(carrier_env_t *penv);

//定时回收多余内存
//+发送缓冲区
//+接收缓冲区
//extern int shrink_memory(void *arg);

#define CR_HASH_MAP_T_TARGET 1
#define CR_HASH_MAP_T_CLIENT 2
//target_hash_map
extern int init_target_hash_map(carrier_env_t *penv);
//client_hash_map
extern int check_client_hash_map(carrier_env_t *penv);
extern int check_target_hash_map(carrier_env_t *penv);

extern int clear_hash_map(carrier_env_t *penv , char type);
extern int insert_hash_map(carrier_env_t *penv , char type , char entry_type , unsigned int value , void *refer);
extern int del_from_hash_map(carrier_env_t *penv , char type , char entry_type , unsigned int value);
extern cr_hash_entry_t *fetch_hash_entry(carrier_env_t *penv , char type , char entry_type , unsigned int value);
extern int dump_hash_map(carrier_env_t *penv , char type);


#endif /* _CARRIER_BASE_H_ */
