package proc

import (
	"fmt"
	"sync"
)


type ProcHeader interface {
    Send(target_id int, sending_data []byte, data_len int) int
    SendByLock(target_id int, sending_data []byte, data_len int) int
    Recv(recv_buff []byte, recv_len int, sender *int) int
    Close() int	
}



////////////////SPEC PROC STRUCT//////////////
type Proc struct {
	bd int
	sync.Mutex
}

/*
 * 打开本进程的通信
 * @name_space:系统的命名空间
 * @proc_id:服务进程的全局ID
 * @return:
 * -1:failed
 * >=0:SUCCESS 返回bridge句柄描述符
 */
func Open(name_space string, proc_id int) *Proc {
	p := new(Proc)
	ret := OpenBridge(name_space, proc_id, -1);
	if ret < 0 {
		fmt.Printf("open bridge of %s at %d failed!\n", name_space , proc_id);
		return nil;
	}
	
	p.bd = ret;
	return p;
}

/*
 * 发送数据到目标进程
 * @target_id:目标服务进程的全局ID
 * @sending_data:发送的数据
 * @len:发送数据长度
 * @return:
 * -1：错误
 * -2：发送缓冲区满
 * -3：发送数据超出包长
 * 0：成功
 */
func (p *Proc) Send(target_id int, sending_data []byte, data_len int) int {
	if p.bd < 0 {
		return -1;
	}
	return SendBridge(p.bd , target_id, sending_data, data_len);
}

/*
 * 发送数据到目标进程.发送时加锁
 * @target_id:目标服务进程的全局ID
 * @sending_data:发送的数据
 * @len:发送数据长度
 * @return:
 * -1：错误
 * -2：发送缓冲区满
 * -3：发送数据超出包长
 * 0：成功
 */
func (p *Proc) SendByLock(target_id int, sending_data []byte, data_len int) int {
	if p.bd < 0 {
		return -1;
	}
	p.Lock();
	ret := SendBridge(p.bd , target_id, sending_data, data_len);
	p.Unlock();
	return ret;
}

/*
 * 接收其他进程的数据
 * @recv_buff:接收数据缓冲区
 * @recv_len:接收缓冲区长度
 * @sender:发送的进程ID
 * @return:
 * -1：错误
 * -2：接收缓冲区空
 * -3：接收数据超出包长
 * else:实际接收的长度
 */
func (p *Proc) Recv(recv_buff []byte, recv_len int, sender *int) int{
	if p.bd < 0 {
		return -1;
	}
	return RecvBridge(p.bd, recv_buff, recv_len, sender, -1);
}

/*
 * 关闭进程通信
  * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @return -1:failed 0:success
 */
func (p *Proc) Close() int{
	if p.bd < 0 {
		return -1;
	}
	return CloseBridge(p.bd);
}

