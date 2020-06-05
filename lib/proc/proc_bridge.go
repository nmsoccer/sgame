package proc

import (
	"fmt"
	"unsafe"		
)
/*
// This is a wrapper of proc_bridge APIs for golang
// basic libs[proc_bridge slog stlv] and headers should be installed first.
// refer:https://github.com/nmsoccer/proc_bridge

#cgo CFLAGS: -g
#cgo LDFLAGS: -lproc_bridge -lslog -lstlv -lm -lrt
#include <proc_bridge/proc_bridge.h>
*/
import "C"

/*
 * open bridge
 * @name_space:系统的命名空间
 * @proc_id:服务进程的全局ID
 * @slogd:slog的描述句柄
 * @return:
 * -1:failed
 * >=0:SUCCESS 返回bridge句柄描述符
 */
func OpenBridge(name_space string, proc_id int, slogd int) int {
	_func_ := "OpenBridge";
	if name_space=="" || proc_id<=0 {
		fmt.Printf("%s Failed! arg illegal!", _func_);
		return -1;
	}
	cs := C.CString(name_space);
	defer C.free(unsafe.Pointer(cs));
	result := C.open_bridge(cs , C.int(proc_id) , C.int(slogd));
	return int(result);
}

/*
 * close bridge_hub of process
  * @bd:bridge_descriptor,该进程打开的bridge描述符
 * @return -1:failed 0:success
 */
func CloseBridge(bd int) int {
	result := C.close_bridge(C.int(bd));
	return int(result);
}

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
func SendBridge(bd int, target_id int, sending_data []byte, data_len int) int {
	bs := sending_data[:data_len];
	//C.my_print(((*C.char)(unsafe.Pointer(&bs[0]))) , C.int(data_len));
	result := C.send_to_bridge(C.int(bd) , C.int(target_id) ,  ((*C.char)(unsafe.Pointer(&bs[0]))) , C.int(data_len));
	return int(result);
}

/*
 * recv_from_bridge
 * @bd:bridge_descriptor,该进程打开的bridge描述符
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
func RecvBridge(bd int, recv_buff []byte, recv_len int, sender *int, drop_time int) int {
	bs := recv_buff[:recv_len];
	result := C.recv_from_bridge(C.int(bd) , ((*C.char)(unsafe.Pointer(&bs[0]))) , C.int(recv_len) , ((*C.int)(unsafe.Pointer(sender))) , 
		C.int(drop_time));
	return int(result);
}
