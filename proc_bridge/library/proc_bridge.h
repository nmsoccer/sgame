/*
 * proc_bridge.h
 *
 *  Created on: 2013-12-22
 *      Author: Administrator
 */

#ifndef PROC_BRIDGE_H_
#define PROC_BRIDGE_H_

#ifdef __cplusplus
extern "C"{
#endif

#include "proc_bridge_base.h"

/*
 * open bridge
 * @name_space:系统的命名空间
 * @proc_id:服务进程的全局ID
 * @slogd:slog的描述句柄
 * @return:
 * -1:failed
 * >=0:SUCCESS 返回bridge句柄描述符
 */
extern int open_bridge(char *name_space , int proc_id , int slogd);

/*
 * close bridge_hub of process
  * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @return -1:failed 0:success
 */
extern int close_bridge(int bd);

/*
 * send_to_bridge
 * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @target_id:目标服务进程的全局ID
 * @sending_data:发送的数据
 * @len:发送数据长度
 * @return:
 * -1：错误
 * -2：发送缓冲区满
 * -3：发送数据超出包长
 * 0：成功
 */
extern int send_to_bridge(int bd , int target_id , char *sending_data , int len);

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
extern int recv_from_bridge(int bd , char *recv_buff , int recv_len , int *sender , int drop_time);


/*
 * 根据命名空间和proc_id获得对应的key
 * @-1:failed else:key
 */
extern int get_bridge_shm_key(char *name_space , int proc_id , int creater , int slogd);

extern bridge_hub_t *bd2bridge(int bd);

#ifdef __cplusplus
}
#endif
#endif /* PROC_BRIDGE_H_ */
