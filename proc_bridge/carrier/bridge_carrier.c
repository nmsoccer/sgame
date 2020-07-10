/*
 * bridge_carrier.c
 *
 * 每一个proc对应的守护进程
 * 负责从bridge里取proc发送的包到对应进程；
 * 从socket读发送给proc的数据装填到bridge
 *
 *  Created on: 2013-12-24[mainly updated on 2019-02-11]
 *      Author: nmsoccer
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>
#include <errno.h>
#include <stdarg.h>
#include <ctype.h>
#include <limits.h>
#include <sys/file.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <arpa/inet.h>
#include <sys/epoll.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <signal.h>
#include <time.h>
#include <unistd.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <stlv/stlv.h>
#include "proc_bridge.h"
#include "carrier_base.h"
#include "carrier_lib.h"
#include "manager_lib.h"

#define DEFAULT_CFG_FILE "./carrier.cfg"	//default cfg file

#define MY_LOG_NAME "carrier.log"
#define MY_SLOG_LEVEL SL_DEBUG
//#define INFO_NORMAL SL_DEBUG
//#define INFO_MAIN	SL_INFO
//#define INFO_ERR	SL_ERR
static int slogd = -1; //slog descriptor

//#define MAX_SEND_COUNT	10	//每tick最多发送的包数
#define MS_PER_TICK 100 //每tick毫秒数 100ms
#define MS_MAT_EPOLL_WAIT 10 //100ms的tick粒度太大，只允许epoll_wait这么久的时间(在全部空闲的情况下carrier以10ms的真实tick运行)
#define MAX_DISPATCH_BRIDGE_MS 50 //每tick最多花50毫秒来读取并发送bridge里的数据

#define MAX_EPOLL_QUEUE (1024*2)

#define LEN_TAG (sizeof(short))
#define LEN_LENGTH (sizeof(int))

extern int errno;

static target_info_t target_info;
static client_list_t client_list;

static struct epoll_event ep_event_list[MAX_EPOLL_QUEUE];

static carrier_env_t carrier_env;
static carrier_env_t *penv;

static void handle_signal(int sig);
//static int carrier_print_info(char type , void *file , ...);
static int  connect_to_remote(void *arg);
static void handle_connecting_fd(target_detail_t *ptarget , struct epoll_event *pevent);
static int handle_target_fd(target_detail_t *ptarget , struct epoll_event *pevent);
//static int close_target_fd(target_detail_t *ptarget , const char *reason , int epoll_fd , char del_from_epoll);
static int parse_target_list(char *target_list , target_detail_t *ptarget);
static int dispatch_bridge(int reward_ms);
static int show_help(void);
static int set_nonblock(int fd);
static int fetch_send_channel(carrier_env_t *penv , bridge_hub_t *phub , char *buff);
static int fetch_send_channel_stlv(carrier_env_t *penv , bridge_hub_t *phub , char *stlv_buff);
//static int append_recv_channel(bridge_hub_t *phub , char *buff);
static int read_client_socket(int socket , bridge_hub_t *phub);
static int free_client_info(client_info_t *pclient);
static int read_carrier_cfg(carrier_env_t *penv , char flag);
static int print_target_info(target_info_t *ptarget_info);
static int free_target_info(target_info_t *ptarget_info);
static int copy_target_info(target_info_t *pdst , target_info_t *psrc , char init);
static int del_one_target(target_info_t *ptarget_info , target_detail_t *ptarget);
static void demon_proc();
static int add_ticker(carrier_env_t *penv);
static int check_bridge(void *arg);
static int manage_tick_print(void *arg);
static int print_bridge_info(bridge_hub_t *phub);
static int set_sock_option(int sock_fd , int send_size , int recv_size , int no_delay);
static int check_client_info(void *arg);
static int check_run_statistics(void *arg);
static int check_signal_stat(void *arg);
static int check_hash(void *arg);
static int check_snd_buff_memory(void *arg);
static int check_tmp_file(void *arg);
static int recv_client_pkg(carrier_env_t *penv , client_info_t *pclient , bridge_package_t *pkg);
static int expand_recv_buff(carrier_env_t *penv , client_info_t *pclient);
static int flush_recving_buff(carrier_env_t *penv , client_info_t *pclient);

//static bridge_hub_t *phub = NULL;
//static int epoll_fd;

int main(int argc , char **argv)
{
	//long ticks = 0;
	int opt;
	short proc_port = 0;
	target_detail_t *ptarget = NULL;
	client_info_t *pclient = NULL;
	SLOG_OPTION slog_option;

	int listen_socket = -1;	/*监听socket*/
	int acc_socket = -1;	/*accept的socket*/
	int ret = -1;
	int i = 0;
	int value = 0;

	int lock_file_fd = -1;
	int handle_fd = -1;
	int active_fds = 0;

	struct sockaddr_in serv_addr;
	struct sockaddr_in cli_addr;
	socklen_t addr_len = 0;

	struct epoll_event ep_event;
	//struct epoll_event ep_event_list[MAX_EPOLL_QUEUE];
	char buff[1024] = {0};
	char is_target_sock = 0;
	//cost
	int cost_ms = 0;
	int reward_ms = 0;
	int net_idle = 0;
	int total_cost = 0;
	int epoll_cost = 0;
	long long start_ms = 0;
	long long end_ms = 0;

	/***Open Log*/
	memset(&slog_option , 0 , sizeof(slog_option));
	slog_option.log_degree = SLD_MILL;
	slog_option.log_size = 40*1024*1024;
	strncpy(slog_option.type_value._local.log_name , MY_LOG_NAME , sizeof(slog_option.type_value._local.log_name));
	slogd = slog_open(SLT_LOCAL , MY_SLOG_LEVEL , &slog_option , NULL);
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

	/***初始化*/
	memset(&target_info , 0 , sizeof(target_info_t));
	memset(&client_list , 0 , sizeof(client_list));

	memset(&carrier_env , 0 , sizeof(carrier_env));
	penv = &carrier_env;
	penv->ptarget_info = &target_info;
	penv->pclient_list = &client_list;
	penv->slogd = slogd;

	/*获取参数*/
	while((opt = getopt(argc , argv , "p:n:i:hS::N:")) != -1)
	{
		switch(opt)
		{
		case 'p':
			proc_port = atoi(optarg);
			break;
		case 'i':
			penv->proc_id = atoi(optarg);
			break;
		case 'N':
			strncpy(penv->name_space , optarg , sizeof(penv->name_space));
			break;
		case 'n':
			if(optarg)
				strncpy(penv->proc_name , optarg , sizeof(penv->proc_name));
			break;
		case 'S':
			if(optarg)
			{
				if(strlen(optarg) >= sizeof(penv->cfg_file_name))
				{
					slog_log(slogd , SL_ERR , "cfg path:%s is too long." , optarg);
					return 0;
				}

				strncpy(penv->cfg_file_name , optarg , sizeof(penv->cfg_file_name));
			}
			else
				strncpy(penv->cfg_file_name , DEFAULT_CFG_FILE , sizeof(penv->cfg_file_name));
			break;
		case'h':
		default:
			show_help();
			return 0;
		}
	}

	//参数检查
	if(penv->proc_id<=0)
	{
		slog_log(slogd , SL_ERR , "Error:proc id not set!");
		show_help();
		return -1;
	}
	if(proc_port<=0)
	{
		slog_log(slogd , SL_ERR , "Error:proc port not set!");
		show_help();
		return -1;
	}

	if(strlen(penv->name_space) <= 0)
	{
		slog_log(slogd , SL_ERR , "Error:name space is not set!");
		show_help();
		return -1;
	}

	/*
	if(!target_list || strlen(target_list)==0)
	{
		slog_log(slogd , SL_ERR , "Error:target list not set!");
		show_help();
		return -1;
	}*/
	slog_log(slogd , SL_INFO , "name_space:%s proc_id:%d , proc_port:%d , cfg_file:%s" , penv->name_space , penv->proc_id , proc_port , penv->cfg_file_name);

	//锁文件
	//snprintf(penv->lock_file_name , sizeof(penv->lock_file_name) , PROC_BRIDGE_HIDDEN_DIR_FORMAT"/carrier.%d.lock" ,  penv->name_space ,
	//		penv->proc_id);
	snprintf(penv->lock_file_name , sizeof(penv->lock_file_name) , PROC_BRIDGE_HIDDEN_DIR_FORMAT"/"PROC_BRIDGE_HIDDEN_PID_FILE ,  penv->name_space ,
			penv->proc_id);
	lock_file_fd = open(penv->lock_file_name , O_RDWR|O_CREAT , 0644);
	if(lock_file_fd < 0)
	{
		slog_log(slogd , SL_ERR , "open lock_file:%s failed! err:%s" , penv->lock_file_name , strerror(errno));
		return -1;
	}

		//try lock  file
	ret = flock(lock_file_fd , LOCK_EX|LOCK_NB);
	if(ret < 0)
	{
		if(errno == EWOULDBLOCK)
		{
			memset(buff , 0 , sizeof(buff));
			ret = read(lock_file_fd , buff , sizeof(buff));
			slog_log(slogd , SL_ERR , "carrier process PID:%d with id:%d is running. please kill it first!" , atoi(buff) , penv->proc_id);
		}
		else
		{
			slog_log(slogd , SL_ERR , "try to lock file %s failed! err:%s" , penv->lock_file_name , strerror(errno));
		}
		return 0;
	}
	penv->lock_file_fd = lock_file_fd;

	/*守护进程*/
	demon_proc();

	/*write pid to lock file*/
	memset(buff , 0 , sizeof(buff));
	snprintf(buff , sizeof(buff) , "%-10d" , getpid());
	write(lock_file_fd , buff , strlen(buff));
	slog_log(slogd , SL_INFO , "lock %s success!" , penv->lock_file_name);


	//读配置文件
	ret = read_carrier_cfg(penv , 0);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "read_cfg failed!");
		return -1;
	}

	//设置stlv
	STLV_SET_LOG(slogd);
	STLV_CHECK_SUM_SIZE(MAX_CHECK_SUM_BYTES);

	/***打开bridge*/
	ret = open_bridge(penv->name_space , penv->proc_id , slogd);
	penv->phub = bd2bridge(ret);
	//phub = attach_bridge(penv->name_space , penv->proc_id , slogd);
	if(penv->phub == NULL)
	{
		slog_log(slogd , SL_ERR , "Error: open bridge failed!");
		slog_close(slogd);
		return 0;
	}
	//penv->phub = phub;
	memset(penv->phub->proc_name , 0 , strlen(penv->phub->proc_name));
	strncpy(penv->phub->proc_name , penv->proc_name , sizeof(penv->phub->proc_name));
	memset(penv->phub->snd_bitmap , 0 , sizeof(penv->phub->snd_bitmap));	//重新拉起之后对控制位图清零
	penv->max_expand_size = penv->phub->send_buff_size * 2 + (1024*1024);
	penv->block_snd_size = penv->phub->send_buff_size + (1024*1024);
	slog_log(slogd, SL_INFO , "Main:open bridge success! max_expand_size:%ld block_size:%ld" , penv->max_expand_size , penv->block_snd_size);

    /*获得bridge的key文件及内容*/
	snprintf(penv->key_file_name , sizeof(penv->key_file_name) , PROC_BRIDGE_HIDDEN_DIR_FORMAT"/"PROC_BRIDGE_HIDDEN_KEY_FILE , penv->name_space , 
	    penv->proc_id);
    penv->shm_key = get_bridge_shm_key(penv->name_space , penv->proc_id , 0 , penv->slogd);
	if(penv->shm_key < 0)
	{
		slog_log(slogd , SL_ERR , "Error: get shmkey failed!");
		slog_close(slogd);
		return 0;
	}

		//print
	print_target_info(&target_info);

	/*reg signal*/
	signal(SIGTERM , handle_signal);
	signal(SIGINT , handle_signal);
	signal(SIGKILL , handle_signal);
	signal(SIGUSR1 , handle_signal);
	signal(SIGUSR2 , handle_signal);

	/***创建监听socket*/
	//socket
	listen_socket = socket(AF_INET , SOCK_STREAM , 0);
	addr_len = sizeof(struct sockaddr_in);
	if(listen_socket < 0)
	{
		slog_log(slogd , SL_ERR, "Error:bridge carrier:call socket failed!");
		return -1;
	}

	//bind and resuse addr
	value = 1;
	ret = setsockopt(listen_socket , SOL_SOCKET , SO_REUSEADDR , &value , sizeof(value));
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "set sockopt:resuse addr failed! err:%s" , strerror(errno));
		return -1;
	}

	memset(&serv_addr , 0 , sizeof(serv_addr));
	serv_addr.sin_family = AF_INET;
	serv_addr.sin_addr.s_addr = INADDR_ANY;
	serv_addr.sin_port = htons(CARRIER_REAL_PORT(proc_port));/*监听端口为proc_port + CARRIER_PORT_ADD*/
	if(bind(listen_socket , (struct sockaddr *)&serv_addr , addr_len) < 0)
	{
		slog_log(slogd , SL_ERR, "Error:bridge carrier:bind socket port %d failed! err:%s" , CARRIER_REAL_PORT(proc_port) , strerror(errno));
		return -1;
	}

	/*set epoll*/
	penv->epoll_fd = epoll_create(MAX_EPOLL_QUEUE);
	if(penv->epoll_fd < 0)
	{
		slog_log(slogd , SL_ERR , "Error:bridge carrier:create epoll failed!");
		close(listen_socket);
		return -1;
	}

	//put listen_socket into epoll
	ep_event.events = EPOLLIN;
	ep_event.data.fd = listen_socket;
	ret = epoll_ctl(penv->epoll_fd , EPOLL_CTL_ADD , listen_socket , &ep_event);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "Error:bridge carrier:add listen socket into epoll list failed!");
		close(listen_socket);
		return -1;
	}

	/*listen*/
	listen(listen_socket , 10);

	/*if Manager Init it*/
	if(penv->proc_id <= MANAGER_PROC_ID_MAX)
	{
		penv->pmanager = calloc(1 , sizeof(manager_info_t));
		if(!penv->pmanager)
		{
			slog_log(slogd , SL_ERR , "manager:%d starts failed! manager_info alloc error! err:%s" , penv->proc_id , strerror(errno));
			return -1;
		}
		//basic info
		penv->pmanager->stat = MANAGE_STAT_INIT;
		penv->pmanager->penv = penv;

		//msg log
		memset(&slog_option , 0 , sizeof(slog_option));
		slog_option.log_size = (20*1024*1024);
		strncpy(slog_option.type_value._local.log_name , MANAGER_MSG_LOG , sizeof(slog_option.type_value._local.log_name));
		penv->pmanager->msg_slogd = slog_open(SLT_LOCAL , SL_DEBUG , &slog_option , NULL);
		if(penv->pmanager->msg_slogd < 0)
		{
			slog_log(slogd , SL_ERR , "manager:%d starts failed! msg log:%s can not open!" , penv->proc_id , MANAGER_MSG_LOG);
			return -1;
		}

		//init item list
		if(init_manager_item_list(penv) < 0)
		{
			slog_log(slogd , SL_ERR , "manager:%d starts failed! init manger item_list error!" , __FUNCTION__);
			return -1;
		}

	}

	/***Append Ticker*/
	ret = add_ticker(penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "main:add ticker failed!");
		return -1;
	}

	/***Started*/
	penv->started_ts = time(NULL);
	srand(penv->started_ts);
	slog_log(slogd , SL_INFO , "bridge_carrier carry proc %d,run on %s:%d success!" , penv->proc_id , "127.0.0.1" , proc_port+CARRIER_PORT_ADD);
	/***Main Logic*/
	while(1)
	{
		cost_ms = 0;
		total_cost = 0;
		start_ms = get_curr_ms();

		//timer tick
		cost_ms = iter_time_ticker(penv);
		//slog_log(penv->slogd , SL_VERBOSE , "main:iter_ticker:cost:%ld" , cost_ms);
		total_cost += cost_ms;

		/*取包发送*/
		cost_ms = dispatch_bridge(reward_ms);
		total_cost += cost_ms;
		if(cost_ms > 0)
		{
			//slog_log(penv->slogd , SL_DEBUG , "main:dispatch_bridge:cost:%ld" , cost_ms);
		}
		/*如果取包消耗为0则进行-主动推送*/
		//if(total_cost >= 0)
		//{
			cost_ms = iter_sending_list(penv , 0);
			total_cost += cost_ms;
			if(cost_ms>0)
			{
				//slog_log(penv->slogd , SL_DEBUG , "main:trig iter sending:cost:%ld" , cost_ms);
			}
		//}

		/*epoll wait*/
		/*
		epoll_cost = MS_PER_TICK-total_cost;
		epoll_cost = epoll_cost>MS_MAT_EPOLL_WAIT?MS_MAT_EPOLL_WAIT:epoll_cost;	//实际上100ms的tick实在太长了，我们最多允许wait MS_MAT_EPOLL_WAITms
		epoll_cost = epoll_cost<0?1:epoll_cost;*/
		if(net_idle<=0)
			epoll_cost = 1;
		else
			epoll_cost = 1 + net_idle/1000; //每1s才会加1ms给epoll_cost
		active_fds = epoll_wait(penv->epoll_fd , ep_event_list , MAX_EPOLL_QUEUE , epoll_cost);
		if(active_fds < 0)
		{
			//slog_log(slogd , SL_VERBOSE , "epoll_wait err:%s" , strerror(errno));
			//continue;
		}

		/*handle each fd*/
		for(i=0; i<active_fds; i++)
		{
			handle_fd =  ep_event_list[i].data.fd;
			/*---------Listen Socket--------------*/
			if( handle_fd == listen_socket)
			{
				acc_socket = accept(listen_socket , (struct sockaddr *)&cli_addr , &addr_len);
				if(acc_socket < 0)
				{
					slog_log(slogd , SL_ERR , "Error:accept socket failed!");
					continue;
				}

				pclient = (client_info_t *)calloc(1 , sizeof(client_info_t));
				if(!pclient)
				{
					slog_log(slogd , SL_ERR , "error:alloc client info about %s:%d failed!err:%s" , inet_ntoa(cli_addr.sin_addr) , ntohs(cli_addr.sin_port) ,
							strerror(errno));
					continue;
				}

				//add this socket
				set_nonblock(acc_socket);	/*非阻塞*/
				//接收缓冲区需要扩大,发送缓冲区可以缩小因为不发送
				slog_log(slogd , SL_INFO , "<<<<<<<<<<<<<<<Main:Accept socket from %s:%d!}" , inet_ntoa(cli_addr.sin_addr) , ntohs(cli_addr.sin_port));
				set_sock_option(acc_socket , 2048 , CARRIER_SOCKET_BUFF_BIG , 0);
				ep_event.events = EPOLLIN | EPOLLET;
				ep_event.data.fd = acc_socket;
				ret = epoll_ctl(penv->epoll_fd , EPOLL_CTL_ADD , acc_socket , &ep_event);
				if(ret < 0)
				{
					slog_log(slogd , SL_ERR , "error:bridge_carrier:add acc socket into epoll list failed!");
					close(acc_socket);
					free(pclient);
					continue;
				}

				//append to clientlist
				pclient->fd = acc_socket;
				pclient->buff = pclient->main_buff;
				strncpy(pclient->client_ip , inet_ntoa(cli_addr.sin_addr) , sizeof(pclient->client_ip));
				pclient->client_port = ntohs(cli_addr.sin_port);
				pclient->connect_time = time(NULL);
				pclient->verify = 0;

				client_list.total_count++;
				pclient->next = client_list.list;
				client_list.list = pclient;
				slog_log(slogd , SL_INFO , "append socket %d[%s:%d] to client info success!" , acc_socket , pclient->client_ip , pclient->client_port);

				ret = insert_hash_map(penv , CR_HASH_MAP_T_CLIENT , CR_HASH_ENTRY_T_FD , acc_socket , pclient);
				if(ret < 0)
					slog_log(slogd , SL_ERR , "insert (%d:%d) 0x%X to hash map failed!" , CR_HASH_ENTRY_T_FD , acc_socket , pclient);
				else
					slog_log(slogd , SL_INFO , "insert (%d:%d) 0x%X to hash map success!" , CR_HASH_ENTRY_T_FD , acc_socket , pclient);
				continue;
			}

			/*---------Target Sock--------------*/
			ptarget = NULL;
			is_target_sock = 0;
			do
			{
				if(target_info.target_count <= 0)
					break;

				//search target fd
				/*
				ptarget = target_info.head.next;
				while(ptarget)
				{
					//找到正在连接状态的该fd
					if(ptarget->fd==handle_fd)
					{
						is_target_sock = 1;
						break;
					}
					ptarget = ptarget->next;
				}*/
				ptarget = fd_2_target(penv , handle_fd);

				//该fd非target fd
				//if(!is_target_sock)
				//	break;
				if(!ptarget)
					break;

				//handle
				if(ptarget->connected == TARGET_CONN_PROC)	//处理正在连接的sock
					handle_connecting_fd(ptarget , &ep_event_list[i]);
				else if(ptarget->connected == TARGET_CONN_DONE)	//处理已连接的sock
				{
					handle_target_fd(ptarget , &ep_event_list[i]);
				}
				else	//NONE?
				{
					slog_log(slogd , SL_FATAL , "target sock %d[%s:%d]in epoll but none status!" , ptarget->fd , ptarget->ip_addr , ptarget->port);
					close_target_fd(penv , ptarget , "main-epoll" , penv->epoll_fd , 1);
				}

			}while(0);

			//本fd是一个target的fd，不需要进一步处理了
			if(is_target_sock)
				continue;

			/*----------CLIENT SOCK---------------*/
			if(ep_event_list[i].events & EPOLLIN)
			{
				//read msg
				read_client_socket(handle_fd , penv->phub);
			}

		}

		//calc reward
		if(active_fds == 0)	//等待了一个完整wait时间，说明此段时间网络空闲，则奖励给下次dispatch多5ms
		{
			reward_ms += 5;
			reward_ms = reward_ms>=40?40:reward_ms; //奖励最多不超过40ms.所以理论上最多有90%的时间可以用来读取bridge并分发[MAX_DISPATCH_BRIDGE_MS+reward]
			net_idle += 1;	//网络空闲则等待多加1us最高不超MAX_EPOLL_WAIT 大概会在(sum(1,10)s)后回归到10ms的epoll_wait时间以适应交互式的通信
			net_idle = net_idle>10000?10000:net_idle; //10000 ~= MAX_EPOLL_WAIT/0.001
			if(net_idle%1000 == 0 && net_idle!=10000)
			{
				//slog_log(slogd , SL_VERBOSE , "net idle incresed to %d" , net_idle);
			}

		}
		else	//如果未等待则检查sleep时间,同时网络空闲恢复初始值(epoll_wait的等待时间恢复为忙等1ms)
		{
			reward_ms = 0;
			net_idle = 0;
			//slog_log(slogd , SL_VERBOSE , "net_idle reset!");
			end_ms = get_curr_ms();
			//slog_log(penv->slogd , SL_DEBUG , "main:handle recving:cost:%ld" , end_ms - (start_ms+total_cost));
			if((end_ms-start_ms) >= MS_PER_TICK)
			{

			}
			else
			{
				if(total_cost==0)
					usleep(1000);
				//usleep((end_ms-start_ms)*1000);
			}
		}
		//slog_log(penv->slogd , SL_VERBOSE , "main:tick:reward_ms:%d" , reward_ms);
	}

}

static void handle_signal(int sig)
{
	switch(sig)
	{
	case SIGTERM:
		//slog_log(slogd , SL_INFO , "Main:Recv TERM SIGNAL, exit...");
		//slog_close(slogd);
		//break;
	case SIGINT:
		//slog_log(slogd , SL_INFO , "Main:Recv INT SIGNAL, exit...");
		//slog_close(slogd);
		//break;
	case SIGKILL:
		slog_log(slogd , SL_INFO , "Main:Recv EXIT SIGNAL, exit...");
		send_carrier_msg(&carrier_env , CR_MSG_EVENT , MSG_EVENT_T_SHUTDOWN , NULL , NULL);
		//sleep(2);
		penv->sig_map.sig_exit = 1;
		break;
	case SIGUSR2:
		slog_log(slogd , SL_INFO , "Main:Recv USR2 SIGNAL, please check channel.info!");
		print_bridge_info(penv->phub);
		break;
	case SIGUSR1:
		slog_log(slogd , SL_INFO , "Recv USR1 SIGNAL, try to reload cfg later...");
		penv->sig_map.sig_reload = 1;
		/*
		ret = read_carrier_cfg(&carrier_env , 1);
		slog_log(slogd , SL_INFO , "-----------------------Rload Cfg Finished-------------------");
		send_carrier_msg(&carrier_env , CR_MSG_EVENT , MSG_EVENT_T_RELOAD , &ret , NULL);*/
		break;
	default:
		return;
	}

	//exit(0);
	return;
}

static int show_help(void)
{
	printf("-p <proc port>: listening port\n");
	printf("-h: show help\n");
	printf("-i <proc id>: proc_id in proc_bridge sys\n");
	//printf("-s: target list stores in arg. -t <target list> is needed\n");
	printf("-S [cfg path]: target list stores in cfg file. if cfg not specified,default path is %s" , DEFAULT_CFG_FILE);
	printf("-t <target list>: target list\n");
	printf("-n [proc name]: carrier supported process of proc_name");
	printf("-N <name space>: name space of such proc_bridge sys\n");
	return 0;
}


static int set_nonblock(int fd)
{
	int flag;

	flag = fcntl(fd ,  F_GETFL , 0);
	flag |= O_NONBLOCK;

	return fcntl(fd , F_SETFL , flag);
}


/*
 * fetch_send_channel
 * 从send_channel中获一个package
 * @phub:该进程打开的bridge
 * @buff:
 * @return:
 * -1：错误
 * -2：发送缓冲区空
 * >=0：成功 并返回读取的包长
 */
static int fetch_send_channel_stlv(carrier_env_t *penv , bridge_hub_t *phub , char *stlv_buff)
{
	bridge_package_t *pstpack;
	char *send_channel = NULL;
	int copyed = 0;

	int head_pos = 0;
	int tail_pos = 0;
	int channel_len;
	int pack_len;
	int data_len;
	int head_len = 0;
	int stlv_len = -1;
	char head_buff[BRIDGE_PACK_HEAD_LEN] = {0};
	char buff[BRIDGE_PACK_LEN + 64];
	/***Arg Check*/
	if(!phub ||!stlv_buff)
	{
		return -1;
	}

	head_pos = phub->send_head;
	tail_pos = phub->send_tail;
	/***接收*/
	//1.检查是否有数据
	if(head_pos == tail_pos)
	{
		return -2;
	}

	//获取发送区地址
	send_channel = GET_SEND_CHANNEL(phub);

	//other
	channel_len = phub->send_buff_size;
	pack_len = sizeof(bridge_package_t);

	//1.5 预读头部
	if((channel_len - head_pos) < pack_len)	/*余下不足头部*/
	{
		copyed = channel_len - head_pos;
		memcpy(head_buff , &send_channel[head_pos] , copyed);
		memcpy(&head_buff[copyed] , &send_channel[0] , pack_len-copyed);
	}
	else	/*余下可以放下下头部*/
	{
		memcpy(head_buff , &send_channel[head_pos] , pack_len);
	}
	pstpack = (bridge_package_t *)head_buff;
	data_len = pstpack->pack_head.data_len;
	pack_len += data_len;

	//Handle
	//1.除了[head->tail]不足一个包长的情况，其他情况都直接压缩到缓冲区里
	if((channel_len - head_pos) >= pack_len)
	{
		stlv_len = STLV_PACK_ARRAY(stlv_buff , &send_channel[head_pos] , pack_len);
		if(stlv_len == 0)
		{
			slog_log(penv->slogd , SL_ERR , "%s pack failed!" , __FUNCTION__);
			return -1;
		}
		head_pos += pack_len;
		head_pos %= channel_len;
	}
	else //2.不足一个包长要特殊处理
	{
		slog_log(penv->slogd , SL_DEBUG , "%s special pack! head_pos:%d channel_len:%d pack_len:%d" , __FUNCTION__ , head_pos , channel_len ,
				pack_len);
		head_len = sizeof(bridge_package_t);
		//2.先读取头部区
		if((channel_len - head_pos) < head_len)	/*余下不足头部*/
		{
			copyed = channel_len - head_pos;
			memcpy(buff , &send_channel[head_pos] , copyed);
			memcpy(&buff[copyed] , &send_channel[0] , head_len-copyed);
			head_pos = 0 + head_len - copyed;
		}
		else	/*余下可以放下下头部*/
		{
			memcpy(buff , &send_channel[head_pos] , head_len);
			head_pos += head_len;
			head_pos %= channel_len;
		}

		//3.获得头部后
		pstpack = (bridge_package_t *)buff;
		data_len = pstpack->pack_head.data_len;
		pack_len = head_len + data_len;

		//4.读取数据
		if((channel_len - head_pos) < data_len)	/*余下不足数据*/
		{
			copyed = channel_len - head_pos;
			memcpy(pstpack->pack_data , &send_channel[head_pos] , copyed);
			memcpy(&pstpack->pack_data[copyed] , &send_channel[0] , data_len-copyed);
			head_pos = 0 + data_len - copyed;
		}
		else	/*余下可以放下数据*/
		{
			memcpy(pstpack->pack_data, &send_channel[head_pos] , data_len);
			head_pos += data_len;
			head_pos %= channel_len;
		}

		//5.打包
		stlv_len = STLV_PACK_ARRAY(stlv_buff , buff , pack_len);
		if(stlv_len == 0)
		{
			slog_log(penv->slogd , SL_ERR , "%s 2 pack failed!" , __FUNCTION__);
			return -1;
		}
	}

	//4.在读完该内存之后再修改位置指针，因为write会比较head指针位置.否则会出现同步错误
	phub->sending_count--;
	phub->send_head = head_pos;

	return stlv_len;
}



/*
 * fetch_send_channel
 * 从send_channel中获一个package
 * @phub:该进程打开的bridge
 * @buff:
 * @return:
 * -1：错误
 * -2：发送缓冲区空
 * >=0：成功 并返回读取的包长
 */
static int fetch_send_channel(carrier_env_t *penv , bridge_hub_t *phub , char *buff)
{
	bridge_package_t *pstpack;
	char *send_channel = NULL;
	//int empty_space = 0;
	//int send_count;
	int copyed = 0;

	int head_pos = 0;
	int tail_pos = 0;
	int channel_len;
	int pack_len;
	int data_len;

#ifdef _TRACE_DEBUG
	char test_buff[_TRACE_DEBUG_BUFF_LEN] = {0};
	int result = 0;
#endif

	/***Arg Check*/
	if(!phub ||!buff)
	{
		return -1;
	}

	head_pos = phub->send_head;
	tail_pos = phub->send_tail;
	/***接收*/
	//1.检查是否有数据
	//if(phub->send_full == 0 && phub->send_head==phub->send_tail)
	if(head_pos == tail_pos)
	{
		return -2;
	}

	//2.获取发送区地址
	send_channel = GET_SEND_CHANNEL(phub);

	//other
	channel_len = phub->send_buff_size;
	pack_len = sizeof(bridge_package_t);
	//head_pos = phub->send_head;

#ifdef _TRACE_DEBUG
	slog_log(slogd , SL_DEBUG , "%s 1 [%d<-->%d]<%d>" , __FUNCTION__ , head_pos , tail_pos , channel_len);
#endif

	//2.先读取头部区
	if((channel_len - head_pos) < pack_len)	/*余下不足头部*/
	{
		copyed = channel_len - head_pos;
		memcpy(buff , &send_channel[head_pos] , copyed);
		memcpy(&buff[copyed] , &send_channel[0] , pack_len-copyed);
		head_pos = 0 + pack_len - copyed;
	}
	else	/*余下可以放下下头部*/
	{
		memcpy(buff , &send_channel[head_pos] , pack_len);
		head_pos += pack_len;
		head_pos %= channel_len;
	}

	//3.获得头部后
	pstpack = (bridge_package_t *)buff;
	data_len = pstpack->pack_head.data_len;

	//4.读取数据
	if((channel_len - head_pos) < data_len)	/*余下不足数据*/
	{
		copyed = channel_len - head_pos;
		memcpy(pstpack->pack_data , &send_channel[head_pos] , copyed);
		memcpy(&pstpack->pack_data[copyed] , &send_channel[0] , data_len-copyed);
		head_pos = 0 + data_len - copyed;
	}
	else	/*余下可以放下数据*/
	{
		memcpy(pstpack->pack_data, &send_channel[head_pos] , data_len);
		head_pos += data_len;
		head_pos %= channel_len;
	}


	//4.在读完该内存之后再修改位置指针，因为write会比较head指针位置.否则会出现同步错误
	phub->sending_count--;
	phub->send_head = head_pos;

#ifdef _TRACE_DEBUG
	result = sizeof(bridge_package_head_t) + data_len;
	memcpy(test_buff , pstpack->pack_data , data_len);
	slog_log(slogd , SL_DEBUG , "%s 2 [%d<-->%d](%d:%d)(%d:%d)%s" , __FUNCTION__, head_pos , tail_pos , head_pos , phub->send_head ,
			data_len , result , test_buff);
#endif
	//phub->send_full = 0;

	return (sizeof(bridge_package_head_t) + data_len);
}

/*
 * try_fetch_send_head
 * 从send_channel中获一个package的头部 但并不实质拷贝
 * @phub:该进程打开的bridge
 * @buff:
 * @return:
 * -1：错误
 * -2：发送缓冲区空
 * >=0：成功 并返回读取的长度
 */
static int try_fetch_send_head(bridge_hub_t *phub , bridge_package_t *buff)
{
	char *send_channel = NULL;
	int copyed = 0;

	int head_pos = 0;
	int tail_pos = 0;
	int channel_len;
	int pack_len;

	/***Arg Check*/
	if(!phub ||!buff)
	{
		return -1;
	}

	head_pos = phub->send_head;
	tail_pos = phub->send_tail;
	/***接收*/
	//1.检查是否有数据
	if(head_pos == tail_pos)
	{
		return -2;
	}

	//2.获取发送区地址
	send_channel = GET_SEND_CHANNEL(phub);

	//other
	channel_len = phub->send_buff_size;
	pack_len = sizeof(bridge_package_t);

	//2.读取头部区
	if((channel_len - head_pos) < pack_len)	/*余下不足头部*/
	{
		copyed = channel_len - head_pos;
		memcpy(buff , &send_channel[head_pos] , copyed);
		memcpy(&buff[copyed] , &send_channel[0] , pack_len-copyed);
		head_pos = 0 + pack_len - copyed;
	}
	else	/*余下可以放下下头部*/
	{
		memcpy(buff , &send_channel[head_pos] , pack_len);
		head_pos += pack_len;
		head_pos %= channel_len;
	}

	return pack_len;
}

/*
 * 将target_list的内容解析到target_info里
 * target_list:
 * [@proc_name&proc_id&ip_addr&port]
 */
static int parse_target_list(char *target_list , target_detail_t *ptarget)
{
	//target_detail_t *ptarget = NULL;
	proc_entry_t proc_entry;
	int ret = -1;

	/***Arg Check*/
	if(!target_list || strlen(target_list)<=0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! target_list null!" , __FUNCTION__);
		return -1;
	}
	if(!ptarget)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! target_info null!" , __FUNCTION__);
		return -1;
	}
	if(target_list[0] != '@')	//必须以@开头
	{
		slog_log(slogd , SL_ERR , "<%s> failed! illegal target_list:%s without '@' at first!" , __FUNCTION__ , target_list);
		return -1;
	}

	/***Init*/
	memset(&proc_entry , 0 , sizeof(proc_entry_t));

	/**Handle*/
	ret = parse_proc_info(target_list , &proc_entry , slogd);
	if(ret < 0)
		return -1;

	/***COPY*/
	strncpy(ptarget->target_name , proc_entry.name , PROC_ENTRY_NAME_LEN);
	ptarget->proc_id = proc_entry.proc_id;
	strncpy(ptarget->ip_addr , proc_entry.ip_addr , PROC_ENTRY_IP_LEN);
	ptarget->port = proc_entry.port;

	return 0;
}

/*
 * 链接到远程carrier
 */
static int  connect_to_remote(void *arg)
{
	struct sockaddr_in remote_addr;
	struct epoll_event ep_event;
	proc_entry_t proc_entry;
	target_detail_t *ptarget = NULL;
	int remote_socket;
	int count = 0;
	int ret = -1;
	long curr_ts = 0;
	int connected = 0;
	static char need_report = 0;
	manage_item_t *pitem = NULL;
	carrier_env_t *penv = (carrier_env_t *)arg;

	//check circle
	if(!need_report)
	{
		curr_ts = time(NULL);
		if((curr_ts-carrier_env.started_ts) >= 20)	//进程拉起前20秒不用上报，等待路由建立
		{
			send_carrier_msg(&carrier_env , CR_MSG_EVENT , MSG_EVENT_T_START , &(carrier_env.started_ts) , NULL);
			need_report = 1;
		}
	}

	if(target_info.target_count <= 0)
		return 0;

	/*与每一个远程proc的bridge_carrier建立链接*/
	ptarget = target_info.head.next;
	for(;ptarget;ptarget=ptarget->next)
	{
		//检查各项状态
			//已连接
		if(ptarget->connected == TARGET_CONN_DONE)
			connected++;
			//未连接&连接中
		if((ptarget->connected==TARGET_CONN_NONE ||ptarget->connected==TARGET_CONN_PROC) && need_report)
		{
			memset(&proc_entry , 0 , sizeof(proc_entry));
			strncpy(proc_entry.name , ptarget->target_name , sizeof(proc_entry.name));
			strncpy(proc_entry.ip_addr , ptarget->ip_addr , sizeof(proc_entry.ip_addr));
			proc_entry.port = ptarget->port;
			proc_entry.proc_id = ptarget->proc_id;
			send_carrier_msg(&carrier_env , CR_MSG_EVENT , MSG_EVENT_T_CONNECTING , &proc_entry , NULL);
		}



		//0.只有无连接状态才继续
		if(ptarget->connected != TARGET_CONN_NONE)
		{
			continue;
		}

		//1.create socket
		remote_socket = socket(AF_INET , SOCK_STREAM , 0);
		if(remote_socket < 0)
		{
			slog_log(slogd , SL_ERR, "<%s> Error:bridge carrier:create remote socket to proc %d[%s:%d] failed!}" , __FUNCTION__ , ptarget->proc_id ,
					ptarget->ip_addr , ptarget->port);
			continue;
		}

		//2.non block
		set_nonblock(remote_socket);

		//3.connect
		bzero(&remote_addr , sizeof(struct sockaddr_in));
		remote_addr.sin_family = AF_INET;
		remote_addr.sin_port = htons(CARRIER_REAL_PORT(ptarget->port));	/*实际上链接的是对端proc的carrier*/
		inet_pton(AF_INET , ptarget->ip_addr , &remote_addr.sin_addr);
		if(connect(remote_socket , (struct sockaddr *)&remote_addr , sizeof(struct sockaddr_in)) < 0)
		{
			if(errno != EINPROGRESS)	//not in progress
			{
				slog_log(slogd , SL_ERR , "<%s> Error:connect to proc:%d,%s:%d failed! err:%s" , __FUNCTION__ , ptarget->proc_id ,
						ptarget->ip_addr , ptarget->port+CARRIER_PORT_ADD , strerror(errno));

				//close
				close(remote_socket);
				ptarget->connected = TARGET_CONN_NONE;
			}
			else	// in progress
			{
				slog_log(slogd , SL_INFO , "<%s> connect to proc:%d,%d[%s:%d] in progress" , __FUNCTION__ , ptarget->proc_id , remote_socket ,
						ptarget->ip_addr , ptarget->port+CARRIER_PORT_ADD);

				//.epoll-ctrl
				memset(&ep_event , 0 , sizeof(ep_event));
				ep_event.events = EPOLLIN | EPOLLOUT | EPOLLET;	//read-write and edge trigger
				ep_event.data.fd = remote_socket;
				ret = epoll_ctl(penv->epoll_fd , EPOLL_CTL_ADD , remote_socket , &ep_event);
				if(ret < 0)
				{
					slog_log(slogd , SL_ERR , "%s Error:add remote socket %d[%s:%d] to epoll list failed!" , __FUNCTION__ , remote_socket ,
							ptarget->ip_addr , ptarget->port);

					close(remote_socket);
					ptarget->connected = TARGET_CONN_NONE;
				}
				else
				{
					//check
					//target_info.epoll_manage_in_connect++;
					ptarget->fd = remote_socket;
					ptarget->connected = TARGET_CONN_PROC;
					insert_hash_map(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_FD , ptarget->fd , ptarget);
				}
			}
			continue;
		}

		//connect success
		slog_log(slogd , SL_INFO , ">>>>>>>>>>>Connect to proc:%d,%s:%d success!" , ptarget->proc_id ,
				ptarget->ip_addr , ptarget->port+CARRIER_PORT_ADD);

		memset(&ep_event , 0 , sizeof(ep_event));
		ep_event.events = EPOLLIN | EPOLLOUT | EPOLLET;	//read-write and edge trigger
		ep_event.data.fd = remote_socket;
		ret = epoll_ctl(penv->epoll_fd , EPOLL_CTL_ADD , remote_socket , &ep_event);
		if(ret < 0)
		{
			slog_log(slogd , SL_ERR , "%s Error:add remote socket %d[%s:%d] to epoll list failed!" , __FUNCTION__ , remote_socket ,
					ptarget->ip_addr , ptarget->port);

			close(remote_socket);
			ptarget->connected = TARGET_CONN_NONE;
			continue;
		}


		ptarget->connected = TARGET_CONN_DONE;
		ptarget->fd = remote_socket;
		ptarget->snd_head = ptarget->snd_tail = ptarget->snd_buff_len = 0;
		ptarget->snd_buff = NULL;
		insert_hash_map(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_FD , ptarget->fd , ptarget);
		//mange only
		pitem = get_manage_item_by_id(penv , ptarget->proc_id);
		if(pitem)
		{
			pitem->latest_update = time(NULL);
			pitem->my_conn_stat = ptarget->connected;
		}

		set_sock_option(remote_socket , CARRIER_SOCKET_BUFF_BIG , 1024*2 , 1);
		count++;

	}

	if(count > 0)
		slog_log(slogd , SL_INFO , "This Time Connected %d Success!" , count);

	if(connected == target_info.target_count && need_report)
		send_carrier_msg(&carrier_env , CR_MSG_EVENT , MSG_EVENT_T_CONNECT_ALL , NULL , NULL);

	return 0;
}


/*
 * 读proc发送的包到相应的fd
 * reward_ms:奖励给该函数的毫秒数.它与上一次tick有关
 */
static int dispatch_bridge(int reward_ms)
{
	bridge_package_t *pstpack;
	target_detail_t *ptarget = NULL;
	conn_traffic_t *ptraffic = NULL;
	//char stlv_pack[BRIDGE_PACK_LEN*2];
	//int stlv_len = 0;

	char buff[BRIDGE_PACK_LEN+64];
	int bridge_pack_len = 0;
	int ret = 0;
	bridge_info_t *pbridge_info = &penv->bridge_info;
	long curr_ts = 0;
	long long enter_ms = get_curr_ms();
	long long curr_ms = 0;
	long long end_ms = 0;
	int size_door = 10*1024;
#ifdef _TRACE_DEBUG
	char test_buff[_TRACE_DEBUG_BUFF_LEN] = {0};
#endif

	while(1)
	{
		//check time cost
		curr_ms = get_curr_ms();
		if((curr_ms - enter_ms) >= MAX_DISPATCH_BRIDGE_MS+reward_ms)	//超出每tick最大限额
			break;

		curr_ts = curr_ms/1000;
		/**Fetch a Pack*/
		//stlv_len = fetch_send_channel_stlv(penv , penv->phub , stlv_pack);
		bridge_pack_len = fetch_send_channel(penv , penv->phub , buff);
		if(bridge_pack_len == -1)
		{
			slog_log(slogd , SL_ERR , "%s fetch package failed for err!" , __FUNCTION__);
			break;
		}
		else if(bridge_pack_len == -2)
		{
			//slog_log(slogd , SL_DEBUG , "%s fetch failed for empty buff!" , __FUNCTION__);
			// try flush
			break;
		}
		else if(bridge_pack_len == 0)
		{
			slog_log(slogd , SL_ERR , "%s fetch an zero package!" , __FUNCTION__);
			break;
		}
		//bridge_pack_len = STLV_VALUE_INFO(stlv_pack , (char **)&pstpack);
		pstpack = (bridge_package_t *)buff;


		//slog_log(slogd , SL_DEBUG , "%s fetch pack success! len:%d and target:%d ts:%lld content:%s" , __FUNCTION__ , bridge_pack_len , pstpack->pack_head.recver_id ,
		//		pstpack->pack_head.send_ms , test_buff);
		//manager获得的包都源于manager_tool 一般不作转发
		if(penv->proc_id <= MANAGER_PROC_ID_MAX)
		{
			slog_log(slogd , SL_DEBUG , "<%s> this is a manager pkg" , __FUNCTION__);
			if(pstpack->pack_head.recver_id == penv->proc_id)
			{
				if(pstpack->pack_head.data_len == sizeof(manager_cmd_req_t))
					handle_manager_cmd(penv , &pstpack->pack_data[0]);
				else
				{
					slog_log(penv->slogd , SL_ERR , "<%s> manager handle failed! data len:%d not equal to cmd_req:%d. drop it!!" , __FUNCTION__ ,
												pstpack->pack_head.data_len , sizeof(manager_cmd_req_t));
				}

				continue;
			}
		}

		/***Get Target Info*/
		ptarget = proc_id2_target(penv , &target_info , pstpack->pack_head.recver_id);
		/*
		ptarget = target_info.head.next;		
		while(ptarget)
		{
			if(ptarget->proc_id == pstpack->pack_head.recver_id)
				break;
			ptarget = ptarget->next;
		}*/
		if(!ptarget)
		{
			slog_log(slogd , SL_ERR , "%s failed! target %d not found!" , __FUNCTION__ , pstpack->pack_head.recver_id);
			pbridge_info->send.dropped++;
			pbridge_info->send.latest_drop = curr_ts;
			continue;
		}

		ptraffic = &ptarget->traffic;
		/***Check Target Stat*/
		if(ptarget->connected != TARGET_CONN_DONE)
		{
			slog_log(slogd , SL_ERR , "%s failed! connection is lost. target:%d" , __FUNCTION__ , ptarget->proc_id);
			pbridge_info->send.dropped++;
			pbridge_info->send.latest_drop = curr_ts;
			ptraffic->dropped++;
			ptraffic->latest_drop = curr_ts;
			continue;
		}

		/***Init*/
		if(!ptarget->snd_buff || ptarget->snd_buff_len==0)
		{
			ret = expand_target_buff(penv , ptarget);
			if(ret < 0)
			{
				slog_log(slogd , SL_ERR , "%s init send_buff to [%s:%d] failed!" , __FUNCTION__ , ptarget->target_name , ptarget->proc_id);
				pbridge_info->send.dropped++;
				pbridge_info->send.latest_drop = curr_ts;
				ptraffic->dropped++;
				ptraffic->latest_drop = curr_ts;
				continue;
			}
		}

#ifdef _TRACE_DEBUG
		memcpy(test_buff , &bridge_pack[sizeof(bridge_package_head_t)] , pstpack->pack_head.data_len);
		test_buff[pstpack->pack_head.data_len] = 0;
		slog_log(slogd , SL_DEBUG , "%s fetch pack success! len:%d and target:%d ts:%lld pre-seq:%d content:%s" , __FUNCTION__ , bridge_pack_len , pstpack->pack_head.recver_id ,
				pstpack->pack_head.send_ms , ptraffic->handled , test_buff);
#endif

		/***Flush Target*/
		//if(!TARGET_IS_EMPTY(ptarget))
		size_door = (ptraffic->ave_size>0)?ptraffic->ave_size*2:size_door;
		//if(TARGET_DATA_LEN(ptarget)>=size_door)
		if(TARGET_DATA_LEN(ptarget)>=BRIDGE_PACK_LEN)
		{
			slog_log(slogd , SL_DEBUG , "%s is sending remaining package to %d data_len:%lu" , __FUNCTION__ , ptarget->proc_id ,TARGET_DATA_LEN(ptarget));
			ret = flush_target(penv , ptarget);
			switch(ret)
			{
			case -1:	//出现网络故障，则重置链接
				close_target_fd(penv , ptarget , __FUNCTION__ , penv->epoll_fd , 1);
				continue;
			break;
			case 0:	//未发送数据或发送部分数据 需要加入sending_list
			case 2:
				append_sending_node(penv , ptarget);
			break;
			case 1:	//全部发送则不管了
			break;
			default:
			break;
			}
		}

		/***Send Current Pack*/
		if(!TARGET_IS_EMPTY(ptarget))	//如果缓冲区未空，则说明当前不能发送，在STLV包之后投入缓冲区
		{
			slog_log(slogd , SL_DEBUG , "%s target not empty! append directly!" , __FUNCTION__);
			append_sending_node(penv , ptarget);

			//剩余缓冲区空间不足则扩充缓冲区
			if(TARGET_EMPTY_SPACE(ptarget) < (STLV_PACK_SAFE_LEN(bridge_pack_len)))
			{
				ret = expand_target_buff(penv , ptarget);
				if(ret < 0)
				{
					slog_log(slogd , SL_ERR , "%s expand target failed! drop package. flush buff imcomplete. but target buff left space is too small! left:%d proper:%d" ,
							__FUNCTION__ , TARGET_EMPTY_SPACE(ptarget) , STLV_PACK_SAFE_LEN(bridge_pack_len));
					pbridge_info->send.dropped++;
					pbridge_info->send.latest_drop = curr_ts;
					ptraffic->dropped++;
					ptraffic->latest_drop = curr_ts;
					continue;
				}
			}

			//pack
			ret = pkg_2_target(penv , ptarget , (char *)pstpack , bridge_pack_len);
			//ret = pkg_2_target_stlv(penv , ptarget , stlv_pack , stlv_len);
			//stlv_len = STLV_PACK_ARRAY((unsigned char *)&ptarget->buff[ptarget->tail] , (unsigned char *)pstpack , bridge_pack_len);
			if(ret != 0)
			{
				slog_log(slogd , SL_ERR , "%s flush buff imcomplete and drop package for stlv pack failed!" , __FUNCTION__);
				pbridge_info->send.dropped++;
				pbridge_info->send.latest_drop = curr_ts;
				ptraffic->dropped++;
				ptraffic->latest_drop = curr_ts;
				continue;
			}

			//upate
			slog_log(slogd , SL_VERBOSE , "%s flush buff imcomplete and saved to buff success!" , __FUNCTION__);

			//只记录业务包
			if(pstpack->pack_head.pkg_type == BRIDGE_PKG_TYPE_NORMAL)
			{
				pbridge_info->send.handled++;
				pbridge_info->send.max_pkg_size = (pstpack->pack_head.data_len>pbridge_info->send.max_pkg_size)?pstpack->pack_head.data_len:pbridge_info->send.max_pkg_size;
				if(pbridge_info->send.min_pkg_size <= 0)
					pbridge_info->send.min_pkg_size = pstpack->pack_head.data_len;
				else
					pbridge_info->send.min_pkg_size = (pstpack->pack_head.data_len<pbridge_info->send.min_pkg_size)?pstpack->pack_head.data_len:pbridge_info->send.min_pkg_size;
				pbridge_info->send.ave_pkg_size = (pbridge_info->send.ave_pkg_size*(pbridge_info->send.handled-1)+pstpack->pack_head.data_len)/pbridge_info->send.handled;
			}
			//单条链接记录所有包
			ptraffic->handled++;
			ptraffic->max_size = (pstpack->pack_head.data_len>ptraffic->max_size)?pstpack->pack_head.data_len:ptraffic->max_size;
			if(ptraffic->min_size <= 0)
				ptraffic->min_size = pstpack->pack_head.data_len;
			else
				ptraffic->min_size = (pstpack->pack_head.data_len<ptraffic->min_size)?pstpack->pack_head.data_len:ptraffic->min_size;
			ptraffic->ave_size = (ptraffic->ave_size*(ptraffic->handled-1)+pstpack->pack_head.data_len)/ptraffic->handled;

			ret = TARGET_DATA_LEN(ptarget);
			ptarget->max_tail = ret>ptarget->max_tail?ret:ptarget->max_tail;
			continue;
		}

		//缓冲区已空，或可发送
		//打包
		ret = pkg_2_target(penv , ptarget , (char *)pstpack , bridge_pack_len);
		//ret = pkg_2_target_stlv(penv , ptarget , stlv_pack , stlv_len);
		//stlv_len = STLV_PACK_ARRAY(ptarget->buff , pstpack , bridge_pack_len);
		if(ret != 0)
		{
			pbridge_info->send.dropped++;
			pbridge_info->send.latest_drop = curr_ts;
			ptraffic->dropped++;
			ptraffic->latest_drop = curr_ts;
			slog_log(slogd , SL_ERR , "%s drop package for stlv pack failed!" , __FUNCTION__);
			continue;
		}

		//记录业务包
		if(pstpack->pack_head.pkg_type == BRIDGE_PKG_TYPE_NORMAL)
		{
			pbridge_info->send.handled++;
			pbridge_info->send.max_pkg_size = (pstpack->pack_head.data_len>pbridge_info->send.max_pkg_size)?pstpack->pack_head.data_len:pbridge_info->send.max_pkg_size;
			if(pbridge_info->send.min_pkg_size <= 0)
				pbridge_info->send.min_pkg_size = pstpack->pack_head.data_len;
			else
				pbridge_info->send.min_pkg_size = (pstpack->pack_head.data_len<pbridge_info->send.min_pkg_size)?pstpack->pack_head.data_len:pbridge_info->send.min_pkg_size;
			pbridge_info->send.ave_pkg_size = (pbridge_info->send.ave_pkg_size*(pbridge_info->send.handled-1)+pstpack->pack_head.data_len)/pbridge_info->send.handled;
		}
		//单条链接记录所有包
		ptraffic->handled++;
		ptraffic->max_size = (pstpack->pack_head.data_len>ptraffic->max_size)?pstpack->pack_head.data_len:ptraffic->max_size;
		if(ptraffic->min_size <= 0)
			ptraffic->min_size = pstpack->pack_head.data_len;
		else
			ptraffic->min_size = (pstpack->pack_head.data_len<ptraffic->min_size)?pstpack->pack_head.data_len:ptraffic->min_size;
		ptraffic->ave_size = (ptraffic->ave_size*(ptraffic->handled-1)+pstpack->pack_head.data_len)/ptraffic->handled;

		append_sending_node(penv , ptarget);
		slog_log(slogd , SL_DEBUG , "%s is saving curr package to %d data_len:%d seq:%d" , __FUNCTION__ , ptarget->proc_id ,ret , ptraffic->handled);
		//发送
		/*
		ret = TARGET_DATA_LEN(ptarget);
		ptarget->max_tail = ret>ptarget->max_tail?ret:ptarget->max_tail;
		ptarget->delay_starts_ms = curr_ms;
		slog_log(slogd , SL_DEBUG , "%s is sending curr package to %d data_len:%d seq:%d" , __FUNCTION__ , ptarget->proc_id ,ret , ptraffic->handled);
		ret = flush_target(penv , ptarget);
		switch(ret)
		{
		case -1:	//出现网络故障，则重置链接
			close_target_fd(penv , ptarget , __FUNCTION__ , penv->epoll_fd , 1);
		break;
		case 0:	//未发送数据或发送部分数据 需要加入sending_list
		case 2:
			append_sending_node(penv , ptarget);
		break;
		case 1:	//全部发送则不管了
		break;
		default:
		break;
		}
		*/

		/*
		slog_log(slogd , SL_DEBUG , "%s is sending curr package to %d data_len:%d seq:%d" , __FUNCTION__ , ptarget->proc_id ,ret , ptraffic->handled);
		ret = direct_send(penv , ptarget , stlv_pack , stlv_len);
		if(ret < 0)		//出现网络故障，则重置链接
			close_target_fd(penv , ptarget , __FUNCTION__ , penv->epoll_fd , 1);
		else if(ret < stlv_len)	//未发送数据或发送部分数据 需要加入sending_list
		{
			append_sending_node(penv , ptarget);
			ret = pkg_2_target_stlv(penv , ptarget , &stlv_pack[ret] , stlv_len-ret);
			if(ret != 0)
			{
				pbridge_info->send.dropped++;
				pbridge_info->send.latest_drop = curr_ts;
				ptraffic->dropped++;
				ptraffic->latest_drop = curr_ts;
				slog_log(slogd , SL_ERR , "%s drop package for pkg_2_target_stlv failed!" , __FUNCTION__);
				//这种情况似乎需要断连 因为部分数据已经遗失
			}
			else
				slog_log(slogd , SL_DEBUG , "%s save part of data to target! stlv_len:%d ret:%d" , __FUNCTION__ , stlv_len , ret);
		}
		*/
		//发送完成

	}

	end_ms = get_curr_ms();
	return (end_ms-enter_ms);
}



static int read_client_socket(int fd , bridge_hub_t *phub)
{

	char pack_buff[BRIDGE_PACK_LEN];
	proc_entry_t proc_entry;
	//char *next_buff = NULL;
	client_info_t *pclient = NULL;
	bridge_package_t *recv_pkg = NULL;
	bridge_info_t *pbridge_info = &penv->bridge_info;
	int len = 0;
	int space = 0;
	int pos = 0;
	int ret = 0;
	unsigned int pack_len = 0;
	unsigned int value_len = 0;
	char info;
	long curr_ts = time(NULL);
	long long curr_ms = get_curr_ms();
	int diff_ms = 0;

#ifdef _TRACE_DEBUG
	char test_buff[_TRACE_DEBUG_BUFF_LEN] = {0};
#endif

	/***Search Client Info*/
	/*
	pclient = client_list.list;
	for(i=0; i<client_list.total_count && pclient; i++)
	{
		//match fd
		if(pclient->fd == fd)
			break;
		//search next
		pclient = pclient->next;
	}*/
	pclient = fd_2_client(penv , fd);
	if(!pclient)
	{
		slog_log(slogd , SL_ERR , "%s can not find fd:%d" , __FUNCTION__ , fd);
		return -1;
	}


	//持续读哦 爽得一笔
	while(1)
	{
		//calc space
		space = sizeof(pclient->main_buff) - pclient->tail;

		//read sock
		len = recv(fd , &pclient->buff[pclient->tail] , space , 0);

		//failed
		if(len < 0)
		{
			switch(errno)
			{
			case EAGAIN:
			//case EWOULDBLOCK:
				//no more data
				slog_log(slogd , SL_VERBOSE , "%s no more data!" , __FUNCTION__);
			break;

			default:
				//other other close fd
				slog_log(slogd , SL_ERR , "%s failed! err:%s and try to close connection. %d<%s:%d>" , __FUNCTION__ , strerror(errno) ,
						fd , pclient->client_ip , pclient->client_port);
				pbridge_info->recv.reset_connect++;
				pbridge_info->recv.latest_reset = curr_ts;
				free_client_info(pclient);
			break;
			}

			break;
		}

		//peer close
		if(len == 0)
		{
			slog_log(slogd , SL_INFO , "%s detect client closed! close connection %d<%s:%d>" , __FUNCTION__ , fd , pclient->client_ip ,
					pclient->client_port);

			/*send error*/
			memset(&proc_entry , 0 , sizeof(proc_entry));
			proc_entry.proc_id = 0;
			strncpy(proc_entry.ip_addr , pclient->client_ip , sizeof(proc_entry.ip_addr));
			proc_entry.port = pclient->client_port;

			send_carrier_msg(&carrier_env , CR_MSG_ERROR , MSG_ERR_T_LOST_CONN , &proc_entry , NULL);
			/*release client_info*/
			pbridge_info->recv.reset_connect++;
			pbridge_info->recv.latest_reset = curr_ts;
			free_client_info(pclient);
			break;
		}

		//handle data
		slog_log(slogd , SL_VERBOSE , "%s read %d data." , __FUNCTION__ , len);
		pclient->tail += len;

		//持续解包
		pos = 0;
		for(;;)
		{
			if(pos >= pclient->tail)
			{
				slog_log(slogd , SL_VERBOSE , "%s unpack this-time readed data finish!" , __FUNCTION__);
				break;
			}

			//unpack package
			//memset(pack_buff , 0 , sizeof(pack_buff));
			value_len = STLV_UNPACK(&info , pack_buff , &pclient->buff[pos] , pclient->tail-pos , &pack_len);
			if(value_len == 0) //解包失败，
			{
				//接收缓冲区不足.不应出现对端发送一个>=max-pkg-size的包 断掉
				if(info == STLV_UNPACK_FAIL_BUFF_LEN && (pclient->tail-pos)>=sizeof(pclient->main_buff))
				{
					slog_log(slogd , SL_ERR , "<%s> failed! recv buff not enoght?! wtf!! it may be from wrong source! now try to close  connection. %d<%s:%d>" , __FUNCTION__ ,
													fd , pclient->client_ip , pclient->client_port);
					pbridge_info->recv.reset_connect++;
					pbridge_info->recv.latest_reset = curr_ts;
					free_client_info(pclient);
					return -1;
				}
				//check sum错误 数据非法 断掉连接
				if(info == STLV_UNPACK_FAIL_CHECK_SUM)
				{
					slog_log(slogd , SL_ERR , "<%s> failed! check sum error! it may be from wrong source! now try to close  connection. %d<%s:%d>" , __FUNCTION__ ,
								fd , pclient->client_ip , pclient->client_port);
					pbridge_info->recv.reset_connect++;
					pbridge_info->recv.latest_reset = curr_ts;
					free_client_info(pclient);
					return -1;
				}

				//非完整包，正常情况，等待下一次数据到来
				slog_log(slogd , SL_DEBUG , "%s imcomplete pack. waiting for next recv!" , __FUNCTION__);
				break;
			}

			//unpack success
			pos += pack_len;
			recv_pkg = (bridge_package_t *)pack_buff;

			//check manage
			if(recv_pkg->pack_head.pkg_type==BRIDGE_PKG_TYPE_CR_MSG)
			{
				manager_handle(carrier_env.pmanager , pack_buff , slogd);
				continue;	//no need to dispatch to channel
			}

			//check inner-proto
			if(recv_pkg->pack_head.pkg_type == BRIDGE_PKG_TYPE_INNER_PROTO)
			{
				recv_inner_proto(penv , pclient , pack_buff);
				continue;
			}

			//check verify
			if(!pclient->verify)
			{
				slog_log(slogd , SL_ERR , "<%s> recved from no-verfiy client. will close %d<%s:%d>" , __FUNCTION__ , fd , pclient->client_ip ,
						pclient->client_port);
				pbridge_info->recv.reset_connect++;
				pbridge_info->recv.latest_reset = curr_ts;
				free_client_info(pclient);
				return -1;
			}

#ifdef _TRACE_DEBUG
			memcpy(test_buff , &recv_pkg->pack_data , recv_pkg->pack_head.data_len);
			test_buff[recv_pkg->pack_head.data_len] = 0;
			slog_log(slogd , SL_DEBUG , "%s recv an pack! value:%d pack_len:%d pos:%d tail:%d. src:%d dest:%d pkg_type:%d seq:%d content:%s" , __FUNCTION__ , value_len , pack_len ,
					pos , pclient->tail , recv_pkg->pack_head.sender_id , recv_pkg->pack_head.recver_id , recv_pkg->pack_head.pkg_type ,
					pclient->traffic.handled+1 , test_buff);
#endif

			//记录业务包数据.[内部协议不进入统计]
			pbridge_info->recv.handled++;
			pbridge_info->recv.max_pkg_size = (recv_pkg->pack_head.data_len>pbridge_info->recv.max_pkg_size)?recv_pkg->pack_head.data_len:pbridge_info->recv.max_pkg_size;
			if(pbridge_info->recv.min_pkg_size <= 0)
				pbridge_info->recv.min_pkg_size = recv_pkg->pack_head.data_len;
			else
				pbridge_info->recv.min_pkg_size = (recv_pkg->pack_head.data_len<pbridge_info->recv.min_pkg_size)?recv_pkg->pack_head.data_len:pbridge_info->recv.min_pkg_size;
			pbridge_info->recv.ave_pkg_size = (pbridge_info->recv.ave_pkg_size*(pbridge_info->recv.handled-1)+recv_pkg->pack_head.data_len)/pbridge_info->recv.handled;
			//traffic
			pclient->traffic.handled++;
			pclient->traffic.max_size = (recv_pkg->pack_head.data_len>pclient->traffic.max_size)?recv_pkg->pack_head.data_len:pclient->traffic.max_size;
			if(pclient->traffic.min_size <= 0)
				pclient->traffic.min_size = recv_pkg->pack_head.data_len;
			else
				pclient->traffic.min_size = (recv_pkg->pack_head.data_len<pclient->traffic.min_size)?recv_pkg->pack_head.data_len:pclient->traffic.min_size;
			pclient->traffic.ave_size = (pclient->traffic.ave_size*(pclient->traffic.handled-1)+recv_pkg->pack_head.data_len)/pclient->traffic.handled;
			diff_ms = curr_ms - recv_pkg->pack_head.send_ms;
			diff_ms = diff_ms<=0?0:diff_ms;
			pclient->traffic.delay_time = (pclient->traffic.delay_time * pclient->traffic.delay_count + diff_ms)/(pclient->traffic.delay_count + 1);
			pclient->traffic.delay_count++;

			//try flush buff


			//dispatch
			//ret = append_recv_channel(phub , pack_buff);
			ret = recv_client_pkg(penv , pclient , (bridge_package_t *)pack_buff);
			if(ret == 0)
				slog_log(slogd , SL_DEBUG , "%s >>Read From Socket and dispatch pack to Bridge success! type:%d,length:%d,src:%d,recv:%d,data_len:%d" , __FUNCTION__ , info , value_len , ((bridge_package_t*)pack_buff)->pack_head.sender_id ,
								((bridge_package_t*)pack_buff)->pack_head.recver_id , ((bridge_package_t*)pack_buff)->pack_head.data_len);
			else
			{
				/*
				if(ret == 1)
					slog_log(slogd , SL_ERR , "%s >>Read From Socket and dispatch pack to Bridge failed for err! type:%d,length:%d,src:%d,recv:%d,data_len:%d" , __FUNCTION__ , info , value_len , ((bridge_package_t*)pack_buff)->pack_head.sender_id ,
									((bridge_package_t*)pack_buff)->pack_head.recver_id , ((bridge_package_t*)pack_buff)->pack_head.data_len);
				else
					slog_log(slogd , SL_ERR , "%s >>Read From Socket and dispatch pack to Bridge failed for channel full! type:%d,length:%d,src:%d,recv:%d,data_len:%d" , __FUNCTION__ , info , value_len , ((bridge_package_t*)pack_buff)->pack_head.sender_id ,
									((bridge_package_t*)pack_buff)->pack_head.recver_id , ((bridge_package_t*)pack_buff)->pack_head.data_len);*/
				slog_log(slogd , SL_ERR , "%s >>Read From Socket and dispatch pack to Bridge failed ! type:%d,length:%d,src:%s,recv:%d,data_len:%d" , __FUNCTION__ , info , value_len , pclient->proc_name ,
									((bridge_package_t*)pack_buff)->pack_head.recver_id , ((bridge_package_t*)pack_buff)->pack_head.data_len);
				pbridge_info->recv.dropped++;
				pbridge_info->recv.latest_drop = curr_ts;
				pclient->traffic.dropped++;
				pclient->traffic.latest_drop = curr_ts;
			}
		}

		//修正缓冲区
		//缓冲区再无数据
		if(pclient->tail<=pos)
		{
			pclient->tail = 0;
			continue;
		}

		//没有成功解包一次不用修正缓冲区
		if(pos == 0)
		{
			continue;
		}

		//缓冲区有数据[拷贝到另一缓冲区头部并更改buff指针]
		if(pclient->buff == pclient->main_buff)
		{
			//slog_log(slogd , SL_DEBUG , "%s copy %d bytes" , __FUNCTION__ , pclient->tail-pos);
			memcpy(pclient->back_buff , &pclient->buff[pos] , pclient->tail-pos);
			pclient->buff = pclient->back_buff;
			pclient->tail = pclient->tail-pos;
		}
		else
		{
			//slog_log(slogd , SL_DEBUG , "%s copy %d bytes" , __FUNCTION__ , pclient->tail-pos);
			memcpy(pclient->main_buff , &pclient->buff[pos] , pclient->tail-pos);
			pclient->buff = pclient->main_buff;
			pclient->tail = pclient->tail-pos;
		}

	}

	return 0;
}


static int print_bridge_info(bridge_hub_t *phub)
{
	FILE *fp;

	/***Arg Check*/
	if(!phub)
	{
		return -1;
	}

	/***Open FP*/
	fp = fopen("channel.info" , "w+");
	if(!fp)
	{
		slog_log(slogd , SL_ERR , "open channel.info failed!");
		return 0;
	}

	/***Print Send*/
	fprintf(fp , "\t--------Send Channel--------\n");
	fprintf(fp , "Total Len:%d\n" , phub->send_buff_size);
	fprintf(fp , "Empty:%d\nSended Count:%d\nSending Count:%d\n" , phub->send_head==phub->send_tail?1:0 ,
			phub->sended_count , phub->sending_count);
	fprintf(fp , "Head:%d\nTail:%d\n" , phub->send_head , phub->send_tail);

	/***Print Recv*/
	fprintf(fp , "\t--------Recv Channel--------\n");
	fprintf(fp , "Total Len:%d\n" , phub->recv_buff_size);
	fprintf(fp , "Empty:%d\nRecved Count:%d\nRecving Count:%d\n" , phub->recv_head==phub->recv_tail?1:0 , phub->recved_count , phub->recving_count);
	fprintf(fp , "Head:%d\nTail:%d\n" , phub->recv_head , phub->recv_tail);

	fflush(fp);
	fclose(fp);

	slog_log(slogd , SL_INFO , "Main:Print Bridge Info Finished!");
	return 0;
}


/*
* 修改socket缓冲区大小
* @sock_fd: 套接字描述符
* @send_size: 发送缓冲区大小
* @recv_size: 接收缓冲区大小
* @no_delay: 关闭nagle算法
*/
static int set_sock_option(int sock_fd , int send_size , int recv_size , int no_delay)
{
	int s_size;
	int r_size;
	int nagle;
	int keepalive = 1;

	socklen_t opt_len;
	int ret;

	opt_len = sizeof(int);

	/*设置缓冲区大小*/
	if(send_size > 0)
	{
		ret = setsockopt(sock_fd , SOL_SOCKET , SO_SNDBUF , &send_size , opt_len);
		if(ret < 0)
			slog_log(slogd , SL_ERR , "<%s> fd:%d set sock send buff to %d failed!err:%s" , __FUNCTION__ , sock_fd , send_size , strerror(errno));
	}

	if(recv_size > 0)
	{
		ret = setsockopt(sock_fd , SOL_SOCKET , SO_RCVBUF , &recv_size , opt_len);
		if(ret < 0)
			slog_log(slogd , SL_ERR , "<%s> fd:%d set sock recv buff to %d failed!err:%s" , __FUNCTION__ , sock_fd , recv_size , strerror(errno));
	}

	/*KeepAlive*/
	ret = setsockopt(sock_fd , SOL_SOCKET , SO_KEEPALIVE , &keepalive , sizeof(keepalive));
	if(ret < 0)
		slog_log(slogd , SL_ERR , "<%s> fd:%d set sock keepalive failed!err:%s" , __FUNCTION__ , sock_fd , strerror(errno));


	/*关闭nagle算法*/
	if(no_delay == 1)
	{
		ret = setsockopt(sock_fd , IPPROTO_TCP , TCP_NODELAY , &no_delay , sizeof(no_delay));
		if(ret < 0)
			slog_log(slogd , SL_ERR , "<%s> fd:%d set sock no_delay failed!err:%s" , __FUNCTION__ , sock_fd , strerror(errno));
	}

	/*打印设置之后的缓冲区长度*/
	getsockopt(sock_fd , SOL_SOCKET , SO_SNDBUF , &s_size , &opt_len);
	getsockopt(sock_fd , SOL_SOCKET , SO_RCVBUF , &r_size , &opt_len);
	getsockopt(sock_fd , IPPROTO_TCP , TCP_NODELAY , &nagle , &opt_len);
	getsockopt(sock_fd , SOL_SOCKET , SO_KEEPALIVE , &keepalive , &opt_len);

    slog_log(slogd , SL_INFO , "MAIN:socket new send buff size: %dK; recv buff size: %dK NoDelay:%d KeepAlive:%d" , s_size/1024 ,
    		r_size/1024 , nagle , keepalive);
    return 0;
}

/*
 * 清空某个channel的target发送缓冲区
 * -1:错误
 *  0:成功
 */
/*
static int flush_target(target_detail_t *ptarget)
{
	int ret = 0;
	int result = 0;
	//ptarget is non-null

	slog_log(slogd , SL_INFO , "%s is sending package to %d. rest_count:%d latest_ts:%ld" , __FUNCTION__ , ptarget->proc_id ,
						ptarget->ready_count , ptarget->latest_ts);

	//send
	ret = send(ptarget->fd ,  ptarget->buff , ptarget->tail , 0);

	//send failed
	if(ret < 0)
	{
		switch(errno)
		{
		case EAGAIN:	//socket发送缓冲区满，稍后再试
		//case EWOULDBLOCK:
			slog_log(slogd , SL_INFO , "%s send failed for socket buff full!" , __FUNCTION__);
		break;
		default:
			slog_log(slogd , SL_ERR , "%s send failed for err:%s. and will reset connection." , __FUNCTION__ , strerror(errno));
			//此时出现错误，应主动关闭链接，用于清除本端和对端的缓冲区数据，防止错误包雪崩。并在后续进行重连
			close(ptarget->fd);
			ptarget->fd = -1;
			ptarget->connected = 0;
			ptarget->buff = ptarget->main_buff;
			ptarget->tail = 0;
			result = -1;
		break;
		}

		return result;
	}

	//send all of data
	if(ret >= ptarget->tail)
	{
		slog_log(slogd , SL_DEBUG , "%s flush all buff success!" , __FUNCTION__ );
		ptarget->tail = 0;
		ptarget->latest_ts = 0;
		ptarget->ready_count = 0;
		return 0;
	}

	//send part of data
	slog_log(slogd , SL_DEBUG , "%s flush part of buff! sended:%d all:%d" , __FUNCTION__ , ret , ptarget->tail);
	if(ptarget->buff == ptarget->main_buff)
	{
		memcpy(ptarget->back_buff , &ptarget->buff[ret] , ptarget->tail-ret);
		ptarget->buff = ptarget->back_buff;
	}
	else
	{
		memcpy(ptarget->main_buff , &ptarget->buff[ret] , ptarget->tail-ret);
		ptarget->buff = ptarget->main_buff;
	}
	ptarget->tail = ptarget->tail - ret;
	return 0;
}
*/

/*
static int world_tick()
{
	long start_ms = get_curr_ms();
	connect_to_remote(penv);
	if(penv->proc_id <= MANAGER_PROC_ID_MAX)	//manager
		manage_tick_print(penv);
	check_bridge(penv);
	check_client_info(penv);
	check_run_statistics(penv);
	check_signal_stat(penv);
	return (get_curr_ms()-start_ms);
}*/

static int add_ticker(carrier_env_t *penv)
{
	int ret = -1;
	int slogd = penv->slogd;

	/***Add*/
	ret = append_carrier_ticker(penv , connect_to_remote , TIME_TICKER_T_CIRCLE , TICK_CONNECT_TO_REMOTE , "connect_to_remote" ,
			(void *)penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add connect_to_remote failed!" , __FUNCTION__);
		return -1;
	}
	if(penv->proc_id <= MANAGER_PROC_ID_MAX)
	{
		ret = append_carrier_ticker(penv , manage_tick_print , TIME_TICKER_T_CIRCLE , TICK_MANAGE_PRINT , "manage_tick_print" ,
				penv);
		if(ret < 0)
		{
			slog_log(slogd , SL_ERR , "<%s> add manage_tick_print failed!" , __FUNCTION__);
			return -1;
		}
	}
	ret = append_carrier_ticker(penv , check_bridge , TIME_TICKER_T_CIRCLE , TICK_CHECK_BRIDGE , "check_bridge" ,
			penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_bridge failed!" , __FUNCTION__);
		return -1;
	}
	ret = append_carrier_ticker(penv , check_client_info , TIME_TICKER_T_CIRCLE , TICK_CHECK_CLIENT_INFO , "check_client_info" ,
			penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_client_info failed!" , __FUNCTION__);
		return -1;
	}
	ret = append_carrier_ticker(penv , check_run_statistics , TIME_TICKER_T_CIRCLE , TICK_CHECK_RUN_STATISTICS , "check_run_statistics" ,
			penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_run_statistics failed!" , __FUNCTION__);
		return -1;
	}
	ret = append_carrier_ticker(penv , check_signal_stat , TIME_TICKER_T_CIRCLE , TICK_CHECK_SIG_STAT , "check_signal_stat" ,
			penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_signal_stat failed!" , __FUNCTION__);
		return -1;
	}
	/*
	ret = append_carrier_ticker(penv , iter_sending_node , TIME_TICKER_T_CIRCLE , TICK_ITER_SENDING_LIST , "iter_sending_list" ,
				penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_signal_stat failed!" , __FUNCTION__);
		return -1;
	}
	*/
	ret = append_carrier_ticker(penv , check_hash , TIME_TICKER_T_CIRCLE , TICK_CHECK_HASH_MAP , "check_hash_map" ,
					penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_hash failed!" , __FUNCTION__);
		return -1;
	}

	ret = append_carrier_ticker(penv , check_snd_buff_memory , TIME_TICKER_T_CIRCLE , TICK_CHECK_SND_BUFF_MEMROY , "check_snd_buff_memory" ,
						penv);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_hash failed!" , __FUNCTION__);
		return -1;
	}

    ret = append_carrier_ticker(penv , check_tmp_file , TIME_TICKER_T_CIRCLE , TICK_CHECK_TMP_FILE , "check_tmp_file" , penv);
    if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> add check_tmp_file failed!" , __FUNCTION__);
		return -1;
	}

	return 0;
}

static void handle_connecting_fd(target_detail_t *ptarget , struct epoll_event *pevent)
{
	struct sockaddr_in target_addr;
	manage_item_t *pitem = NULL;
	char buff[1024] = {0};
	char key[BRIDGE_PROC_CONN_VERIFY_KEY_LEN] = {0};
	int handle_fd = -1;
	int ret = -1;
	int success = 0;
	if(!ptarget || !pevent)
		return;
	handle_fd = ptarget->fd;
	socklen_t  length = sizeof(struct sockaddr_in);

	//检查连接
	//1.可写但不可读。连接OK
	if(!(pevent->events & EPOLLIN) && (pevent->events & EPOLLOUT))
	{
		//target_info.epoll_manage_in_connect--;
		slog_log(slogd , SL_INFO , "%s connecting fd:%d non-readable and writable. connect to [%s:%d] success!" , __FUNCTION__ , handle_fd ,
				ptarget->ip_addr , ptarget->port);

		//update info
		ptarget->connected = TARGET_CONN_DONE;
		ptarget->snd_buff = NULL;
		ptarget->snd_buff_len = 0;
		ptarget->snd_head = ptarget->snd_tail = 0;

		//mange only
		pitem = get_manage_item_by_id(penv , ptarget->proc_id);
		if(pitem)
		{
			pitem->latest_update = time(NULL);
			pitem->my_conn_stat = ptarget->connected;
		}

		//send
		gen_verify_key(penv , key , sizeof(key));
		send_inner_proto(penv , ptarget , INNER_PROTO_VERIFY_REQ , key , NULL);
		set_sock_option(handle_fd , CARRIER_SOCKET_BUFF_BIG , 1024*2 , 1);
		return;
	}

	//2.可读可写[进一步检查],这种情况一般是发生了错误。因为对端不会往这个socket写数据，如果可读则一定是有错误发生了.
	if((pevent->events & EPOLLIN) && (pevent->events & EPOLLOUT))
	{
		slog_log(slogd , SL_INFO , "%s. connecting fd:%d readable and writable.[%s:%d]." , __FUNCTION__ , handle_fd , ptarget->ip_addr ,
				ptarget->port);

		//read err-info
		ret = read(handle_fd , buff , sizeof(buff));
		if(ret > 0)
			slog_log(slogd , SL_ERR , "read %d:%s" , ret , buff);
		else if(ret == 0)
			slog_log(slogd , SL_ERR , "closed");
		else
			slog_log(slogd , SL_ERR , "err:%s" , strerror(errno));

		//进一步尝试连接，如果连接失败且错误码[EISCONN 则是OK的 其他就GG了]
		bzero(&target_addr , sizeof(struct sockaddr_in));
		target_addr.sin_family = AF_INET;
		target_addr.sin_port = htons(CARRIER_REAL_PORT(ptarget->port));
		inet_pton(AF_INET , ptarget->ip_addr , &target_addr.sin_addr);
		ret = connect(handle_fd , (struct sockaddr *)&target_addr , length);

		//check ret
		if(ret < 0)
		{
			//connect success
			if(errno == EISCONN)
			{
				slog_log(slogd , SL_INFO , "%s .connecting fd:%d readable and writable. connect to [%s:%d] success! " , __FUNCTION__ ,
						handle_fd , ptarget->ip_addr , ptarget->port);

				success = 1;
				//update info
				ptarget->connected = TARGET_CONN_DONE;
				ptarget->snd_buff = NULL;
				ptarget->snd_buff_len = ptarget->snd_head = ptarget->snd_tail = 0;
				set_sock_option(handle_fd , CARRIER_SOCKET_BUFF_BIG , 1024*2 , 1);

				//mange only
				pitem = get_manage_item_by_id(penv , ptarget->proc_id);
				if(pitem)
				{
					pitem->latest_update = time(NULL);
					pitem->my_conn_stat = ptarget->connected;
				}

				//should read valid data.but not in this soft.
			}
			else //connect fail
				slog_log(slogd , SL_ERR , "%s.connecting fd:%d readable and writable. re-connect to [%s:%d] wrong." , __FUNCTION__ , handle_fd ,
						ptarget->ip_addr , ptarget->port);

		}
		else
			slog_log(slogd , SL_ERR , "%s.connecting fd:%d readable and writable. connect to [%s:%d] failed! reconn ret %d!" , __FUNCTION__ , handle_fd ,
					ptarget->ip_addr , ret );

		//del epoll manage
		if(!success)
		{
			close_target_fd(penv , ptarget , "connecting_fd:r&w" , penv->epoll_fd , 1);
		}
		return;
	}

	//3.可读不可写
	if((pevent->events & EPOLLIN) && (pevent->events & EPOLLOUT))
	{
		slog_log(slogd , SL_ERR , "%s.connecting fd:%d readable and non-writable. connect to [%s:%d] failed!" , __FUNCTION__ , handle_fd ,
				ptarget->ip_addr);

		//try print err
		ret = read(handle_fd , buff , sizeof(buff));
		if(ret > 0)
			slog_log(slogd , SL_ERR , "%s readed %d:%s" , __FUNCTION__ , ret , buff);
		else
			slog_log(slogd , SL_ERR , "%s err:%s" , __FUNCTION__ , strerror(errno));

		close_target_fd(penv , ptarget , "connecting_fd:r&nw" , penv->epoll_fd , 1);
		return;
	}

	//4.不可读不可写？
	slog_log(slogd , SL_FATAL , "%s.connecting fd:%d non-readable and non-writable. connect to [%s:%d] failed!" , __FUNCTION__ , handle_fd ,
			ptarget->ip_addr);

	//reset
	close_target_fd(penv , ptarget , "connecting_fd:nr&nw" , penv->epoll_fd , 1);
	return;
}

//target sock可读一般情况是对端关闭了连接
static int handle_target_fd(target_detail_t *ptarget , struct epoll_event *pevent)
{
	char buff[1024] = {0};
	int ret = -1;
	manage_item_t *pmanage_item = NULL;

	if(!ptarget || !pevent)
		return -1;

	if(pevent->events & EPOLLIN)
	{
		slog_log(slogd , SL_INFO , "<%s> %d[%s:%d]readable." , __FUNCTION__ , ptarget->fd , ptarget->ip_addr , ptarget->port);
		ret = read(ptarget->fd , buff , sizeof(buff));
		if(ret > 0)
			slog_log(slogd , SL_INFO , "<%s> read %d:%s" , __FUNCTION__ , ret , buff);
		else if(ret == 0)
		{
			slog_log(slogd , SL_INFO , "<%s> detect peer %d[%s:%d] closed!" , __FUNCTION__ , ptarget->fd , ptarget->ip_addr , ptarget->port);
			if(penv->proc_id<=MANAGER_PROC_ID_MAX && penv->pmanager)
			{
				slog_log(penv->pmanager->msg_slogd , SL_INFO , ERROR_PRINT_PREFIX" proc %d close a connection. <%s:%d>[%s:%d]" , ptarget->proc_id , ptarget->target_name ,
						ptarget->proc_id , ptarget->ip_addr , ptarget->port);
				pmanage_item = get_manage_item_by_id(penv , ptarget->proc_id);
				pmanage_item->my_conn_stat = TARGET_CONN_PROC;	//will reconnect again
				pmanage_item->latest_update = time(NULL);
				memset(&pmanage_item->conn_stat , 0 , sizeof(pmanage_item->conn_stat));

			}
			close_target_fd(penv , ptarget , __FUNCTION__ , penv->epoll_fd , 1);
		}
		else	//would not happend
		{
			slog_log(slogd , SL_INFO , "<%s> detect peer %d[%s:%d] err:%s" , __FUNCTION__ , ptarget->fd , ptarget->ip_addr , ptarget->port ,
					strerror(errno));
			close_target_fd(penv , ptarget , __FUNCTION__ , penv->epoll_fd , 1);
		}
	}
	else
	{
		//do nothing
		//slog_log(slogd , SL_INFO , "<%s> %d not read." , __FUNCTION__ , ptarget->fd);
	}


	return 0;
}


static int free_client_info(client_info_t *pclient)
{
	client_info_t *pprev = NULL;
	cr_hash_entry_t *pentry = NULL;
	char found = 0;

	if(!pclient)
		return -1;

	if(client_list.total_count <= 0)
	{
		slog_log(slogd , SL_ERR , "%s failed. list empty! client_fd:%d" , __FUNCTION__ , pclient->fd);
		return -1;
	}

	//search
	do
	{
		pprev = client_list.list;
		if(!pprev)
			break;

		//特殊节点[pclient恰好是第一个]
		if(pprev == pclient)
		{
			client_list.list = pclient->next;
			found = 1;
			break;
		}

		//other
		while(pprev)
		{
			if(pprev->next == pclient)
			{
				pprev->next = pclient->next;
				found = 1;
				break;
			}
			pprev = pprev->next;
		}
		break;
	}while(0);

	//free
	if(!found)
		slog_log(slogd , SL_ERR , "%s client not found! client:%d[%s:%d]" , __FUNCTION__ , pclient->fd , pclient->client_ip , pclient->client_port);


	client_list.total_count--;
	slog_log(slogd , SL_INFO , "%s success! client:%d[%s:%d] client_count:%d" , __FUNCTION__ , pclient->fd , pclient->client_ip ,
			pclient->client_port , client_list.total_count);

	//remove from hash
	del_from_hash_map(penv , CR_HASH_MAP_T_CLIENT , CR_HASH_ENTRY_T_FD , pclient->fd);
	//确保hash数据没有问题
	pentry = fetch_hash_entry(penv , CR_HASH_MAP_T_CLIENT , CR_HASH_ENTRY_T_FD , pclient->fd);
	if(pentry)
	{
		slog_log(slogd , SL_FATAL , "<%s> client hash of (%d:%d) %s is still in hash after remove!" , __FUNCTION__ , CR_HASH_ENTRY_T_FD ,
				pclient->fd , pclient->proc_name);
		memset(pentry , 0 , sizeof(cr_hash_entry_t));	//清空 防止误用
	}

	close(pclient->fd);
	free(pclient);
	return 0;
}

//读取并分析配置文件
//flag:0系统启动时的加载 1:进程运行中的动态加载
static int read_carrier_cfg(carrier_env_t *penv , char flag)
{
	char line_buff[2048] = {0};
	char shrink_buff[2048] = {0}; //去处空格的数据
	target_detail_t *ptarget = NULL;
	FILE *fp = NULL;
	int ret = 0;
	int i = 0;
	int pos = 0;
	int line_no = 1;
	char *p , *endptr , *presult = NULL;

#define CFG_OPTION_MIN 1
#define CFG_OPTION_TARGET 1
#define CFG_OPTION_MAX_PKG 2
#define CFG_OPTION_MAX 2
	int curr_option = 0;

	//option value
	target_info_t *ptarget_info_tmp;	//直接用结构体会栈溢出 尼玛
	long value = -1;
	long max_pkg = -1;

	/***Init*/
	ptarget_info_tmp = calloc(1 , sizeof(target_info_t));
	if(!ptarget_info_tmp)
	{
		slog_log(slogd , SL_ERR , "<%s> calloc target_info failed!" , __FUNCTION__);
		return -1;
	}
	memset(ptarget_info_tmp , 0 , sizeof(target_info_t));

	/***Open File*/
	if(strlen(penv->cfg_file_name) <= 0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed. cfg name nil!" , __FUNCTION__);
		goto _failed;
	}

	fp = fopen(penv->cfg_file_name , "r");
	if(!fp)
	{
		slog_log(slogd , SL_ERR , "<%s> open %s failed! err:%s" , __FUNCTION__ , penv->cfg_file_name , strerror(errno));
		goto _failed;
	}

	/***Parse Cfg*/
	while(1)
	{
		//read
		memset(line_buff , 0 , sizeof(line_buff));
		memset(shrink_buff , 0 , sizeof(shrink_buff));
		presult = fgets(line_buff , sizeof(line_buff) , fp);
		if(!presult)
		{
			if(ferror(fp))
			{
				slog_log(slogd , SL_ERR , "<%s> read failed!line:%d" , __FUNCTION__ , line_no);
				goto _failed;
			}

			if(feof(fp))
			{
				slog_log(slogd , SL_INFO , "<%s> end reading." , __FUNCTION__);
				break;
			}

			slog_log(slogd , SL_FATAL , "<%s> read meets an problem %s at line:%d" , __FUNCTION__ , line_buff , line_no);
		}

		line_buff[strlen(line_buff) -1] = 0;	//strip \n
		slog_log(slogd , SL_VERBOSE , "<%s> >>[%d]read:%s" , __FUNCTION__ , line_no , line_buff);
		//strip space character etc.
		for(i=0,pos=0; i<strlen(line_buff); i++)
		{
			if(isspace(line_buff[i]))
				continue;

			shrink_buff[pos] = line_buff[i];
			pos++;
		}

		//strip #
		p = strchr(shrink_buff , '#');
		if(p)
			p[0] = 0;

		//dispatch first characer
		switch(shrink_buff[0])
		{
		case '#': //comments
			slog_log(slogd , SL_VERBOSE , "<%s> [%d] comment:%s" , __FUNCTION__ , line_no , &shrink_buff[1]);
			break;
		case '[': //option
			p = strchr(shrink_buff , ']');
			if(!p)
			{
				slog_log(slogd , SL_ERR , "<%s> [%d] no ']' found! content:%s" , __FUNCTION__ , line_no , line_buff);
				goto _failed;
			}

			p[0] = 0;
			if(strcmp(&shrink_buff[1] , "target") == 0)
			{
				curr_option = CFG_OPTION_TARGET;
				slog_log(slogd , SL_DEBUG , "<%s> [%d] match option:'%s'" , __FUNCTION__ , line_no , &shrink_buff[1]);
			}
			else if(strcmp(&shrink_buff[1] , "max_pkg") == 0)
			{
				curr_option = CFG_OPTION_MAX_PKG;
				slog_log(slogd , SL_DEBUG , "<%s> [%d] match option:'%s'" , __FUNCTION__ , line_no , &shrink_buff[1]);
			}
			else
			{
				slog_log(slogd , SL_ERR , "<%s> [%d] illegal option:%s" , __FUNCTION__ , line_no , &shrink_buff[1]);
				goto _failed;
			}
			break;
		case '=': //value
			if(curr_option<CFG_OPTION_MIN || curr_option>CFG_OPTION_MAX)
			{
				slog_log(slogd , SL_ERR , "<%s> [%d] illegal option value:%s" , __FUNCTION__ , line_no , &shrink_buff[1]);
				goto _failed;
			}

			//[TARGET VALUE]
			if(curr_option == CFG_OPTION_TARGET)
			{
				ptarget = calloc(1 , sizeof(target_detail_t));
				if(!ptarget)
				{
					slog_log(slogd , SL_ERR , "<%s> [%d] calloc target_detail failed!");
					goto _failed;
				}
				ret = parse_target_list(&shrink_buff[1] , ptarget);
				if(ret < 0)
				{
					slog_log(slogd , SL_ERR , "<%s> [%d] parse target failed! target:%s" , __FUNCTION__ , line_no , &shrink_buff[1]);
					goto _failed;
				}

				//append
				ptarget->next = ptarget_info_tmp->head.next;
				ptarget_info_tmp->head.next = ptarget;
				ptarget_info_tmp->target_count++;
			}

			//[MAX_PKG VALUE]
			if(curr_option == CFG_OPTION_MAX_PKG)
			{
				value = strtol(&shrink_buff[1] , &endptr , 10);
				if((errno==ERANGE && (value==LONG_MAX || value==LONG_MIN)) ||
				    (errno!=0 && value==0))
				{
					slog_log(slogd , SL_ERR , "<%s> [%d] illegal option value:%s" , __FUNCTION__ , line_no , &shrink_buff[1]);
					goto _failed;
				}

				if(endptr == &shrink_buff[1])
				{
					slog_log(slogd , SL_ERR , "<%s> [%d] illegal option value:%s. No digits were found" , __FUNCTION__ , line_no , &shrink_buff[1]);
					goto _failed;
				}

				max_pkg = value;
			}

			break;
		case '\0':	//empty line
			break;
		default:
			slog_log(slogd , SL_ERR , "<%s> parse cfg failed. illegal head character:'%c' at line:%d" , __FUNCTION__ , line_buff[0] , line_no);
			goto _failed;
		}

		line_no++;
	}

	//Set
_success:
    if(flag == 0)	//系统启动时全部覆盖
    	copy_target_info(&target_info , ptarget_info_tmp , 1);
    else	//进程运行中动态修正
    {
    	copy_target_info(&target_info , ptarget_info_tmp , 0);
    	rebuild_manager_item_list(penv);
    }
	free_target_info(ptarget_info_tmp);
	slog_log(slogd , SL_INFO , "<%s> success. line:%d" , __FUNCTION__ , line_no-1);
	fclose(fp);
	return 0;

_failed:
	slog_log(slogd , SL_INFO , "<%s> failed" , __FUNCTION__);
	free_target_info(ptarget_info_tmp);
	fclose(fp);
	return -1;
}

static void demon_proc()
{
	int ret;
	pid_t pid;

	//成为进程组非组长进程[这样不会有get tty的风险]
	pid = fork();
	if(pid < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> 1st fork failed!" , __FUNCTION__);
		exit(-1);
	}
	if(pid > 0)	/*父进程退出*/
	{
		exit(0);	//auto close every opened-fd
	}

	//成为新的会话组长与进程组长
	sleep(1);	//等待一小会 过继到init进程
	ret = setsid();
	if(ret < 0)
	{
		slog_log(slogd  , SL_ERR , "<%s> setsid failed! err:%s" , strerror(errno));
		exit(-1);
	}

	//新会话里的非组长进程 彻底断绝希望
	pid = fork();
	if(pid < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> 2nd fork failed!" , __FUNCTION__);
		exit(-1);
	}
	if(pid > 0)
		exit(0);

	sleep(1);

	//close stdin,stdou,stderr
	close(0);
	close(1);
	close(2);
	slog_log(slogd , SL_ERR , "<%s> success!" , __FUNCTION__);
}

static int print_target_info(target_info_t *ptarget_info)
{
	target_detail_t *ptarget;
	if(!ptarget_info)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! target_info null");
		return -1;
	}

	slog_log(slogd , SL_INFO , "--------TARGET_INFO--------");
	slog_log(slogd , SL_INFO , "[count]:%d" , ptarget_info->target_count);

	/***Print Each*/
	ptarget = ptarget_info->head.next;
	while(ptarget)
	{
		slog_log(slogd , SL_INFO , "<%s> fd:%d proc_id:%d ip:%s port:%d" , ptarget->target_name , ptarget->fd , ptarget->proc_id ,
				ptarget->ip_addr ,	ptarget->port);
		ptarget = ptarget->next;
	}
	slog_log(slogd , SL_INFO , "-----------------------\n");
	return 0;
}

static int free_target_info(target_info_t *ptarget_info)
{
	target_detail_t *ptarget = NULL;
	target_detail_t *ptmp = NULL;
	if(!ptarget_info)
	{
		slog_log(slogd , SL_ERR, "<%s> failed! target_info nil!" , __FUNCTION__);
		return -1;
	}

	ptarget = ptarget_info->head.next;
	while(ptarget)
	{
		ptmp = ptarget->next;
		free(ptarget);
		ptarget = ptmp;
	}

	free(ptarget_info);
	return 0;
}

//init:0:增量更新[dst不为空]; 1:全量覆盖[适用于dst为空列表]
static int copy_target_info(target_info_t *pdst , target_info_t *psrc , char init)
{
	target_detail_t *ptarget = NULL;
	target_detail_t *ptmp = NULL;
	int ret = -1;
	if(!pdst || !psrc)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! src or dst null!" , __FUNCTION__);
		return -1;
	}

	//print
	slog_log(slogd , SL_DEBUG , "=========BEFORE=======");
	slog_log(slogd , SL_DEBUG , "+++++++++DST++++++++++");
	print_target_info(pdst);
	slog_log(slogd , SL_DEBUG , "+++++++++SRC++++++++++");
	print_target_info(psrc);
	slog_log(slogd , SL_DEBUG , "<%s> starts. override:%d" , __FUNCTION__ , init);

	//全量覆盖[初始化的情况]
	if(init)
	{
		//src 全部移植到dst
		ptarget = psrc->head.next;
		while(ptarget)
		{
			//从src取下节点
			psrc->head.next = ptarget->next;
			psrc->target_count--;

			//接入dst
			ptarget->next = pdst->head.next;
			pdst->head.next = ptarget;
			pdst->target_count++;

			//更新target
			ptarget = psrc->head.next;
		}

	}
	else	//增量更新
	{
		//更新src to dst
		ptarget = psrc->head.next;
			//for-each src item
		while(ptarget)
		{
			//1.在dst找对应的entry
			ptmp = proc_id2_target(NULL , pdst , ptarget->proc_id);

			//2.found
			if(ptmp)
			{
				//核心数据是否发生了变化
				if(strcmp(ptmp->ip_addr , ptarget->ip_addr)==0 && ptmp->port==ptarget->port)
					slog_log(slogd , SL_INFO , "[%d]<%s:%d> equal." , ptarget->proc_id , ptarget->ip_addr , ptarget->port);
				else
				{
					slog_log(slogd , SL_INFO , "[%d] changed! src:<%s:%d> will override dst:<%s:%d>" , ptarget->proc_id , ptarget->ip_addr , ptarget->port ,
							ptmp->ip_addr , ptmp->port);


						//关闭原有连接并清除相关数据
					close_target_fd(penv , ptmp , __FUNCTION__ , penv->epoll_fd , 1);

						//update
					memset(ptmp->ip_addr , 0 , sizeof(ptmp->ip_addr));
					strncpy(ptmp->ip_addr , ptarget->ip_addr , sizeof(ptmp->ip_addr));
					ptmp->port = ptarget->port;
				}

				//是否需改名
				if(strcmp(ptmp->target_name , ptarget->target_name) != 0)
				{
					memset(ptmp->target_name , 0 , sizeof(ptmp->target_name));
					strncpy(ptmp->target_name , ptarget->target_name , sizeof(ptmp->target_name));
				}
			}
			else	//3. not found
			{
				slog_log(slogd , SL_INFO , "[%s][%d]<%s:%d> will be added to dst." , ptarget->target_name , ptarget->proc_id , ptarget->ip_addr , ptarget->port);
				//create one
				ptmp = (target_detail_t *)calloc(1 , sizeof(target_detail_t));
				if(!ptmp)
					slog_log(slogd , SL_INFO , "<%s> create new target proc_name:%s proc_id:%d addr:<%s:%d> failed! err:%s" , __FUNCTION__ , ptarget->target_name,
							ptarget->proc_id ,	ptarget->ip_addr , ptarget->port , strerror(errno));
				else
				{

					//set info
					ptmp->fd = -1;
					ptmp->connected = TARGET_CONN_NONE;
					ptmp->proc_id = ptarget->proc_id;
					strncpy(ptmp->target_name , ptarget->target_name , sizeof(ptmp->target_name));
					strncpy(ptmp->ip_addr , ptarget->ip_addr , sizeof(ptmp->ip_addr));
					ptmp->port = ptarget->port;

					//add
					ptmp->next = pdst->head.next;
					pdst->head.next = ptmp;
					pdst->target_count++;
				}

			}

			//4.continue
			ptarget = ptarget->next;
		}

		//删除dst里冗余的target
		ptarget = pdst->head.next;
			//for-each target
		while(ptarget)
		{
			if(!proc_id2_target(NULL , psrc , ptarget->proc_id))	//src 里没有这个项目
			{
				//delete it
				slog_log(slogd , SL_INFO , "<%s> will delete target from dst. [%s:%d]<%s:%d>" , __FUNCTION__ , ptarget->target_name ,
						ptarget->proc_id , ptarget->ip_addr , ptarget->port);

				//close
				close_target_fd(penv , ptarget , "del from dst" , penv->epoll_fd , 1);

				//del it
				if(del_one_target(pdst , ptarget) != 0)
				{
					slog_log(slogd , SL_FATAL , "<%s> del target from dst failed! [%s:%d]<%s:%d>" , __FUNCTION__ , ptarget->target_name ,
							ptarget->proc_id , ptarget->ip_addr , ptarget->port);
				}
				else
				{
					ptmp = ptarget;
					ptarget = ptarget->next;
					free(ptmp);
					continue;
				}

			}

			ptarget = ptarget->next;
		}
	}

	//target地址可能发生变化 重置引用的结构
	del_sending_list(penv);
	clear_hash_map(penv , CR_HASH_MAP_T_TARGET);


	//print
	slog_log(slogd , SL_DEBUG , "=========AFTER=======");
	slog_log(slogd , SL_DEBUG , "+++++++++DST++++++++++");
	print_target_info(pdst);
	slog_log(slogd , SL_DEBUG , "+++++++++SRC++++++++++");
	print_target_info(psrc);

	//重构hash
	ret = init_target_hash_map(penv);
	if(ret < 0)
		slog_log(slogd , SL_ERR , "<%s> init hash failed!" , __FUNCTION__);
	else
	{
		ptarget = pdst->head.next;
		while(ptarget)
		{
			//proc_id
			if(ptarget->proc_id > 0)
				insert_hash_map(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_PROCID , ptarget->proc_id , ptarget);

			//fd
			if(ptarget->fd > 0)
				insert_hash_map(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_FD , ptarget->fd , ptarget);

			ptarget = ptarget->next;
		}

		//print hash_map
		slog_log(slogd , SL_INFO , "!!!!rehash result!!!!");
		dump_hash_map(penv , CR_HASH_MAP_T_TARGET);
	}

	return 0;
}

static int del_one_target(target_info_t *ptarget_info , target_detail_t *ptarget)
{
	target_detail_t *pprev = NULL;

	/**Arg Check*/
	if(!ptarget_info || !ptarget)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! arg null!" , __FUNCTION__);
		return -1;
	}

	/***首节点特殊处理*/
	pprev = ptarget_info->head.next;
	if(pprev && pprev == ptarget)
	{
		ptarget_info->head.next = pprev->next;
		ptarget_info->target_count--;
		ptarget_info->target_count = ptarget_info->target_count<0?0:ptarget_info->target_count;
		return 0;
	}

	/***其他节点*/
	while(pprev)
	{
		if(pprev->next == ptarget)
		{
			pprev->next = ptarget->next;
			ptarget_info->target_count--;
			ptarget_info->target_count = ptarget_info->target_count<0?0:ptarget_info->target_count;
			return 0;
		}

		pprev = pprev->next;
	}

	slog_log(slogd , SL_ERR , "<%s> not found target.[%s:%d]<%s:%d>" , __FUNCTION__ , ptarget->target_name , ptarget->proc_id ,
			ptarget->ip_addr , ptarget->port);
	return -1;
}

static int manage_tick_print(void *arg)
{
	carrier_env_t *penv = (carrier_env_t *)arg;
	print_manage_info(penv);
	return 0;
}

static int check_bridge(void *arg)
{
	struct shmid_ds buff;
	int ret = -1;
	bridge_hub_t *phub = NULL;
	carrier_env_t *penv = (carrier_env_t *)arg;

	if(!penv)
		return -1;
	phub = penv->phub;
	//slog_log(slogd , SL_VERBOSE , "<%s> old attached:%d" , __FUNCTION__ , phub->attached);

	/***Get Read Attach*/
	ret = shmctl(phub->shm_id , IPC_STAT , (struct shmid_ds *)&buff);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> get shm stat failed! err:%s" , __FUNCTION__ , strerror(errno));
		return -1;
	}

	phub->attached = buff.shm_nattch;

	if(phub->attached < 2)
		send_carrier_msg(penv , CR_MSG_ERROR , MSG_ERR_T_UPPER_LOSE , NULL , NULL);
	else
		send_carrier_msg(penv , CR_MSG_EVENT , MSG_EVENT_T_UPPER_RUNNING , NULL , NULL);

	return 0;
}

static int check_client_info(void *arg)
{
	carrier_env_t *penv = (carrier_env_t *)arg;
	client_list_t *pclient_list = penv->pclient_list;
	client_info_t *pclient = NULL;
	long curr_ts = 0;
	int slogd = penv->slogd;

	/***Check Verify*/
	if(pclient_list->total_count <= 0)
		return 0;

_try_del_client:
	pclient = pclient_list->list;
	while(pclient)
	{
		if(!pclient->verify)
		{
			curr_ts = time(NULL);
			if((curr_ts-pclient->connect_time)>10)
			{
				slog_log(slogd , SL_INFO , "<%s> will close client %d<%s:%d> for not verify from %s" , __FUNCTION__ , pclient->fd , pclient->client_ip , pclient->client_port ,
						format_time_stamp(pclient->connect_time));

				free_client_info(pclient);
				goto _try_del_client;
			}
		}
		pclient = pclient->next;
	}

	return 0;
}

static int check_run_statistics(void *arg)
{
	msg_event_stat_t stat_event;
	carrier_env_t *penv = (carrier_env_t *)arg;
	int ret = -1;

	//event
	memset(&stat_event , 0 , sizeof(msg_event_stat_t));

	//fill data
	memcpy(&stat_event.bridge_info , &penv->bridge_info , sizeof(bridge_info_t));

	//some info
	stat_event.bridge_info.send.total_size = penv->phub->send_buff_size;
	stat_event.bridge_info.send.head = penv->phub->send_head;
	stat_event.bridge_info.send.tail = penv->phub->send_tail;
	stat_event.bridge_info.send.handing = penv->phub->sending_count;

	stat_event.bridge_info.recv.total_size = penv->phub->recv_buff_size;
	stat_event.bridge_info.recv.head = penv->phub->recv_head;
	stat_event.bridge_info.recv.tail = penv->phub->recv_tail;
	stat_event.bridge_info.recv.handing = penv->phub->recving_count;

	//send
	ret = send_carrier_msg(penv , CR_MSG_EVENT , MSG_EVENT_T_REPORT_STATISTICS , &stat_event , NULL);
	if(ret < 0)
	{
		slog_log(penv->slogd , SL_ERR , "<%s> send msg failed!" , __FUNCTION__);
		return -1;
	}
	return 0;
}

static int check_signal_stat(void *arg)
{
	carrier_env_t *penv = (carrier_env_t *)arg;
	int ret = -1;
	//check exit
	if(penv->sig_map.sig_exit)
	{
		slog_log(penv->slogd , SL_INFO , "<%s> dectect sig-exit. shutting down..." , __FUNCTION__);
		if(penv->proc_id<=MANAGER_PROC_ID_MAX && penv->pmanager)
			penv->pmanager->stat = MANAGE_STAT_BAD;
		print_manage_info(penv);
		clear_hash_map(penv , CR_HASH_MAP_T_TARGET);
		clear_hash_map(penv , CR_HASH_MAP_T_CLIENT);
		slog_close(penv->slogd);
		exit(0);
		return 0;
	}
	//check reload
	if(penv->sig_map.sig_reload)
	{
		slog_log(slogd , SL_INFO , "-----------------------Begin to Reload Cfg-------------------");
		ret = read_carrier_cfg(&carrier_env , 1);
		slog_log(slogd , SL_INFO , "-----------------------Rload Cfg Finished-------------------");
		send_carrier_msg(&carrier_env , CR_MSG_EVENT , MSG_EVENT_T_RELOAD , &ret , NULL);

		if(ret == 0)
			penv->sig_map.sig_reload = 0;
	}
	return 0;
}

static int check_snd_buff_memory(void *arg)
{
	target_info_t *ptarget_info = &target_info;
	target_detail_t *ptarget = NULL;	
	carrier_env_t *penv = (carrier_env_t *)arg;
	unsigned int new_len = 0;
	char *new_buff = NULL;
	unsigned int should_copy = 0;
	unsigned int data_len = 0;

	if (ptarget_info->target_count <= 0)
		return 0;

	//检查每个target之buff是否需要shrink
	ptarget = ptarget_info->head.next;
	while(ptarget)
	{
		do
		{
			//no buff
			if(!ptarget->snd_buff)
				break;

			//memory < 1M
			if(ptarget->snd_buff_len < (1024*1024))
				break;

			//data_len <= 1/3*buff_len
			data_len = TARGET_DATA_LEN(ptarget);
			if(data_len >= ptarget->snd_buff_len/3)
				break;

			//shrink memory
			new_len = ptarget->snd_buff_len*2/3; //新长度等于原长度的2/3
			slog_log(penv->slogd , SL_INFO , "<%s> try to shrink snd_memory! proc_id:%d proc_name:%s data_len:%d buff_len:%d new_len:%d" , __FUNCTION__ ,
					ptarget->proc_id , ptarget->target_name , data_len , ptarget->snd_buff_len , new_len);
				//new buff
			new_buff = calloc(1 , new_len);
			if(!new_buff)
			{
				slog_log(penv->slogd , SL_ERR , "<%s> alloc memory failed! proc_id:%d proc_name:%s data_len:%d buff_len:%d new_len:%d" , __FUNCTION__ ,
									ptarget->proc_id , ptarget->target_name , data_len , ptarget->snd_buff_len , new_len);
				return -1;
			}

				//empty data
			if(data_len <= 0)
			{
				free(ptarget->snd_buff);
				ptarget->snd_buff = new_buff;
				ptarget->snd_buff_len = new_len;
				break;
			}

				//copy data
			if(ptarget->snd_head < ptarget->snd_tail)
			{
				should_copy = ptarget->snd_tail-ptarget->snd_head;
				memcpy(new_buff , &ptarget->snd_buff[ptarget->snd_head] , should_copy);
				ptarget->snd_head = 0;
				ptarget->snd_tail = should_copy;
			}
			else if(ptarget->snd_head > ptarget->snd_tail)
			{
				should_copy = ptarget->snd_buff_len - ptarget->snd_head;
				memcpy(new_buff , &ptarget->snd_buff[ptarget->snd_head] , should_copy);
				memcpy(&new_buff[should_copy] , &ptarget->snd_buff[0] , ptarget->snd_tail);
				ptarget->snd_head = 0;
				ptarget->snd_tail = should_copy + ptarget->snd_tail;
			}
			else //empty nothing
			{
				//nothing
			}
			free(ptarget->snd_buff);
			ptarget->snd_buff = new_buff;
			ptarget->snd_buff_len = new_len;
			break;
		}
		while(0);

		ptarget = ptarget->next;
	}

	return 0;
}

//保证tmp目录下的文件有效性.系统会定期清除该目录
static int check_tmp_file(void *arg)
{
	int fd = -1;
	int ret = -1;
	char buff[64] = {0};
	carrier_env_t *penv = (carrier_env_t *)arg;

    //rewrite lock file
    lseek(penv->lock_file_fd , 0 , SEEK_SET);

	snprintf(buff , sizeof(buff) , "%-10d" , getpid());
	write(penv->lock_file_fd , buff , strlen(buff));
	slog_log(slogd , SL_INFO , "rewrite %s success!" , penv->lock_file_name);
	

    //rewrite key file
    memset(buff , 0 , sizeof(buff));
	fd = open(penv->key_file_name , O_RDWR|O_TRUNC , 0);
	if(fd < 0)
	{
		slog_log(penv->slogd , SL_ERR , "<%s> open %s failed! err:%s" , __FUNCTION__ , penv->key_file_name , strerror(errno));
		return -1;
	}

	snprintf(buff , sizeof(buff) , "%-10u" , penv->shm_key);
    ret = write(fd , buff , strlen(buff));
	close(fd);

    if(ret < 0)
	{
		slog_log(penv->slogd , SL_ERR , "<%s> write to key file:%s failed! err:%s" , __FUNCTION__ , penv->key_file_name , strerror(errno));
		return -1;		
	} 

    slog_log(penv->slogd , SL_INFO , "<%s> rewrite to key file:%s success! key:%d" , __FUNCTION__ , penv->key_file_name , penv->shm_key);
	return 0;
	
}

//接收来自客户端的pkg
//return:-1:参数错误 -2:缓冲区满且到达上限 0:投递成功
static int recv_client_pkg(carrier_env_t *penv , client_info_t *pclient , bridge_package_t *pkg)
{
	int slogd = -1;
	int ret = 0;
	int pkg_len = 0;
	long curr_ts = 0;

	/***Arg Check*/
	if(!penv || !pclient || !pkg)
		return -1;
	curr_ts = time(NULL);
	slogd = penv->slogd;
	pkg_len = GET_PACK_LEN(pkg->pack_head.data_len);

	/***Try Flush Buff*/
	if(pclient->recv_buffer.tail > 0)
	{
		flush_recving_buff(penv , pclient);
	}

	/***如果buff仍有数据则将数据放入buff中*/
	if(pclient->recv_buffer.tail > 0)
	{
		//检查剩余空间
		if((pclient->recv_buffer.buff_len-pclient->recv_buffer.tail) < pkg_len)
		{
			ret = expand_recv_buff(penv , pclient);
			if(ret != 0)	//无法再扩展缓冲区了，则只能丢包
				return -2;
		}

		//再做一次检查
		if((pclient->recv_buffer.buff_len-pclient->recv_buffer.tail) < pkg_len)
		{
			slog_log(slogd , SL_FATAL , "<%s> fatal error! buff is still not enough after expanding! new_buf_len:%d tail:%d pkg_len:%d" , __FUNCTION__ ,
					pclient->recv_buffer.buff_len , pclient->recv_buffer.tail , pkg_len);
			return -2;
		}

		//投入缓冲区
		if(pclient->recv_buffer.tail <= 0)
			pclient->recv_buffer.delay_starts = curr_ts;
		memcpy(&pclient->recv_buffer.buff[pclient->recv_buffer.tail] , (char *)pkg , pkg_len);

		//update
		pclient->recv_buffer.tail += pkg_len;
		pclient->recv_buffer.max_tail = pclient->tail>pclient->recv_buffer.max_tail?pclient->tail:pclient->recv_buffer.max_tail;
		return 0;
	}

	/***尝试直接投递*/
	ret = append_recv_channel(penv->phub , (char *)pkg , slogd);
	if(ret == 0)	//成功
		return 0;

	/***投递失败则放入缓冲区*/
	//检查剩余空间
	if((pclient->recv_buffer.buff_len-pclient->recv_buffer.tail) < pkg_len)
	{
		ret = expand_recv_buff(penv , pclient);
		if(ret != 0)	//无法再扩展缓冲区了，则只能丢包
			return -2;
	}

	//再做一次检查
	if((pclient->recv_buffer.buff_len-pclient->recv_buffer.tail) < pkg_len)
	{
		slog_log(slogd , SL_FATAL , "<%s> fatal error! buff is still not enough after expanding! new_buf_len:%d tail:%d pkg_len:%d" , __FUNCTION__ ,
				pclient->recv_buffer.buff_len , pclient->recv_buffer.tail , pkg_len);
		return -2;
	}

	//投入缓冲区
	if(pclient->recv_buffer.tail <= 0)
		pclient->recv_buffer.delay_starts = curr_ts;
	memcpy(&pclient->recv_buffer.buff[pclient->recv_buffer.tail] , (char *)pkg , pkg_len);

	//update
	pclient->recv_buffer.tail += pkg_len;
	pclient->recv_buffer.max_tail = pclient->tail>pclient->recv_buffer.max_tail?pclient->tail:pclient->recv_buffer.max_tail;
	return 0;
}

//扩展recv_buff缓冲区
//return:-1 参数错误 -2:到达上限 0:success
static int expand_recv_buff(carrier_env_t *penv , client_info_t *pclient)
{
	int slogd = -1;
	int new_buff_len = 0;
	char *pnew_buff = NULL;

	/***Arg Check*/
	if(!penv || !pclient)
		return -1;
	slogd = penv->slogd;

	/***Check Size*/
	if(pclient->recv_buffer.buff_len >= MAX_CLIENT_BUFF_SIZE)
	{
		slog_log(slogd , SL_ERR , "<%s> failed for buff len max![%d]" , __FUNCTION__ , pclient->recv_buffer.buff_len);
		return -2;
	}

	/***Alloc New*/
	if(pclient->recv_buffer.buff_len <= 0)
		new_buff_len = (BRIDGE_PACK_LEN * 2);	//default
	else
		new_buff_len = pclient->recv_buffer.buff_len += (BRIDGE_PACK_LEN*2);
	pnew_buff = calloc(1 , new_buff_len);
	if(!pnew_buff)
	{
		slog_log(slogd , SL_ERR , "<%s> failed for alloc new. buff len:%d! err:%s" , __FUNCTION__ , new_buff_len , strerror(errno));
		return -2;
	}

	/***Copy*/
	if(pclient->recv_buffer.tail > 0 && pclient->recv_buffer.buff)
		memcpy(pnew_buff , pclient->recv_buffer.buff , pclient->recv_buffer.tail);

	/***Update*/
	if(pclient->recv_buffer.buff)
		free(pclient->recv_buffer.buff);
	pclient->recv_buffer.buff = pnew_buff;
	pclient->recv_buffer.buff_len = new_buff_len;
	return 0;
}


//投递接收缓存的数据到bridge中
//-1:失败
//ELSE 返回刷入的字节
static int flush_recving_buff(carrier_env_t *penv , client_info_t *pclient)
{
	int slogd = -1;
	bridge_package_t *ppkg = NULL;
	int ret;
	long long start_ms = 0;
	long long curr_ms = 0;
	int flush_pos = 0;
	char *buff = NULL;
	int total = 0;
	int pkg_len = 0;

	/***Arg Check*/
	if(!penv || !pclient)
		return -1;
	if(pclient->recv_buffer.tail <= 0 || !pclient->recv_buffer.buff)
		return 0;

	/***Init*/
	start_ms = get_curr_ms();
	slogd = penv->slogd;
	buff = pclient->recv_buffer.buff;

	/***Flush*/
	while(1)
	{
		//1.no more data
		if(flush_pos >= pclient->recv_buffer.tail)
		{
			slog_log(slogd , SL_DEBUG , "<%s> flush complete!" , __FUNCTION__);
			break;
		}

		//2.time overflow
		curr_ms = get_curr_ms();
		if((curr_ms - start_ms) >= 20)	//超过20ms就等待下次再刷[按照内存速度20ms大概可以copy400K]
		{
			slog_log(slogd , SL_DEBUG , "<%s> time overflow. waiting for next." , __FUNCTION__);
			break;
		}

		//3.try post
		ppkg = (bridge_package_t *)&buff[flush_pos];
		pkg_len = GET_PACK_LEN(ppkg->pack_head.data_len);
		ret = append_recv_channel(penv->phub , (char *)ppkg , slogd);
		if(ret == 0)
		{
			total += pkg_len;
			flush_pos += pkg_len;
			slog_log(slogd , SL_DEBUG , "<%s> success! flushed:%d" , __FUNCTION__ , pkg_len);
		}
		else if(ret == -1)	//bridge error
		{
			slog_log(slogd , SL_ERR , "<%s> failed for bridge error!" , __FUNCTION__);
			break;
		}
		else if(ret == -2) //bridge full
		{
			slog_log(slogd , SL_ERR , "<%s> failed for bridge full!" , __FUNCTION__);
			break;
		}

		//continue
	}

	/***Modify*/
	if(flush_pos > 0)	//已经发送成功过则需要移动数据到头部
	{
		memmove(&buff[0] , &buff[flush_pos] , pclient->recv_buffer.tail-flush_pos);
		pclient->recv_buffer.tail = pclient->recv_buffer.tail - flush_pos;
	}

	if(pclient->recv_buffer.tail <= 0)
	{
		pclient->recv_buffer.tail = 0;
		pclient->recv_buffer.delay_starts = 0;
	}
	/***Return*/
	slog_log(slogd , SL_DEBUG , "<%s> flush %d and rest:%d" , __FUNCTION__ , total , pclient->recv_buffer.tail);
	return total;
}

static int check_hash(void *arg)
{
	carrier_env_t *penv = (carrier_env_t *)arg;
	static char print_circle = 0;
	//check
	if(print_circle == 0)
	{
		print_circle++;
		check_target_hash_map(penv);
		check_client_hash_map(penv);
	}
	if(print_circle >= 20)
		print_circle = 0;
	return 0;
}
