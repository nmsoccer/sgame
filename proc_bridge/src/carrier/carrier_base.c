/*
 * carrier_base.c
 *
 *  Created on: 2019年3月25日
 *      Author: nmsoccer
 */
#include <slog/slog.h>
#include <errno.h>
#include <string.h>
#include <stdlib.h>
#include <sys/time.h>
#include "proc_bridge.h"
#include "carrier_base.h"

extern int errno;

/*
 * Hash Map
 * 用来设置hash_entry_list_size
 */
static int hash_size_map[] = { //7 , 13 , 19 , 27 , 37 ,
		61, 113 , 211 , 379 , 509 , 683 , 911 , //<1K
		1217 , 1627 , 2179 , 2909 , 3881 , 6907 , 9209, //<10K
		12281 , 16381 , 21841 , 29123 , 38833 , 51787 , 69061 , 92083, //<100K
		122777,163729,218357,291143,388211,517619,690163,999983, //<1M
		1226959 , 1635947 , 2181271 , 2908361 , 3877817 , 5170427,6893911,9191891, //<10M
		12255871 , 16341163,21788233,29050993,38734667,51646229,68861641,91815541, //<100M
};

/*
 * 添加一个ticker
 * -1:failed 0:success
 */
int append_carrier_ticker(carrier_env_t *penv , CARRIER_TICK func_tick , char type , long long tick_period ,
		char *ticker_name , void *arg)
{
	int slogd = -1;
	time_ticker_t *pticker = NULL;
	long long expire_ms = 0;
	if(!penv || !func_tick)
		return -1;
	slogd = penv->slogd;

	/***Alloc*/
	pticker = calloc(1 , sizeof(time_ticker_t));
	if(!pticker)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! alloc ticker failed! err:%s" , __FUNCTION__ , strerror(errno));
		return -1;
	}
	expire_ms = get_curr_ms() + tick_period;

	pticker->type = type;
	pticker->tick_period = tick_period;
	pticker->expire_ms = expire_ms;
	if(ticker_name)
		strncpy(pticker->ticker_name , ticker_name , sizeof(pticker->ticker_name));
	pticker->func = func_tick;
	pticker->arg = arg;

	/***Append*/
	pticker->next = penv->tick_list.head.next;
	penv->tick_list.head.next = pticker;
	pticker->prev = &penv->tick_list.head;
	if(pticker->next)
		pticker->next->prev = pticker;

	if(penv->tick_list.latest_expire_ms == 0)
		penv->tick_list.latest_expire_ms = expire_ms;
	else
		penv->tick_list.latest_expire_ms = (expire_ms<penv->tick_list.latest_expire_ms)?expire_ms:penv->tick_list.latest_expire_ms;

	penv->tick_list.count++;
	slog_log(slogd , SL_INFO , "<%s> success! count:%d ticker:%s type:%s period:%lld trig:%lld latest_expire_ms:%lld" , __FUNCTION__ , penv->tick_list.count ,
			pticker->ticker_name , type==1?"singe":"circle",tick_period , expire_ms , penv->tick_list.latest_expire_ms);
	return 0;
}

int iter_time_ticker(carrier_env_t *penv)
{
	tick_list_t *ptick_list = NULL;
	time_ticker_t *pticker = NULL;
	time_ticker_t *ptmp = NULL;
	long long curr_ms = get_curr_ms();
	long long new_latest_expire = 0;
	int slogd = -1;

	slogd = penv->slogd;
	ptick_list = &penv->tick_list;

	/***Expired*/
	if(ptick_list->count<=0 || ptick_list->latest_expire_ms > curr_ms)
		return 0;

	/***EXE TICKER*/
	new_latest_expire = 0x7FFFFFFFFFFLL;
	pticker = ptick_list->head.next;
	while(pticker)
	{
		//expired
		if(pticker->expire_ms <= curr_ms)
		{
			//exe ticker
			pticker->func(pticker->arg);

			//SINGLE-SHOT
			if(pticker->type == TIME_TICKER_T_SINGLE_SHOT)
			{
				//try to destroy
				ptmp = pticker->next;
				pticker->prev->next = ptmp;
				if(ptmp)
					ptmp->prev = pticker->prev;
				ptick_list->count--;

				slog_log(slogd , SL_INFO , "<%s> destroy ticker:%s type:%d count:%d" , __FUNCTION__ , pticker->ticker_name , pticker->type ,
						ptick_list->count);
				free(pticker);
				pticker = ptmp;
				continue;
			}
			else	//CIRCLE
			{
				pticker->expire_ms = curr_ms + pticker->tick_period;
				if(pticker->expire_ms < new_latest_expire)
					new_latest_expire = pticker->expire_ms;

				slog_log(slogd , SL_VERBOSE , "<%s> expired tricker %s resets to %lld and new_latest_expire:%lld" , __FUNCTION__ , pticker->ticker_name ,
						pticker->expire_ms , new_latest_expire);
			}

		}
		else	//no expire
		{
			if(pticker->expire_ms < new_latest_expire)
				new_latest_expire = pticker->expire_ms;

			slog_log(slogd , SL_VERBOSE , "<%s> no expire ticker:%s expired at %lld and  new_latest_expire:%lld" , __FUNCTION__ ,
					pticker->ticker_name , pticker->expire_ms , new_latest_expire);
		}

		pticker = pticker->next;
	}

	ptick_list->latest_expire_ms = new_latest_expire;
	slog_log(slogd , SL_VERBOSE , "<%s> finish! ticker count:%d new_latest:%lld" , __FUNCTION__ , ptick_list->count , ptick_list->latest_expire_ms);
	return (get_curr_ms() - curr_ms);
}


//get curr ms
long long get_curr_ms()
{
	int ret = -1;
	struct timeval tv;
	ret = gettimeofday(&tv , NULL);
	if(ret < 0)
		return -1;
	return ((long long)tv.tv_sec*1000)+tv.tv_usec/1000;
}

/*
int shrink_memory(void *arg)
{
	carrier_env_t *penv = arg;
	int slogd = -1;
	static int last_target_pos = 0;
	static int last_client_pos = 0;



	if(!penv)
		return -1;




}*/

//hash_map
int init_target_hash_map(carrier_env_t *penv)
{
	cr_hash_map_t *pmap = NULL;
	target_info_t *ptarget_info = NULL;
	int slogd = -1;
	int i = -1;
	int list_len = 0;

	/***Arg Check*/
	if(!penv || !penv->ptarget_info)
		return -1;

	/***Init*/
	pmap = &penv->target_hash_map;
	ptarget_info = penv->ptarget_info;
	slogd = penv->slogd;

	/***Handle*/
	if(penv->ptarget_info->target_count <= 0)
	{
		slog_log(slogd , SL_INFO , "<%s> finished. target_count:%d" , __FUNCTION__ , penv->ptarget_info->target_count);
		return 0;
	}

	if(pmap->flag)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! hash_list inited!" , __FUNCTION__);
		return -1;
	}

	//list_len
	for(i=0; i<sizeof(hash_size_map)/sizeof(int); i++)
	{
		if(ptarget_info->target_count*2 <= hash_size_map[i])	//fd+proc_id
		{
			list_len = hash_size_map[i];
			break;
		}
	}
	if(list_len <= 0)
	{
		slog_log(slogd , SL_ERR , "<%s> failed! list_len not found! target_count:%d" , __FUNCTION__ , ptarget_info->target_count);
		return -1;
	}

	//alloc
	pmap->plist = calloc(list_len , sizeof(cr_hash_entry_t));
	if(!pmap->plist)
	{
		slog_log(slogd , SL_ERR , "<%s> failed for alloc list! list_len:%d err:%s" , __FUNCTION__ , list_len , strerror(errno));
		return -1;
	}

	//set
	pmap->len = list_len;
	pmap->flag = 1;
	pmap->entry_count = 0;
	slog_log(slogd , SL_INFO , "<%s> success! list_len:%d" , __FUNCTION__ , list_len);
	return 0;
}

int clear_hash_map(carrier_env_t *penv , char type)
{
	cr_hash_map_t *pmap = NULL;
	cr_hash_entry_t *pentry = NULL;
	cr_hash_entry_t *ptmp = NULL;
	int slogd = -1;
	int i = 0;
	char prefix[32] = {0};

	/***Arg Check*/
	if(!penv)
		return -1;

	/***Init*/
	slogd = penv->slogd;
	pmap = &penv->target_hash_map;
	if(type == CR_HASH_MAP_T_TARGET)
	{
		pmap = &penv->target_hash_map;
		strncpy(prefix , "[target]" , sizeof(prefix));
	}
	else if(type == CR_HASH_MAP_T_CLIENT)
	{
		pmap = &penv->client_hash_map;
		strncpy(prefix , "[client]" , sizeof(prefix));
	}
	else
	{
		slog_log(slogd , SL_ERR , "<%s> failed! type %d illegal! entry_type:%d value:%d" , __FUNCTION__ , type);
		return -1;
	}

	/***Basic*/
	slog_log(slogd , SL_INFO , "<%s> %s clear hash_map len:%d entry_count:%d" , __FUNCTION__ , prefix , pmap->len , pmap->entry_count);
	if(!pmap->plist)
	{
		memset(pmap , 0 , sizeof(cr_hash_map_t));
		slog_log(slogd , SL_INFO , "<%s> %s done! hash_list empty!" , __FUNCTION__ , prefix);
		return 0;
	}

	/***Clear*/
	//no entry
	if(pmap->entry_count <= 0)
	{
		free(pmap->plist);
		memset(pmap , 0 , sizeof(cr_hash_map_t));
		return 0;
	}

	//free each entry
	for(i=0; i<pmap->len; i++)
	{
		pentry = pmap->plist[i].next;
		while(pentry)
		{
			ptmp = pentry->next;
			slog_log(slogd , SL_INFO , "<%s> %s del entry (%d:%d)" , __FUNCTION__ , prefix , pentry->type , pentry->value);
			free(pentry);
			pentry = ptmp;
		}
	}
	free(pmap->plist);
	memset(pmap , 0 , sizeof(cr_hash_map_t));
	return 0;
}

/*
 * insert hash_map by type+value
 * -1:failed -2:exist 0:success
 */
int insert_hash_map(carrier_env_t *penv , char type , char entry_type , unsigned int value , void *refer)
{
	cr_hash_map_t *pmap = NULL;
	cr_hash_entry_t *pentry = NULL;
	cr_hash_entry_t *pentry_head = NULL;
	int slogd = -1;
	int pos = 0;
	char prefix[32] = {0};

	/***Arg Check*/
	if(!penv)
		return -1;

	/***Init*/
	if(type == CR_HASH_MAP_T_TARGET)
	{
		pmap = &penv->target_hash_map;
		strncpy(prefix , "[target]" , sizeof(prefix));
	}
	else if(type == CR_HASH_MAP_T_CLIENT)
	{
		pmap = &penv->client_hash_map;
		strncpy(prefix , "[client]" , sizeof(prefix));
	}
	else
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! type %d illegal! entry_type:%d value:%d" , __FUNCTION__ , prefix , type , entry_type , value);
		return -1;
	}
	slogd = penv->slogd;

	/***Basic*/
	if(entry_type!=CR_HASH_ENTRY_T_FD && entry_type!=CR_HASH_ENTRY_T_PROCID)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! type:%d illegal!" , __FUNCTION__ , prefix , entry_type);
		return -1;
	}
	if(!pmap->flag)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! hash_map not inited!" , __FUNCTION__ , prefix);
		return -1;
	}
	if(pmap->len <= 0 || !pmap->plist)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! hash_list len is 0!" , __FUNCTION__ , prefix);
		return -1;
	}

	/***Handle*/
	//1. check exist
	pentry = fetch_hash_entry(penv , CR_HASH_MAP_T_TARGET , entry_type , value);
	if(pentry)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! entry existed! type:%d value:%d" , __FUNCTION__ , prefix , entry_type , value);
		return -2;
	}

	//2.alloc
	pentry = calloc(1 , sizeof(cr_hash_entry_t));
	if(!pentry)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! alloc entry fail. type:%d value:%d err:%s" , __FUNCTION__ , prefix , entry_type , value , strerror(errno));
		return -1;
	}
	pentry->type = entry_type;
	pentry->value = value;
	pentry->refer = refer;

	//3.insert
	pos = value % pmap->len;
	pentry_head = &pmap->plist[pos];

	pentry->next = pentry_head->next;
	pentry_head->next = pentry;
	pmap->entry_count++;

	//return
	slog_log(slogd , SL_INFO , "<%s> %s success! type:%d value:%d refer:0x%X" , __FUNCTION__ , prefix , entry_type , value , refer);
	return 0;
}
int del_from_hash_map(carrier_env_t *penv , char type , char entry_type , unsigned int value)
{
	cr_hash_map_t *pmap = NULL;
	cr_hash_entry_t *pentry = NULL;
	cr_hash_entry_t *pprev = NULL;
	int slogd = -1;
	int pos = -1;
	char prefix[32] = {0};

	/***Arg Check*/
	if(!penv)
		return -1;

	/***Init*/
	slogd = penv->slogd;
	if(type == CR_HASH_MAP_T_TARGET)
	{
		pmap = &penv->target_hash_map;
		strncpy(prefix , "[target]" , sizeof(prefix));
	}
	else if(type == CR_HASH_MAP_T_CLIENT)
	{
		pmap = &penv->client_hash_map;
		strncpy(prefix , "[client]" , sizeof(prefix));
	}
	else
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! type %d illegal! entry_type:%d value:%d" , __FUNCTION__ , prefix , type , entry_type , value);
		return -1;
	}

	/***Basic*/
	if(entry_type!=CR_HASH_ENTRY_T_FD && entry_type!=CR_HASH_ENTRY_T_PROCID)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! type:%d illegal!" , __FUNCTION__ , prefix , entry_type);
		return -1;
	}
	if(!pmap->flag)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! hash_map not inited!" , __FUNCTION__ , prefix);
		return -1;
	}
	if(pmap->len <= 0 || pmap->entry_count<=0 || !pmap->plist)
	{
		slog_log(slogd , SL_INFO , "<%s> %s failed! hash_list empty!" , __FUNCTION__ , prefix);
		return 0;
	}

	/***Search*/
	pos = value % pmap->len;
	pprev = &pmap->plist[pos];
	pentry = pprev->next;
	while(pentry)
	{
		if(pentry->type==entry_type && pentry->value==value)
		{
			pprev->next = pentry->next;
			pmap->entry_count--;
			free(pentry);
			slog_log(slogd , SL_INFO , "<%s> %s success! type:%d value:%d rest_count:%d" , __FUNCTION__ , prefix , entry_type , value , pmap->entry_count);
			return 0;
		}

		pprev = pentry;
		pentry = pentry->next;
	}
	return -1;
}
cr_hash_entry_t *fetch_hash_entry(carrier_env_t *penv , char type , char entry_type , unsigned int value)
{
	cr_hash_map_t *pmap = NULL;
	cr_hash_entry_t *pentry = NULL;
	int slogd = -1;
	int pos = -1;
	char prefix[32] = {0};

	/***Arg Check*/
	if(!penv)
		return NULL;

	/***Init*/
	slogd = penv->slogd;
	if(type == CR_HASH_MAP_T_TARGET)
	{
		pmap = &penv->target_hash_map;
		strncpy(prefix , "[target]" , sizeof(prefix));
	}
	else if(type == CR_HASH_MAP_T_CLIENT)
	{
		pmap = &penv->client_hash_map;
		strncpy(prefix , "[client]" , sizeof(prefix));
	}
	else
	{
		slog_log(slogd , SL_ERR , "<%s> failed! type %d illegal! entry_type:%d value:%d" , __FUNCTION__ , type , entry_type , value);
		return NULL;
	}

	/***Basic*/
	if(entry_type!=CR_HASH_ENTRY_T_FD && entry_type!=CR_HASH_ENTRY_T_PROCID)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! type:%d illegal!" , __FUNCTION__ , prefix , entry_type);
		return NULL;
	}
	if(!pmap->flag)
	{
		slog_log(slogd , SL_ERR , "<%s> %s failed! hash_map not inited!" , __FUNCTION__ , prefix);
		return NULL;
	}
	if(pmap->len <= 0 || pmap->entry_count<=0 || !pmap->plist)
	{
		slog_log(slogd , SL_INFO , "<%s> %s failed! hash_list empty!" , __FUNCTION__ , prefix);
		return NULL;
	}

	/***Search*/
	pos = value % pmap->len;
	pentry = pmap->plist[pos].next;
	while(pentry)
	{
		if(pentry->type==entry_type && pentry->value==value)
			return pentry;

		pentry = pentry->next;
	}

	return NULL;
}

int dump_hash_map(carrier_env_t *penv , char type)
{
	cr_hash_map_t *pmap = NULL;
	cr_hash_entry_t *pentry = NULL;
	int slogd = -1;
	int i = 0;

	/***Arg Check*/
	if(!penv)
		return -1;

	/***Init*/
	slogd = penv->slogd;
	pmap = &penv->target_hash_map;
	if(type == CR_HASH_MAP_T_TARGET)
		pmap = &penv->target_hash_map;
	else if(type == CR_HASH_MAP_T_CLIENT)
		pmap = &penv->client_hash_map;
	else
	{
		slog_log(slogd , SL_ERR , "<%s> failed! type %d illegal!" , __FUNCTION__ , type);
		return -1;
	}

	/***Basic*/
	if(type == CR_HASH_MAP_T_TARGET)
		slog_log(slogd ,SL_INFO , "--------------------------[TARGET_HASH_MAP]-------------");
	else
		slog_log(slogd ,SL_INFO , "--------------------------[CLIENT_HASH_MAP]-------------");
	slog_log(slogd , SL_INFO , "<LEN>:%d <ENTRY>:%d" , pmap->len , pmap->entry_count);
	if(!pmap->plist || pmap->entry_count<=0)
	{
		slog_log(slogd ,SL_INFO , "--------------------------[END]-------------");
		return 0;
	}

	/***Print*/
	//each entry
	for(i=0; i<pmap->len; i++)
	{
		pentry = pmap->plist[i].next;
		if(pentry)
			slog_log(slogd , SL_INFO , "++++%d+++" , i);
		while(pentry)
		{
			slog_log(slogd , SL_INFO , ">>(%d:%d) " , pentry->type , pentry->value);
			pentry = pentry->next;
		}
		if(pentry)
			slog_log(slogd , SL_INFO , "++++%d+++" , i);
	}
	return 0;
}

int check_client_hash_map(carrier_env_t *penv)
{
	int slogd = penv->slogd;
	cr_hash_map_t *pmap = &penv->client_hash_map;
	cr_hash_entry_t *pentry = NULL;
	client_list_t *pclient_list = penv->pclient_list;
	client_info_t *pclient = NULL;
	int list_len = 0;
	int i = 0;
	int ret = -1;

	/***Arg Check*/
	if(!pclient_list)
		return -1;
	if(pclient_list->total_count <= 0)
		return clear_hash_map(penv , CR_HASH_MAP_T_CLIENT);

	/***Init*/
	if(!pmap->flag)
	{
		slog_log(slogd , SL_INFO , "<%s> try to init map!" , __FUNCTION__);
		memset(pmap , 0 , sizeof(cr_hash_map_t));
		//list_len
		for(i=0; i<sizeof(hash_size_map)/sizeof(int); i++)
		{
			if(pclient_list->total_count * 2 <= hash_size_map[i])	//fd
			{
				list_len = hash_size_map[i];
				break;
			}
		}
		if(list_len <= 0)
		{
			slog_log(slogd , SL_ERR , "<%s> Init failed! list_len not found! count:%d" , __FUNCTION__ , pclient_list->total_count);
			return -1;
		}

		//alloc
		pmap->plist = calloc(list_len , sizeof(cr_hash_entry_t));
		if(!pmap->plist)
		{
			slog_log(slogd , SL_ERR , "<%s> failed for alloc list! list_len:%d err:%s" , __FUNCTION__ , list_len , strerror(errno));
			return -1;
		}

		//set
		pmap->len = list_len;
		pmap->flag = 1;
		pmap->entry_count = 0;
		slog_log(slogd , SL_INFO , "<%s> success! list_len:%d" , __FUNCTION__ , list_len);
	}

	/***Check Each Client*/
	pclient = pclient_list->list;
	while(pclient)
	{
		do
		{
			//检视fd
			if(pclient->fd <= 0)
				break;

			//需要验证的client
			if(!pclient->verify)
				break;

			//第一次生成的hash表里一定是稳定建立链接的client
			//获取hash
			pentry = fetch_hash_entry(penv , CR_HASH_MAP_T_CLIENT , CR_HASH_ENTRY_T_FD , pclient->fd);

			//如果有则更新
			if(pentry)
			{
				slog_log(slogd , SL_DEBUG , "<%s> update (%d:%d) 0x%X --> 0x%X" , __FUNCTION__ , CR_HASH_ENTRY_T_FD , pclient->fd ,
						pentry->refer , pclient);
				pentry->refer = pclient;
				break;
			}

			//若无则插入
			ret = insert_hash_map(penv , CR_HASH_MAP_T_CLIENT , CR_HASH_ENTRY_T_FD , pclient->fd , pclient);
			if(ret < 0)
				slog_log(slogd , SL_ERR , "<%s> insert (%d:%d) of %s failed!" , __FUNCTION__ , CR_HASH_ENTRY_T_FD , pclient->fd , pclient->proc_name);
			else
				slog_log(slogd , SL_INFO , "<%s> insert (%d:%d) of %s success!" , __FUNCTION__ , CR_HASH_ENTRY_T_FD , pclient->fd , pclient->proc_name);

			break;
		}while(0);

		pclient = pclient->next;
	}

	slog_log(slogd , SL_DEBUG , "<%s> finish!" , __FUNCTION__);
	//dump
	dump_hash_map(penv , CR_HASH_MAP_T_CLIENT);
	return 0;
}

int check_target_hash_map(carrier_env_t *penv)
{
	int slogd = -1;
	cr_hash_entry_t *pentry = NULL;
	cr_hash_map_t *pmap = NULL;
	target_detail_t *ptarget = NULL;
	int ret = -1;
	if(!penv)
		return -1;

	slogd = penv->slogd;
	pmap = &penv->target_hash_map;
	//hash表的entry=target->count * 2; (fd + proc_id)
	if(pmap->entry_count != penv->ptarget_info->target_count * 2)
	{
		slog_log(slogd , SL_INFO , "<%s> count not match! %d vs %d will rebuild again!" , __FUNCTION__ , pmap->entry_count , penv->ptarget_info->target_count);
		ptarget = penv->ptarget_info->head.next;
		while(ptarget)
		{
			//fd
			if(ptarget->fd > 0)
			{
				pentry = fetch_hash_entry(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_FD , ptarget->fd);
				if(!pentry)
				{
					ret = insert_hash_map(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_FD , ptarget->fd , ptarget);
					if(ret < 0)
						slog_log(slogd , SL_ERR , "<%s> insert (%d:%d) failed!" , __FUNCTION__ , CR_HASH_ENTRY_T_FD , ptarget->fd);
					else
						slog_log(slogd , SL_ERR , "<%s> insert (%d:%d) done!" , __FUNCTION__ , CR_HASH_ENTRY_T_FD , ptarget->fd);
				}
			}

			//proc_id
			if(ptarget->proc_id > 0)
			{
				pentry = fetch_hash_entry(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_PROCID , ptarget->proc_id);
				if(!pentry)
				{
					ret = insert_hash_map(penv , CR_HASH_MAP_T_TARGET , CR_HASH_ENTRY_T_PROCID , ptarget->proc_id , ptarget);
					if(ret < 0)
						slog_log(slogd , SL_ERR , "<%s> insert (%d:%d) failed!" , __FUNCTION__, CR_HASH_ENTRY_T_PROCID , ptarget->fd);
					else
						slog_log(slogd , SL_ERR , "<%s> insert (%d:%d) done!" , __FUNCTION__ , CR_HASH_ENTRY_T_PROCID , ptarget->fd);
				}
			}

			ptarget = ptarget->next;
		}

	}
	else
		slog_log(slogd , SL_DEBUG , "<%s> matched! %d vs %d" , __FUNCTION__ , pmap->entry_count , penv->ptarget_info->target_count);

	//dump
	dump_hash_map(penv , CR_HASH_MAP_T_TARGET);
	return 0;
}

