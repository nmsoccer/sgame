package comm

import (
	"bytes"
	"io"
	"net"
	lnet "sgame/lib/net"
	"sgame/proto/ss"
	"strconv"
	"sync"
	"time"
)

const (
	MAX_PKG_LEN = ss.MAX_SS_MSG_SIZE //200K

	TCP_SERV_ACC_TIMEOUT = 5 //tcp_serv accept timeout ms
	CLIENT_STAT_CLOSING  = 0
	CLIENT_STAT_NORMAL   = 1

	MAX_QUEUE_DATA = MAX_PKG_LEN //max queue data == max_pkg_len
	//RECV_QUEUE_LEN=128
	SND_QUEUE_LEN = 128

	SERV_RECV_QUEUE  = 4096
	SERV_SND_QUEUE   = 4096
	MAX_PKG_PER_RECV = 200 //max pkg per recv

	//QUEUE_INFO
	QUEUE_INFO_NORMAL = 0
	QUEUE_INFO_CLOSE  = 1
	QUEUE_INFO_BROADCAST = 2

	//ClientPkg.PkgType
	CLIENT_PKG_T_NORMAL      = 1 // special pkg using key
	CLIENT_PKG_T_BROADCAST   = 2 //broadcast pkg
	CLIENT_PKG_T_CONN_CLOSED = 3 //client connection closed
	CLIENT_PKG_T_CLOSE_CONN  = 4 //close client

	//R&W TIMEOUT
	CLIENT_RW_TIMEOUT = 2                    //ms
	CLIENT_RW_CACHE   = (2 * MAX_QUEUE_DATA) //tcp_client.snd_cache & tcp_client.recv_cache
)

/*Caller Proc <-> tcp_serv */
type ClientPkg struct {
	PkgType int8
	//Flag uint8
	ClientKey int64 //the key identify a client
	Data      []byte
}

type queue_data struct {
	flag uint8 //tag net_pkg.PKG_OP_XX
	info uint8
	idx  int //index of client
	key  int64
	data []byte
}

type tcp_client struct {
	stat         int8 //refer CLIENT_STAT_XX
	key          int64
	server_close bool         //if connection closed , true:server close connection; false:client close connection
	conn         *net.TCPConn //connection
	index        int          //index in client-array
	recv_cache   []byte
	snd_cache    []byte
	//recv_queue chan *queue_data //client -> server
	snd_queue chan *queue_data //server -> client
}

type TcpServ struct {
	sync.Mutex
	addr        string
	max_conn    int
	curr_conn   int
	close_ch    chan int      //closing connection
	client_list []*tcp_client //client
	exit_ch     chan bool     //exit channel
	loop_index  int
	recv_queue  chan *queue_data //client -> server
	snd_queue   chan *queue_data //server -> client
	key_map     map[int64]int    //client_key <-> client_idx
}

//new tcp_serv and start listener serve
func StartTcpServ(pconfig *CommConfig, addr string, max_conn int) *TcpServ {
	var _func_ = "<StartTcpServ>"
	log := pconfig.Log

	//new
	pserv := new(TcpServ)

	//init
	pserv.addr = addr
	pserv.max_conn = max_conn
	pserv.curr_conn = 0
	pserv.close_ch = make(chan int, pserv.max_conn)
	pserv.client_list = make([]*tcp_client, pserv.max_conn) //init
	pserv.exit_ch = make(chan bool, 1)                      //non-block
	pserv.recv_queue = make(chan *queue_data, SERV_RECV_QUEUE)
	pserv.snd_queue = make(chan *queue_data, SERV_SND_QUEUE)
	pserv.key_map = make(map[int64]int)

	//start
	go tcp_server(pconfig, pserv)
	log.Info("%s at %s finish!", _func_, addr)
	return pserv
}

/*
close tcp serv
*/
func (pserv *TcpServ) Close(pconfig *CommConfig) {
	log := pconfig.Log
	log.Info("tcp_serv closing... addr:%s", pserv.addr)
	pserv.exit_ch <- true
}

/*read from clients
@return: nil:read fail or empty; []*ClientPkg read success
*/
func (pserv *TcpServ) Recv(pconfig *CommConfig) []*ClientPkg {
	var pdata *queue_data
	var _func_ = "TcpServ.Recv"
	log := pconfig.Log
	result_len := len(pserv.recv_queue)

	//empty
	if result_len <= 0 {
		return nil
	}

	//init results
	if result_len > MAX_PKG_PER_RECV { //max pkgs per handle
		result_len = MAX_PKG_PER_RECV
	}
	results := make([]*ClientPkg, result_len)
	i := 0

	//fulfill result
	for {
		if i >= result_len {
			break
		}

		select {
		case pdata = <-pserv.recv_queue:
			results[i] = new(ClientPkg)
			if pdata.info == QUEUE_INFO_NORMAL {
				results[i].PkgType = CLIENT_PKG_T_NORMAL
			} else if pdata.info == QUEUE_INFO_CLOSE {
				log.Info("%s [%d] read close connection! key:%v idx:%v", _func_, i, pdata.key, pdata.idx)
				results[i].PkgType = CLIENT_PKG_T_CONN_CLOSED
			}
			//results[i].ClientKey = pserv.client_list[pdata.idx].key;
			results[i].ClientKey = pdata.key
			results[i].Data = pdata.data
		default:
			//nothing;
		}

		i++
	}

	//log.Debug("%s recv pkg:%d" , _func_ , result_len);
	return results
}

/*Send pkg to client
@return: -1: failed 0:success
*/
func (pserv *TcpServ) Send(pconfig *CommConfig, ppkg *ClientPkg) int {
	var _func_ = "<TcpServ.Send>"
	log := pconfig.Log

	//check cap
	if len(pserv.snd_queue) >= cap(pserv.snd_queue) {
		log.Err("%s failed! send_queue full!", _func_)
		return -1
	}

	if len(ppkg.Data) >= MAX_PKG_LEN {
		log.Err("%s failed! pkg data overflow! len:%d max:%d idx:%d", _func_, len(ppkg.Data), MAX_PKG_LEN, pserv.key_map[ppkg.ClientKey])
		return -1
	}

	//check key
	idx  , ok := pserv.key_map[ppkg.ClientKey];
	if !ok {
		log.Err("%s client not exist! key:%d" , _func_ , ppkg.ClientKey);
		return -1;
	}

	//send
	pdata := new(queue_data)
	pdata.info = QUEUE_INFO_NORMAL
	pdata.idx = idx;
	pdata.data = ppkg.Data
	pdata.key = ppkg.ClientKey
	//check pkg type
	switch ppkg.PkgType {
	case CLIENT_PKG_T_CLOSE_CONN:
		log.Info("%s dectect positive close connection! key:%d", _func_, pdata.key)
		pdata.info = QUEUE_INFO_CLOSE //will close connection
	case CLIENT_PKG_T_BROADCAST:
		log.Info("%s dectect broadcast!", _func_);
		pdata.info = QUEUE_INFO_BROADCAST;
	default:
		//normal pkg
	}
	pserv.snd_queue <- pdata
	return 0
}

func (pserv *TcpServ) GetConnNum() int {
	return pserv.curr_conn
}

/*--------------------------Static Func----------------------------*/
/*--------------------------------tcp_server-----------------------------*/
//generate key
func generate_client_key(idx int) int64 {
	var key int64 = 0
	curr_ts := time.Now().UnixNano() / 1000 / 1000 //ms

	key |= int64(((idx + 1&0xFFFF) << 44))
	key |= (curr_ts & 0xFFFFFFFFFFF)
	return key
}

//go routine of listener
func tcp_server(pconfig *CommConfig, pserv *TcpServ) {
	var _func_ = "<tcp_server>"
	var log = pconfig.Log
	log.Info("%s starting...", _func_)
	//addr
	serv_addr, err := net.ResolveTCPAddr("tcp4", pserv.addr)
	if err != nil {
		log.Err("resolve addr:%s failed! err:%v", pserv.addr, err)
		return
	}

	//listen
	listener, err := net.ListenTCP("tcp4", serv_addr)
	if err != nil {
		log.Err("%s listen at %s failed! err:%v", _func_, pserv.addr, err)
		return
	}

	log.Info("%s listen at %s done!", _func_, pserv.addr)
	//accept
	for {
		//check exit
		if len(pserv.exit_ch) > 0 {
			log.Info("%s detect exit flg!", _func_)
			return
		}
		/*
		select {
		case <-pserv.exit_ch:
			log.Info("%s detect exit flg!", _func_)
			return
		default:
			//nothing
		}*/

		//set deadline 5ms
		listener.SetDeadline(time.Now().Add(TCP_SERV_ACC_TIMEOUT * time.Millisecond))

		//acc
		conn, err := listener.AcceptTCP()
		if err != nil {
			if net_err, ok := err.(net.Error); ok {
				if net_err.Temporary() || net_err.Timeout() { //timeout no-err
					//log.Debug("%s time out." , _func_);
				} else {
					log.Err("%s accept net-err fail! err:%v", _func_, err)
					conn.Close()
				}
			} else {
				log.Err("%s accept fail! err:%v", _func_, err)
				conn.Close()
			}
		} else {
			//wrap connection
			log.Info("%s accept a new connection! peer:%s", _func_, conn.RemoteAddr().String())

			//add client
			pserv.add_client(pconfig, conn)
		}

		//close clients
		pserv.close_clients(pconfig)

		//dispatch send pkg
		pserv.dispatch_send_pkg(pconfig)
	}

}

//detect and close closing clients
func (pserv *TcpServ) close_clients(pconfig *CommConfig) {
	var _func_ = "<tcp_serv.close_clients>"
	log := pconfig.Log

	//check closing
	if len(pserv.close_ch) <= 0 {
		return
	}

	//close
	for {
		select {
		case idx := <-pserv.close_ch:
			pclient := pserv.client_list[idx]
			//notify upper server if client close
			if !pclient.server_close {
				pdata := new(queue_data)
				pdata.key = pclient.key
				pdata.idx = pclient.index
				pdata.info = QUEUE_INFO_CLOSE
				pserv.recv_queue <- pdata
			}

			//close
			close_client(pconfig, pclient)
			delete(pserv.key_map, pserv.client_list[idx].key)
			pserv.client_list[idx] = nil
			pserv.curr_conn--
			log.Info("%s will close connection idx:%d curr_count:%d", _func_, idx, pserv.curr_conn)
		default:
			return
		}
	}

}

func close_connection(pconfig *CommConfig, conn *net.TCPConn) {
	conn.Close()
}

//add a client to tcp_serv
func (pserv *TcpServ) add_client(pconfig *CommConfig, conn *net.TCPConn) {
	var _func_ = "<tcp_serv.add_client>"
	log := pconfig.Log

	//check conn count
	if pserv.curr_conn >= pserv.max_conn {
		log.Err("%s fail! connection count:%d reached uplimit!", _func_, pserv.curr_conn)
		close_connection(pconfig, conn)
		return
	}

	//search an empty index
	var i = 0
	pserv.Lock()
	for i = 0; i < pserv.max_conn; i++ {
		if pserv.client_list[i] == nil {
			break
		}
	}
	pserv.Unlock()
	if i >= pserv.max_conn {
		log.Err("%s fail! no empy pos found! curr:%d max:%d", _func_, pserv.curr_conn, pserv.max_conn)
		close_connection(pconfig, conn)
		return
	}

	//new client
	pclient := new_client(pconfig)
	if pclient == nil {
		log.Err("%s fail! new_client failed!", _func_)
		close_connection(pconfig, conn)
		return
	}
	pclient.index = i
	pclient.conn = conn
	pclient.stat = CLIENT_STAT_NORMAL
	pclient.key = generate_client_key(i)

	//add client
	pserv.Lock()
	pserv.client_list[i] = pclient
	pserv.curr_conn++
	pserv.key_map[pclient.key] = pclient.index
	pserv.Unlock()

	log.Info("%s at %s success! index:%d curr_count:%d", _func_, pserv.addr, i, pserv.curr_conn)
	go pclient.run(pconfig, pserv)
	return
}

/*
dispatch tcp_serv.send_queue to each client
*/
func (pserv *TcpServ) dispatch_send_pkg(pconfig *CommConfig) {
	var _func_ = "<tcpserver.dispatch_send_pkg>"
	log := pconfig.Log

	//check channel
	ch_len := len(pserv.snd_queue)
	if ch_len <= 0 {
		return
	}

	var pdata *queue_data
	var idx int
	var pclient *tcp_client
	//dispatch
	for i := 0; i < ch_len; i++ {
		//queue_data
		pdata = <-pserv.snd_queue

		//check index
		idx = pdata.idx
		if idx >= pserv.max_conn || idx < 0 || pserv.curr_conn <= 0 {
			log.Err("%s illegal pkg! idx error. idx:%d", _func_, idx)
			continue
		}
		pclient = pserv.client_list[pdata.idx]

		//check connection
		if pclient.stat == CLIENT_STAT_CLOSING {
			log.Info("%s detect closing client idx:%d drop pkg!", _func_, idx)
			continue
		}

		//valid key
		if pclient.key != pdata.key {
			log.Info("%s key not matched %v <-> %v idx:%d", _func_, pclient.key, pdata.key, idx)
			continue
		}

		//append to client queue
		//log.Debug("%s append to client idx:%d data:%v success!" , _func_ , idx , pdata.data);
		pclient.append_send_data(pconfig, pdata)
		/*
				if len(pclient.snd_queue) >= cap(pclient.snd_queue) {
				    log.Err("%s fail! client snd_queue full! drop pkg! idx:%d" , _func_ , pclient.index);
				    continue;
			    }
			    //add
			    pclient.snd_queue <- pdata;*/
	}

}

/*--------------------------------tcp_client-----------------------------*/
//create a new tcp_client
func new_client(pconfig *CommConfig) *tcp_client {
	//new
	pclient := new(tcp_client)

	pclient.server_close = false
	pclient.recv_cache = make([]byte, CLIENT_RW_CACHE) //pre alloc cache
	pclient.recv_cache = pclient.recv_cache[:0]
	pclient.snd_cache = make([]byte, CLIENT_RW_CACHE)
	pclient.snd_cache = pclient.snd_cache[:0]
	//pclient.recv_queue = make(chan *queue_data , RECV_QUEUE_LEN);
	pclient.snd_queue = make(chan *queue_data, SND_QUEUE_LEN)
	return pclient
}

//close a tcp_client
func close_client(pconfig *CommConfig, pclient *tcp_client) {
	var _func_ = "<close_client>"
	log := pconfig.Log

	//close connection
	err := pclient.conn.Close()
	if err != nil {
		log.Err("%s close conn err:%v idx:%d", _func_, err, pclient.index)
	}

	//clear data
	pclient.conn = nil
	pclient.snd_cache = nil
	pclient.recv_cache = nil
	//pclient.snd_queue = nil;
	//pclient.recv_queue = nil;
	return
}

//client go-routine
func (pclient *tcp_client) run(pconfig *CommConfig, pserv *TcpServ) {

	for {
		//exit-goroutine
		if pclient.stat == CLIENT_STAT_CLOSING {
			break
		}

		//read from client
		pclient.read(pconfig, pserv)

		//send to client
		pclient.send(pconfig, pserv)

		//sleep
		time.Sleep(1 * time.Millisecond)
	}

}

//append send_data
func (pclient *tcp_client) append_send_data(pconfig *CommConfig, pdata *queue_data) {
	var _func_ = "<tcp_client.append_send_data>"
	log := pconfig.Log

	if len(pclient.snd_queue) >= cap(pclient.snd_queue) {
		log.Err("%s fail! client snd_queue full! drop pkg! idx:%d", _func_, pclient.index)
		return
	}

	//add
	pclient.snd_queue <- pdata
}

//handle special pkg
func (pclient *tcp_client) handle_spec_pkg(pconfig *CommConfig, pserv *TcpServ, pdata *queue_data) {
	var _func_ = "<tcp_client.handle_spec_pkg>"
	log := pconfig.Log

	//switch
	switch pdata.flag {
	case lnet.PKG_OP_ECHO:
		log.Info("%s pkg option:%d [echo]", _func_, pdata.flag)
		//pdata.data = append(pdata.data , []byte("----from server")...);
		//resend
		pclient.append_send_data(pconfig, pdata)
	case lnet.PKG_OP_STAT:
		log.Info("%s pkg option:%d [stat]", _func_, pdata.flag)
		//if pdata.data != []byte(local_net.FETCH_STAT_KEY) {
		if bytes.Compare(pdata.data, []byte(lnet.FETCH_STAT_KEY)) != 0 {
			log.Err("%s get stat fail! illegal key!", _func_)
			break
		}
		msg := "{\"stat\":0, \"conn\":" + strconv.Itoa(pserv.curr_conn) + "}"
		pdata.data = make([]byte, len(msg))
		copy(pdata.data, []byte(msg))
		pclient.append_send_data(pconfig, pdata)
	default:
		log.Info("%s unhandle pkg option:%d", _func_, pdata.flag)
	}

	return
}

func (pclient *tcp_client) read(pconfig *CommConfig, pserv *TcpServ) {
	var _func_ = "<tcp_client.read>"
	var read_len = 0
	var parsed int = 0
	var err error

	log := pconfig.Log
	conn := pclient.conn
	read_data := make([]byte, MAX_QUEUE_DATA)

	if pclient.stat == CLIENT_STAT_CLOSING {
		log.Info("%s in closing! index:%d", _func_, pclient.index)
		return
	}

	//consist read
	for {
		read_data = read_data[:cap(read_data)]
		//set dead line
		conn.SetReadDeadline(time.Now().Add(CLIENT_RW_TIMEOUT * time.Millisecond)) //set timeout

		//read
		read_len, err = conn.Read(read_data)
		//check err
		if err != nil { //parse err
			close_conn := true
			//net err
			if net_err, ok := err.(net.Error); ok {
				if net_err.Temporary() || net_err.Timeout() { //no data prepared
					//log.Debug("%s no data" , _func_);
					close_conn = false
				} else { //other
					log.Err("%s connection net.err other! will close conn! detail:%v index:%d", _func_, err, pclient.index)
				}
			} else if err == io.EOF { //read a closed connection
				log.Info("%s connection closed! index:%d", _func_, pclient.index)
			} else { //other
				log.Err("%s read meets an error! will close! err:%v index:%d", _func_, err, pclient.index)
			}

			//need close
			if close_conn {
				pclient.stat = CLIENT_STAT_CLOSING
				pserv.close_ch <- pclient.index
			}
			return
		}

		//empty useless
		if read_len == 0 {
			log.Info("%s peer closed will close connection index:%d!", _func_, pclient.index)
			pclient.stat = CLIENT_STAT_CLOSING
			pserv.close_ch <- pclient.index
			return
		}

		//append readed
		//log.Debug("%s read %d bytes" , _func_ , read_len);
		pclient.recv_cache = append(pclient.recv_cache, read_data[:read_len]...) //may expand cap(recv_cache)

		//flush cache
		parsed = pclient.flush_recv_cache(pconfig, pserv)
		if parsed < 0 {
			log.Err("%s pkg data illegal! will close connection! idx:%d", _func_, pclient.index)
			pclient.stat = CLIENT_STAT_CLOSING
			pserv.close_ch <- pclient.index
			return
		}
		//log.Debug("%s flush %d pkgs! idx:%d", _func_, parsed, pclient.index)

	}

}

/*parse read bytes to queue-data
@return: -1:error 0:not ready else:handled count of pkg
*/
func (pclient *tcp_client) flush_recv_cache(pconfig *CommConfig, pserv *TcpServ) int {
	var _func_ = "<tcp_client.flush_recv_cache>"
	log := pconfig.Log
	var tag uint8
	var pkg_data []byte
	var pkg_len int
	var raw_data []byte
	var parsed int

	raw_data = pclient.recv_cache
	for {
		//unpack raw data
		tag, pkg_data, pkg_len = lnet.UnPackPkg(raw_data)

		//error
		if tag == 0xFF {
			log.Err("%s error! pkg format illegal!", _func_)
			return -1
		}

		//not- ready
		if tag == 0 {
			log.Debug("%s pkg not ready!", _func_)
			//not moving memory
			if parsed == 0 {
				break
			}

			//reset cache
			copy(pclient.recv_cache[:cap(pclient.recv_cache)], raw_data[:])
			/*
			   if copyed != len(raw_data) {
			   	log.Err("%s reset recv_cache failed! copy:%d raw:%d" , _func_ , copyed , len(raw_data));
			   	return -1;
			   }
			   pclient.recv_cache = pclient.recv_cache[0:copyed];*/
			log.Debug("%s saving  cached %d bytes", _func_, len(raw_data))
			break
		}

		//parse success
		parsed++

		//check lenth
		if pkg_len > MAX_PKG_LEN {
			log.Err("%s drop pkg for lenth overflow! pkg_len:%d max_len:%d idx:%d", _func_, pkg_len, MAX_PKG_LEN, pclient.index)
		} else {
			//put  queue
			pdata := new(queue_data)
			pdata.data = make([]byte, len(pkg_data))
			pdata.key = pclient.key
			copy(pdata.data, pkg_data)
			pdata.idx = pclient.index
			pdata.flag = lnet.PkgOption(tag)
			pdata.info = QUEUE_INFO_NORMAL
			if pdata.flag == lnet.PKG_OP_NORMAL { //normal pkg to upper
				pserv.recv_queue <- pdata
			} else { //other optional pkg
				pclient.handle_spec_pkg(pconfig, pserv, pdata)
			}
			log.Debug("%s success! tag:%x pkg_data:%v pkg_len:%d pkg_option:%d", _func_, tag, pkg_data, pkg_len,
				pdata.flag)
		}

		//forward
		raw_data = raw_data[pkg_len:]
		if len(raw_data) <= 0 {
			pclient.recv_cache = pclient.recv_cache[:0] //clear cache
			//log.Debug("%s no more data!" , _func_);
			//recv may expand cache cap
			if cap(pclient.recv_cache) > CLIENT_RW_CACHE {
				new_cap := cap(pclient.recv_cache) / 2
				if new_cap < CLIENT_RW_CACHE {
					new_cap = CLIENT_RW_CACHE
				}
				log.Info("%s shrink recv_cache from %d to %d idx:%d", _func_, cap(pclient.recv_cache), new_cap)
				pclient.recv_cache = make([]byte, new_cap) //new buffer
				pclient.recv_cache = pclient.recv_cache[:0]
			}

			break
		}

	}

	return parsed
	//pclient.recv_cache = pclient.recv_cache[:0]; //reset
}

//send pkg to client
func (pclient *tcp_client) send(pconfig *CommConfig, pserv *TcpServ) {
	var _func_ = "<tcp_client.send>"
	log := pconfig.Log
	//check conn
	if pclient.stat == CLIENT_STAT_CLOSING {
		return
	}

	//flush cache 1st
	if len(pclient.snd_cache) > 0 {
		ret := pclient.flush_send_cache(pconfig, pserv)
		if ret < 0 {
			log.Err("%s 1st flush failed! will close conn. idx:%d", _func_, pclient.index)
			return
		}

		if ret == 1 {
			log.Debug("%s 1st send part of data! idx:%d", _func_, pclient.index)
			return
		}

		//ret == 0 empty cache
	}

	//check send_queue
	snd_len := len(pclient.snd_queue)
	if snd_len <= 0 {
		return
	}

	//log.Debug("%s snd_queue_len:%d idx:%d" , _func_ , snd_len , pclient.index);
	//handle each pkg
	var pdata *queue_data
	var pkg_len int
	var ret int
	var send_pkg int
	for i := 0; i < snd_len; i++ {
		//fetch a pkg
		pdata = <-pclient.snd_queue

		//check data option
		if pdata.info == QUEUE_INFO_CLOSE {
			log.Info("%s detect info close. will close this connection! c_key:%d idx:%d", _func_,
				pclient.key, pclient.index)
			pclient.stat = CLIENT_STAT_CLOSING
			pclient.server_close = true
			pserv.close_ch <- pclient.index
			return
		}

		//pack pkg
		pkg_len = lnet.PackPkg(pclient.snd_cache[:cap(pclient.snd_cache)], pdata.data, pdata.flag)
		if pkg_len < 0 {
			log.Err("%s pack error! will drop pkg! idx:%d", _func_, pclient.index)
			break
		}
		//log.Debug("%s pkg_len:%d" , _func_ , pkg_len);
		pclient.snd_cache = pclient.snd_cache[:pkg_len]

		//flush cache
		ret = pclient.flush_send_cache(pconfig, pserv)
		if ret < 0 {
			log.Err("%s 2nd flush failed! will close conn. idx:%d", _func_, pclient.index)
			return
		}

		if ret == 1 {
			log.Debug("%s 2nd send part of data! idx:%d", _func_, pclient.index)
			break
		}

		//continue send
		send_pkg++
	}

	log.Debug("%s send pkg:%d idx:%d", _func_, send_pkg, pclient.index)
}

/*parse read bytes to queue-data
@return: -1:error 0:complete 1:send part of data
*/
func (pclient *tcp_client) flush_send_cache(pconfig *CommConfig, pserv *TcpServ) int {
	var _func_ = "<tcp_client.flush_send_cache>"
	log := pconfig.Log
	raw_data := pclient.snd_cache
	conn := pclient.conn

	//send data
	_ = conn.SetWriteDeadline(time.Now().Add(CLIENT_RW_TIMEOUT * time.Millisecond))
	send_len, err := conn.Write(raw_data)
	//log.Debug("%s raw:%d send:%d err:%v" , _func_ , len(raw_data) , send_len , err);

	//check err
	if err != nil { //parse err
		//net err
		if net_err, ok := err.(net.Error); ok {
			if net_err.Temporary() || net_err.Timeout() { //buff full
				log.Debug("%s buff full idx:%d", _func_, pclient.index)
				return 1
			} else { //other
				log.Err("%s connection net.err other! will close conn! detail:%v index:%d", _func_, err, pclient.index)
			}
		} else if err == io.EOF { //write a closed connection
			log.Info("%s connection closed! index:%d", _func_, pclient.index)
		} else { //other
			log.Err("%s write meets an error! will close! err:%v index:%d", _func_, err, pclient.index)
		}

		//need close
		pclient.stat = CLIENT_STAT_CLOSING
		pserv.close_ch <- pclient.index
		return -1
	}

	//check data
	if send_len < len(raw_data) {
		log.Debug("%s not all data sended! send:%d should_send:%d idx:%d", _func_, send_len, len(raw_data), pclient.index)
		raw_data = raw_data[send_len:]
		copy(pclient.snd_cache[:cap(pclient.snd_cache)], raw_data[:])
		/*
				if copyed != len(raw_data) {
					log.Err("%s copy remaining cache failed! copyed:%d src:%d" , _func_ , copyed , len(raw_data));
					pclient.stat = CLIENT_STAT_CLOSING;
			        pserv.close_ch <- pclient.index;
			        return -1;
				}*/
		pclient.snd_cache = pclient.snd_cache[:len(raw_data)]
		return 1
	}

	pclient.snd_cache = pclient.snd_cache[:0]
	return 0
}
