/*
 * proc_bridge.c
 *
 *  Created on: 2013-12-22
 *      Author: Administrator
 */
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <getopt.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <sys/time.h>
#include <time.h>
#include <errno.h>
#include "proc_bridge.h"

extern int errno;

/*
 * BRIDGE_ENV
 */
typedef struct _proc_bridge_space_t
{
	int opened_id;
	char name_space[PROC_BRIDGE_NAME_SPACE_LEN];
	int proc_id;
	int slogd;	//slog
	bridge_hub_t *phub;
}proc_bridge_space_t;

#define INIT_PB_SPACE_COUNT 2
typedef struct _proc_bridge_env_t
{
	int space_count;
	int valid_count;
	proc_bridge_space_t *space_list;
}proc_bridge_env_t;
static proc_bridge_env_t proc_bridge_env = {0 , 0 , NULL};

/*************STATIC FUNC************/
static bridge_hub_t *_open_bridge(char *name_space , int proc_id , int slogd);
static int _send_to_bridge(bridge_hub_t *phub , int target_id , char *sending_data , int len , int slogd);
static int _recv_from_bridge(bridge_hub_t *phub , char *recv_buff , int recv_len , int slogd , int *sender , int drop_time);
static int _close_bridge(bridge_hub_t *phub , int slogd);
//get curr ms
static long long _get_curr_ms();
static int get_snd_bit(bridge_hub_t *phub , int id , int sld);
/****************END****************/

/*
 * open bridge
 * @name_space:系统的命名空间
 * @proc_id:服务进程的全局ID
 * @slogd:slog的描述句柄
 * @return:
 * -1:failed
 * >=0:SUCCESS 返回bridge句柄描述符
 */
int open_bridge(char *name_space , int proc_id , int slogd)
{
	proc_bridge_env_t *penv = &proc_bridge_env;
	proc_bridge_space_t *pspace = NULL;
	int new_count = 0;
	proc_bridge_space_t *pnew_list = NULL;
	int i = 0;


	/***Arg Check*/
	if(!name_space || strlen(name_space)<=0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! name_space illegal!" , __FUNCTION__);
		return -1;
	}

	if(proc_id <= 0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! proc_id:%d illegal!" , __FUNCTION__ , proc_id);
		return -1;
	}

	/***Try Init*/
	if(!penv->space_list)
	{
		penv->space_list = calloc(INIT_PB_SPACE_COUNT , sizeof(proc_bridge_space_t));
		if(!penv->space_list)
		{
			slog_log(slogd , SL_ERR , "<%s> failed for alloc memory! err:%s" , __FUNCTION__ , strerror(errno));
			return -1;
		}
		penv->space_count = INIT_PB_SPACE_COUNT;
	}

	/***Search Existed*/
	for(i=0; i<penv->space_count; i++)
	{
		pspace = &penv->space_list[i];
		if(pspace->proc_id==proc_id && strcmp(pspace->name_space , name_space)==0)
		{
			slog_log(slogd , SL_INFO , "<%s> already opened at %d! <%s:%d>" , __FUNCTION__ , i , name_space , proc_id);
			return i;
		}
	}

	/***Wheter Expand*/
	if(penv->valid_count >= penv->space_count)
	{
		new_count = penv->space_count * 2;
		slog_log(slogd , SL_INFO , "<%s> will try to expand space list from %d --> %d" , __FUNCTION__ , penv->space_count , new_count);

		//1.alloc new list
		pnew_list = calloc(new_count , sizeof(proc_bridge_space_t));
		if(!pnew_list)
		{
			slog_log(slogd , SL_ERR , "<%s> failed for alloc new list %d-->%d memory! err:%s" , __FUNCTION__ , penv->space_count ,
					new_count , strerror(errno));
			return -1;
		}

		//2.copy old
		memcpy(pnew_list , penv->space_list , penv->space_count * sizeof(proc_bridge_space_t));

		//3. free old
		free(penv->space_list);

		//4.add new
		penv->space_count = new_count;
		penv->space_list = pnew_list;
	}

	/***Get An Empty Pos*/
	for(i=0; i<penv->space_count; i++)
	{
		if(penv->space_list[i].proc_id == 0 && strlen(penv->space_list[i].name_space)==0)
			break;
	}
	if(i >= penv->space_count)
	{
		slog_log(slogd , SL_ERR , "<%s> can not find an empty pos! space_count:%d valid:%d" , __FUNCTION__ , penv->space_count , penv->valid_count);
		return -1;
	}
	pspace = &penv->space_list[i];

	/***Attach*/
	pspace->phub = _open_bridge(name_space , proc_id , slogd);
	if(!pspace->phub)
	{
		slog_log(slogd , SL_ERR , "<%s> attach bridge <%s:%d> failed!" , __FUNCTION__ , name_space , proc_id);
		return -1;
	}

	/***Set Info*/
	pspace->opened_id = i;
	pspace->proc_id = proc_id;
	pspace->slogd = slogd;
	strncpy(pspace->name_space , name_space , PROC_BRIDGE_NAME_SPACE_LEN);
	penv->valid_count++;
	slog_log(slogd , SL_INFO , "<%s> at %d success! space_count:%d valid_count:%d" , __FUNCTION__ , i , penv->space_count , penv->valid_count);

	return i;
}

int close_bridge(int bd)
{
	proc_bridge_env_t *penv = &proc_bridge_env;
	proc_bridge_space_t *pspace = NULL;

	/***Arg Check*/
	if(bd < 0 )
	{
		return -1;
	}

	/***Get Space*/
	if(penv->space_count<=0 || penv->valid_count<=0 || bd>=penv->space_count)
	{
		return -1;
	}
	pspace = &penv->space_list[bd];

	/***Check Space*/
	if(pspace->opened_id != bd)
	{
		return -1;
	}

	/***Close*/
	_close_bridge(pspace->phub , pspace->slogd);
	penv->valid_count--;
	slog_log(pspace->slogd , SL_INFO , "<%s> close %d success!" , __FUNCTION__ , bd);
	memset(pspace , 0 , sizeof(proc_bridge_space_t));

	/***Check Valid*/
	if(penv->valid_count <= 0)
	{
		free(penv->space_list);
		penv->space_count = 0;
		penv->valid_count = 0;
		penv->space_list = NULL;
	}

	return 0;
}

/*
 * send_to_bridge
 * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @target_id:目标服务进程的全局ID
 * @sending_data:发送的数据
 * @len:发送数据长度
 * @slogd:slog的描述句柄
 * @return:
 * -1：错误
 * -2：发送缓冲区满
 * -3：发送数据超出包长
 * 0：成功
 */
int send_to_bridge(int bd , int target_id , char *sending_data , int len)
{
	proc_bridge_env_t *penv = &proc_bridge_env;
	proc_bridge_space_t *pspace = NULL;

	/***Arg Check*/
	if(bd < 0 || target_id<=0 || !sending_data || len<=0)
	{
		//slog_log(slogd , SL_ERR , "<%s> failed! arg error. bridge_descriptor:%d , target_id:%d,sending:%X or len:%d illegal!" , __FUNCTION__ , bd , target_id ,
		//		sending_data , len);
		return -1;
	}

	/***Get Space*/
	if(penv->space_count<=0 || penv->valid_count<=0 || bd>=penv->space_count)
	{
		//slog_log(slogd , SL_ERR , "<%s> failed! bd illegal! space_count:%d valid:%d bd:%d" , __FUNCTION__ , penv->space_count , penv->valid_count ,
		//		bd);
		return -1;
	}
	pspace = &penv->space_list[bd];

	/***Check Space*/
	if(pspace->opened_id != bd)
	{
		slog_log(pspace->slogd , SL_ERR , "<%s> failed! descriptor not match! %d != %d" , __FUNCTION__ , pspace->opened_id , bd);
		return -1;
	}

	//check len
	/*
	if(len > BRIDGE_PACKAGE_DATA_LEN)
	{
		slog_log(pspace->slogd , SL_ERR , "<%s> failed! data len:%d is bigger than predefined package-data-len:%d" , __FUNCTION__ , len , BRIDGE_PACKAGE_DATA_LEN);
		return -1;
	}*/


	return _send_to_bridge(pspace->phub , target_id , sending_data , len , pspace->slogd);
}


/*
 * recv_from_bridge
 * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @target_id:目标服务进程的全局ID
 * @recv_buff:接收数据缓冲区
 * @slogd:slog的描述句柄
 * @return:
 * -1：错误
 * -2：接收缓冲区空
 * -3：接收数据超出包长
 * else:发送者proc_id
 */
/*
 * recv_from_bridge
 * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @target_id:目标服务进程的全局ID
 * @recv_buff:接收数据缓冲区
 * @recv_len:接收缓冲区长度
 * @sender:sender proc_id if not null
 * @drop_time: >=0丢弃发送时间超过drop_time(秒)的包; -1:不丢弃任何包
 * @return:
 * -1：错误
 * -2：接收缓冲区空
 * -3：接收数据超出包长
 * else:实际接收的长度
 */
int recv_from_bridge(int bd , char *recv_buff , int recv_len , int *sender , int drop_time)
{
	proc_bridge_env_t *penv = &proc_bridge_env;
	proc_bridge_space_t *pspace = NULL;

	/***Arg Check*/
	if(bd < 0 || !recv_buff || recv_len<= 0)
	{
		return -1;
	}

	/***Get Space*/
	if(penv->space_count<=0 || penv->valid_count<=0 || bd>=penv->space_count)
	{
		return -1;
	}

	pspace = &penv->space_list[bd];
	/***Check Space*/
	if(pspace->opened_id != bd)
	{
		return -1;
	}

	return _recv_from_bridge(pspace->phub , recv_buff , recv_len , pspace->slogd , sender , drop_time);
}

/*
 * 根据bridge_descriptor返回对应的bridge_hub指针
 * @return:NULL failed. ELSE success
 */
bridge_hub_t *bd2bridge(int bd)
{
	proc_bridge_env_t *penv = &proc_bridge_env;
	bridge_hub_t *phub = NULL;
	/***Arg Check*/
	if(bd < 0)
		return NULL;

	if(penv->valid_count <= 0)
		return NULL;

	/***Handle*/
	if(bd >= penv->space_count)
		return NULL;

	phub = penv->space_list[bd].phub;
	return phub;
}

//static int send_proc_id;
/*
 * open bridge
 * @name_space:系统的命名空间
 * @proc_id:服务进程的全局ID
 * @slogd:slog的描述句柄
 * @return:
 * NULL:failed
 * else:SUCCESS
 */
static bridge_hub_t *_open_bridge(char *name_space , int proc_id , int slogd)
{
	bridge_hub_t *pbridge_hub;
	int shm_id;
	int shm_key;

	/***Arg Check*/
	if(proc_id <= 0)
	{
		slog_log(slogd , SL_ERR , "<%s> Error: proc_id illegal!" , __FUNCTION__);
		return NULL;
	}
	if(!name_space || strlen(name_space)<=0)
	{
		slog_log(slogd , SL_ERR , "<%s> Error: name_space null!" , __FUNCTION__);
		return NULL;
	}

	/***Attach*/
	//shm_key = SHM_CREATE_MAGIC + proc_id;
	shm_key = get_bridge_shm_key(name_space , proc_id , 0 , slogd);
	if(shm_key < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed for get shm_key! name_space:%s proc_id:%d" , __FUNCTION__ , name_space , proc_id);
		return NULL;
	}

	//GET SHM
	shm_id = shmget(shm_key , 0 , 0);
	if(shm_id < 0)
	{
		slog_log(slogd , SL_ERR , "%s Error: shmget bridge of %d  failed! err:%s" , __FUNCTION__ , proc_id , strerror(errno));
		return NULL;
	}

	//attach
	pbridge_hub = (bridge_hub_t *)shmat(shm_id , NULL , 0);
	if(!pbridge_hub)
	{
		slog_log(slogd , SL_ERR , "%s Error: shmat bridge of %d failed! err:%s" , __FUNCTION__ , proc_id , strerror(errno));
		return NULL;
	}

	/***Check SHM*/
	//proc_id
	if(pbridge_hub->proc_id != proc_id)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! proc_id is wrong! proc_id:%d shm_proc_id:%d" , __FUNCTION__ , proc_id , pbridge_hub->proc_id);
		return NULL;
	}

	//send buff
	if(pbridge_hub->send_buff_size <= 0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! send_buff_size illegal! proc_id:%d send_size:%d" , __FUNCTION__ , proc_id , pbridge_hub->send_buff_size);
		return NULL;
	}

	//recv buff
	if(pbridge_hub->recv_buff_size <= 0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! recv_buff_size illegal! proc_id:%d recv_size:%d" , __FUNCTION__ , proc_id , pbridge_hub->recv_buff_size);
		return NULL;
	}

	pbridge_hub->attached++;
	slog_log(slogd , SL_INFO , "<%s> recv_size:%d send_size:%d attach:%d" , __FUNCTION__ , pbridge_hub->recv_buff_size , pbridge_hub->send_buff_size ,
			pbridge_hub->attached);
	//set
	//send_proc_id = proc_id;	/*这里记录发送者ID*/

	slog_log(slogd , SL_INFO , "<%s> at %d success! buff_start:%lx send_channel:%lx recv_channel:%lx" , __FUNCTION__ ,proc_id ,
			pbridge_hub->all_buff , GET_SEND_CHANNEL(pbridge_hub),	GET_RECV_CHANNEL(pbridge_hub));
	return pbridge_hub;
}

/*
 * close bridge_hub of process
 * @return -1:failed 0:success
 *
 */
static int _close_bridge(bridge_hub_t *phub , int slogd)
{
	/*
	if(!phub)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! phub null!" , __FUNCTION__);
		return -1;
	}

	slog_log(slogd , SL_INFO , "<%s> success! proc_id:%d" , __FUNCTION__ , phub->proc_id);
	phub->attached--;*/
	return 0;
}


/*
 * send_to_bridge
 * send受到两个因素影响：是否满；是否有剩余空间（head指针位置）。
 * @phub:该进程打开的bridge
 * @target_id:目标服务进程的全局ID
 * @sending_data:发送的数据
 * @len:发送数据长度
 * @slogd:slog的描述句柄
 * @return:
 * -1：错误
 * -2：发送缓冲区满
 * -3：发送数据超出包长
 * 0：成功
 */
static int _send_to_bridge(bridge_hub_t *phub , int target_id , char *sending_data , int len , int slogd)
{
	char buff[sizeof(bridge_package_head_t)] = {0};
	char *send_buff = NULL;
	bridge_package_t *pstpack = NULL;
	int empty_space = 0;
	int send_count = 0;
	int total_count = 0;
	int copyed = 0;

	int send_head = 0;
	int send_tail = 0;

	int tail_pos;
#ifdef _TRACE_DEBUG
	char test_buff[_TRACE_DEBUG_BUFF_LEN] = {0};
#endif
	/***Arg Check*/
	if(!phub || target_id<= 0 || !sending_data || len<=0)
	{
		slog_log(slogd , SL_ERR , "%s failed. arg nil! target:%d send:%lX len:%d" , __FUNCTION__ , target_id , sending_data , len);
		return -1;
	}

	/***发送*/
	//1.检查发送位图是否允许发送
	if(get_snd_bit(phub , target_id , slogd) != 0)
	{
		slog_log(slogd , SL_ERR , "%s failed! snd_bit_map is set! not allowed to send pkg yet!" , __FUNCTION__ );
		return -3;
	}

	send_buff = GET_SEND_CHANNEL(phub);

	//memset(buff , 0 , sizeof(buff));
	//2.检查空闲空间
	total_count = sizeof(buff) + len;
    send_head = phub->send_head;
    send_tail = phub->send_tail;

	/*tail 在head之后，则考察tail<->end + start<->head的长度*/
	if(send_tail >= send_head)
	{
		//检查空闲空间
		empty_space = phub->send_buff_size - send_tail + send_head;
	}
	else	/*tail在head之前，则考察tail<->head*/
	{
		empty_space = send_head - send_tail;
	}
	if(empty_space-1 < total_count)	//预留1B 防止head==tail&&equeue == full
	{
		slog_log(slogd , SL_ERR , "%s failed for not enough space. left space:%d pack_len:%d" , __FUNCTION__ , empty_space , total_count);
		return -3;
	}
	//3.COPY数据到缓冲区,最开始为头部
	pstpack = (bridge_package_t *)buff;
	//memcpy(pstpack->pack_data , sending_data , len);

	//pstpack->pack_head.sender_id = send_proc_id;
	pstpack->pack_head.sender_id = phub->proc_id;
	pstpack->pack_head.recver_id = target_id;
	pstpack->pack_head.data_len = len;
	pstpack->pack_head.send_ms = _get_curr_ms();

	//5.从buff拷贝头部
	tail_pos = send_tail;
	send_count = sizeof(buff);
	/*tail 在head之后，则考察tail<->end + start<->head的长度*/
	if(tail_pos >= send_head)
	{
		//最后剩余空间足够
		if((phub->send_buff_size - tail_pos) >= send_count)
		{
			memcpy(&send_buff[tail_pos] , buff , send_count);
			tail_pos += send_count;
			tail_pos %= phub->send_buff_size;
		}
		else//最后剩余空间不够
		{
			copyed = phub->send_buff_size - tail_pos;
			memcpy(&send_buff[tail_pos] , buff , copyed);
			memcpy(&send_buff[0] , &buff[copyed] , send_count - copyed);
			tail_pos = 0 + send_count - copyed;
			slog_log(slogd , SL_DEBUG , "%s head-checkpoint![%d<-->%d](%d:%d)" , __FUNCTION__ , send_head , send_tail , tail_pos , send_count);
		}

	}
	else	/*tail在head之前，则考察tail<->head*/
	{
		memcpy(&send_buff[tail_pos] , buff , send_count);
		tail_pos += send_count;
	}

	//copy body
	send_count = len;
	/*tail 在head之后，则考察tail<->end + start<->head的长度*/
	if(tail_pos >= send_head)
	{
		//最后剩余空间足够
		if((phub->send_buff_size - tail_pos) >= send_count)
		{
			memcpy(&send_buff[tail_pos] , sending_data , send_count);
			tail_pos += send_count;
			tail_pos %= phub->send_buff_size;
		}
		else//最后剩余空间不够
		{
			copyed = phub->send_buff_size - tail_pos;
			memcpy(&send_buff[tail_pos] , sending_data , copyed);
			memcpy(&send_buff[0] , &sending_data[copyed] , send_count - copyed);
			tail_pos = 0 + send_count - copyed;
			slog_log(slogd , SL_DEBUG , "%s body-checkpoint![%d<-->%d](%d:%d)" , __FUNCTION__ , send_head , send_tail , tail_pos , send_count);
		}

	}
	else	/*tail在head之前，则考察tail<->head*/
	{
		memcpy(&send_buff[tail_pos] , sending_data , send_count);
		tail_pos += send_count;
	}


	//6.在写完内存之后再修改tail指针，因为read会比较tail指针位置。否则会出现同步错误
	/*这里可能会有同步错误 如果先走到226 然后断住 另一方会发现可读 然后修改send_head==send_tail 此时该进程再继续
	 * 然后会赋值send_full==1这是有错误的
	phub->send_tail = tail_pos;
	if(phub->send_head == phub->send_tail)
		phub->send_full = 1;
	*/
	/*bugfix 2 取消full标记
	if(phub->send_head == tail_pos)
		phub->send_full = 1;
	*/

#ifdef _TRACE_DEBUG
	memcpy(test_buff , sending_data , len);
	slog_log(slogd , SL_DEBUG , "[%d<-->%d](%d:%d)%s" , send_head , send_tail , tail_pos , len , test_buff);
#endif

	phub->send_tail = tail_pos;
	phub->sending_count++;

	phub->sended_count++;
	if(phub->sended_count >= 0xFFFFF000) //left 2^12
		phub->sended_count = 0;
	return 0;
}



/*
 * recv_from_bridge
 * send受到两个因素影响：是否满；是否有数据可读（tail指针位置）。
 * @phub:该进程打开的bridge
 * @target_id:目标服务进程的全局ID
 * @recv_buff:接收数据缓冲区
 * @recv_len:接收缓冲区长度
 * @slogd:slog的描述句柄
 * @sender:发送者proc_id if not null
 * @drop_time: >=0丢弃发送时间超过drop_time(秒)的包; -1:不丢弃任何包
 * @return:
 * -1：错误
 * -2：接收缓冲区空
 * -3：接收数据超出包长
 * else:实际接收的长度
 */
static int _recv_from_bridge(bridge_hub_t *phub , char *recv_buff , int recv_len , int slogd , int *sender , int drop_time)
{
	char *recv_channel = NULL;
	bridge_package_t *pstpack;
	char buff[sizeof(bridge_package_t)] = {0};
	int copyed = 0;

	int head_pos = 0;
	int tail_pos = 0;
	int channel_len = 0;
	int pack_len = 0;
	int data_len = 0;
	char should_copy = 1;
	long curr_ts = time(NULL);

#ifdef _TRACE_DEBUG
	char test_buff[_TRACE_DEBUG_BUFF_LEN] = {0};
	int start_read = 0;
#endif

	/***Arg Check*/
	if(!phub ||!recv_buff)
	{
		slog_log(slogd , SL_ERR ,"%s failed for illegal arg!" , __FUNCTION__);
		return -1;
	}

	recv_channel = GET_RECV_CHANNEL(phub);
	head_pos = phub->recv_head;
	tail_pos = phub->recv_tail;

_recv_again:
	/***接收*/
	//1.检查接收区是否有数据
	if(head_pos == tail_pos)
	{
		return -2;
	}


	//接收时不做包结构完整性检查，默认缓冲区里都是结构完整的包，这是由send时保证
	channel_len = phub->recv_buff_size;
	pack_len = sizeof(bridge_package_t);
	//head_pos = phub->recv_head;

#ifdef _TRACE_DEBUG
	slog_log(slogd , SL_DEBUG , "%s [%d<-->%d](%d:%d)" , __FUNCTION__ , head_pos , tail_pos , channel_len , pack_len);
	start_read = head_pos;
#endif

	//2.先读取头部区
	if((channel_len - head_pos) < pack_len)	/*余下不足头部*/
	{
		copyed = channel_len - head_pos;
		memcpy(buff , &recv_channel[head_pos] , copyed);
		memcpy(&buff[copyed] , &recv_channel[0] , pack_len-copyed);
		head_pos = 0 + pack_len - copyed;
	}
	else	/*余下空间可以放下头部*/
	{
		memcpy(buff , &recv_channel[head_pos] , pack_len);
		head_pos += pack_len;
		head_pos %= channel_len;
	}

	//3.获得头部后
	pstpack = (bridge_package_t *)buff;
	data_len = pstpack->pack_head.data_len;
	should_copy = 1;
	if(drop_time>=0 && ((curr_ts-(long)(pstpack->pack_head.send_ms/1000))>=drop_time))
	{
		slog_log(slogd , SL_INFO , "<%s> will drop package out drop_time:%d curr:%ld send:%lld" , __FUNCTION__ , drop_time , curr_ts ,
				pstpack->pack_head.send_ms);
		should_copy = 0;
	}

	//3.5检查缓冲区长度
	if(data_len > recv_len && should_copy)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! data len:%d is bigger than recv_len:%d" , __FUNCTION__ , data_len , recv_len);
		return -3;
	}

	//4.读取数据
	if((channel_len - head_pos) < data_len)	/*余下不足放数据*/
	{
		copyed = channel_len - head_pos;
		if(should_copy)
		{
			memcpy(recv_buff , &recv_channel[head_pos] , copyed);
			memcpy(&recv_buff[copyed] , &recv_channel[0] , data_len-copyed);
		}
		head_pos = 0 + data_len - copyed;
	}
	else	/*余下可以放下数据*/
	{
		if(should_copy)
			memcpy(recv_buff , &recv_channel[head_pos] , data_len);
		head_pos += data_len;
		head_pos %= channel_len;
	}

#ifdef _TRACE_DEBUG
	memcpy(test_buff , recv_buff , data_len);
	test_buff[data_len] = 0;
	slog_log(slogd , SL_DEBUG , "%s [%d<-->%d]<%d , %d>(%d:%d)%s" , __FUNCTION__ , head_pos , tail_pos , start_read , head_pos ,
			channel_len , data_len+pack_len , test_buff);
#endif

	//4.在读完该内存之后再修改位置指针，因为write会比较head指针位置.否则会出现同步错误
	phub->recving_count--;
	phub->recv_head = head_pos;

	//5.本次未发生数据拷贝则读取下一包
	if(!should_copy)
		goto _recv_again;

	if(sender)
		*sender = pstpack->pack_head.sender_id;
	return data_len;
}

/*
 * 根据命名空间和proc_id获得对应的shm key
 * @-1:failed else:key
 */
int get_bridge_shm_key_old(char *name_space , int proc_id , int creater , int slogd)
{
	struct stat file_stat;
	char dir_name[128] = {0};
	char file_path[256] = {0};
	int ret = -1;
	int fd = -1;
	int my_key = 0;
	/***Arg Check*/
	if(!name_space || proc_id<=0)
		return -1;

	slog_log(slogd , SL_INFO , "<%s> name_space:%s proc_id:%d creater:%d" , __FUNCTION__ , name_space , proc_id , creater);
	/***Dir And Path*/
	snprintf(dir_name , sizeof(dir_name) , PROC_BRIDGE_HIDDEN_DIR_FORMAT , name_space);
	//snprintf(file_path , sizeof(file_path) , "%s/carrier.%d.key" , dir_name , proc_id);
    snprintf(file_path , sizeof(file_path) , PROC_BRIDGE_HIDDEN_DIR_FORMAT"/"PROC_BRIDGE_HIDDEN_KEY_FILE , name_space , proc_id);

	/***Try Create Dir*/
	if(creater)
	{
		//dir
		ret = mkdir(dir_name , 0755);
		if(ret < 0)
		{
			do
			{
				if(errno == EEXIST)
				{
					slog_log(slogd , SL_DEBUG , "<%s> dir:%s is existed!" , __FUNCTION__ , dir_name);
					break;
				}
				else
				{
					slog_log(slogd , SL_ERR , "<%s> create dir:%s failed! err:%s" , __FUNCTION__ , dir_name , strerror(errno));
					return -1;
				}
			}
			while(0);
		}

	}

	/***Open File*/
	if(creater)
	{
		fd = open(file_path , O_RDONLY|O_CREAT , 0544);	//should not change
		if(fd < 0)
		{
			slog_log(slogd , SL_ERR , "<%s> open file:%s when try create.failed for %s" , __FUNCTION__ , file_path , strerror(errno));
			return -1;
		}
	}
	else
	{
		fd = open(file_path , O_RDONLY , 0);
		if(fd < 0)
		{
			slog_log(slogd , SL_ERR , "<%s> open file:%s when no create.failed for %s" , __FUNCTION__ , file_path , strerror(errno));
			return -1;
		}
	}

	/***STAT*/
	ret = fstat(fd , &file_stat);
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> stat file:%s filed for %s" , __FUNCTION__ , file_path , strerror(errno));
		return -1;
	}

	//combine key
	my_key = my_key | ((file_stat.st_mtime&0x007F0000) << 8);
	my_key = my_key | ((file_stat.st_ino & 0x0000FFFF) << 8);
	my_key = my_key | (proc_id & 0x000000FF);
	slog_log(slogd , SL_INFO , "<%s> file_path:%s mtime:0x%lX ino:0x%lX proc_id:0x%X key is:0x%X" , __FUNCTION__ , file_path ,
			file_stat.st_mtime , file_stat.st_ino , proc_id , my_key);
	close(fd);
	return my_key;
}



/*
 * 根据命名空间和proc_id获得对应的shm key
 * @-1:failed else:key
 */
int get_bridge_shm_key(char *name_space , int proc_id , int creater , int slogd)
{
	struct stat file_stat;
	char dir_name[128] = {0};
	char file_path[256] = {0};
	char buff[12] = {0};
	int ret = -1;
	int fd = -1;
	int my_key = 0;
	/***Arg Check*/
	if(!name_space || proc_id<=0)
		return -1;

	slog_log(slogd , SL_INFO , "<%s> name_space:%s proc_id:%d creater:%d" , __FUNCTION__ , name_space , proc_id , creater);
	/***Dir And Path*/
	snprintf(dir_name , sizeof(dir_name) , PROC_BRIDGE_HIDDEN_DIR_FORMAT , name_space);
	//snprintf(file_path , sizeof(file_path) , "%s/carrier.%d.key" , dir_name , proc_id);
    snprintf(file_path , sizeof(file_path) , PROC_BRIDGE_HIDDEN_DIR_FORMAT"/"PROC_BRIDGE_HIDDEN_KEY_FILE , name_space , proc_id);

	/***Create*/
	if(creater)
	{
	    //dir
		ret = mkdir(dir_name , 0755);
		if(ret < 0)
		{
			do
			{
				if(errno == EEXIST)
				{
					slog_log(slogd , SL_DEBUG , "<%s> dir:%s is existed!" , __FUNCTION__ , dir_name);
					break;
				}
				else
				{
					slog_log(slogd , SL_ERR , "<%s> create dir:%s failed! err:%s" , __FUNCTION__ , dir_name , strerror(errno));
					return -1;
				}
			}
			while(0);
		}

        // file
        fd = open(file_path , O_RDWR|O_CREAT , 0644);	//
		if(fd < 0)
		{
			slog_log(slogd , SL_ERR , "<%s> open file:%s when try create failed for %s" , __FUNCTION__ , file_path , strerror(errno));
			return -1;
		}

        // generate key
	    ret = fstat(fd , &file_stat);
	    if(ret < 0)
	    {
		    slog_log(slogd , SL_ERR , "<%s> stat file:%s filed for %s" , __FUNCTION__ , file_path , strerror(errno));
		    return -1;
	    }

	    //combine key
	    my_key = my_key | ((file_stat.st_mtime&0x007F0000) << 8);
	    my_key = my_key | ((file_stat.st_ino & 0x0000FFFF) << 8);
	    my_key = my_key | (proc_id & 0x000000FF);	    
        
		//write
        snprintf(buff , sizeof(buff) , "%-10u" , my_key);
        ret = write(fd , buff , strlen(buff));
        if(ret < 0)
		{
			slog_log(slogd , SL_ERR , "<%s> write to key file:%s failed! err:%s" , __FUNCTION__ , file_path , strerror(errno));
			return -1;
		}
		slog_log(slogd , SL_INFO , "<%s> create file_path:%s mtime:0x%lX ino:0x%lX proc_id:0x%X key is:0x%X" , __FUNCTION__ , file_path ,
		    file_stat.st_mtime , file_stat.st_ino , proc_id , my_key);
		close(fd);
        return my_key;
	}

	/***Get*/
	fd = open(file_path , O_RDONLY , 0);
	if(fd < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> open file:%s when no create.failed for %s" , __FUNCTION__ , file_path , strerror(errno));
		return -1;
	}
	
	ret = read(fd , buff , sizeof(buff));
	if(ret < 0)
	{
		slog_log(slogd , SL_ERR , "<%s> read  key file:%s failed! err:%s" , __FUNCTION__ , file_path , strerror(errno));
		close(fd);
		return -1;
	}

    my_key = atoi(buff);
	slog_log(slogd , SL_INFO , "<%s> get key_file:%s key:0x%X" , __FUNCTION__ , file_path , my_key);
	close(fd);
	return my_key;
}


//get curr ms
static long long _get_curr_ms()
{
	int ret = -1;
	struct timeval tv;
	ret = gettimeofday(&tv , NULL);
	if(ret < 0)
		return -1;
	return ((long long)tv.tv_sec*1000)+tv.tv_usec/1000;
}

//get snd bitmap
static int get_snd_bit(bridge_hub_t *phub , int id , int sld)
{
	if(!phub)
		return -1;

	char *bmap = phub->snd_bitmap;
	int len = sizeof(phub->snd_bitmap);
	int byte_seq = 0;
	int bit_seq = 0;
	char v = 1;

	//[0 , len*8-1]
	if(id >= len*8)
	{
		slog_log(sld , SL_ERR , "%s id too big! id:%d\n" , __FUNCTION__ , id);
		return -1;
	}

	byte_seq = id / 8;
	bit_seq = id % 8;
	v = bmap[byte_seq] & (v << bit_seq);
	return (v>0)?1:0;
}
