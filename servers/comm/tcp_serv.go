package comm

import (
  "net"
  "time"
  "sync"
  "io"
  local_net "sgame/lib/net"
)

const (
	CLIENT_STAT_CLOSING=0
	CLIENT_STAT_NORMAL=1
	
	MAX_QUEUE_DATA=(200*1024) //max queue data
    RECV_QUEUE_LEN=1024
    SND_QUEUE_LEN=1024
    
    MAX_PKG_PER_RECV = 20 //max pkg per recv
)




type queue_data struct {
	flag int8
	idx int //index of client
	data []byte
}

type tcp_client struct {
	stat int8 //refer CLIENT_STAT_XX
	conn *net.TCPConn //connection
	index int //index in client-array
	recv_cache []byte
	snd_cache []byte
	//recv_queue chan *queue_data //client -> server
	//snd_queue chan *queue_data  //server -> client
}



type TcpServ struct {
	sync.Mutex
	addr string
	max_conn int
	curr_conn int
	close_ch chan int	//closing connection
	client_list [] *tcp_client //client
	exit_ch chan bool //exit channel
	loop_index int
	recv_queue chan *queue_data //client -> server
	snd_queue chan *queue_data  //server -> client
}

//new tcp_serv and start listener serve
func StartTcpServ(pconfig *CommConfig , addr string , max_conn int) *TcpServ{
	var _func_ = "<StartTcpServ>";
    log := pconfig.Log;
    
    //new
    pserv := new(TcpServ);
    if pserv == nil {
    	log.Err("%s at %s failed! new error!" , _func_ , addr);
    	return nil;
    }
    
    //init
    pserv.addr = addr;
    pserv.max_conn = max_conn;
    pserv.curr_conn = 0;
    pserv.close_ch = make(chan int , pserv.max_conn);
    pserv.client_list = make([] *tcp_client , pserv.max_conn); //init
    pserv.exit_ch = make(chan bool , 2); //non-block
    pserv.recv_queue = make(chan *queue_data , RECV_QUEUE_LEN);
    pserv.snd_queue = make(chan *queue_data , SND_QUEUE_LEN)
    
    //start
    go tcp_server(pconfig, pserv);
    log.Info("%s at %s finish!" , _func_ , addr);
    return pserv;    	
}

func (pserv *TcpServ) Close(pconfig *CommConfig) {
	log := pconfig.Log;
	log.Info("tcp_serv closing... addr:%s" , pserv.addr);
	pserv.exit_ch <- true;	
}

func (pserv *TcpServ) Recv(pconfig *CommConfig) [][]byte{
	var _func_ = "<tcp_serv.Recv>";
	var pdata *queue_data;
	//log := pconfig.Log;		
	result_len := len(pserv.recv_queue);
	
	//empty
	if result_len <= 0 {
		return nil;
	}	
	
	//init results			
	if result_len > MAX_PKG_PER_RECV { //max-20pkgs per handle
		result_len = MAX_PKG_PER_RECV;
	}
	results := make([][]byte , result_len);
	i := 0;
	
	//fulfill result
	for {
		if i >= result_len {
			break;
		}
		
		select {
			case pdata = <- pserv.recv_queue:
			    results[i] = pdata.data;
			default:
			    //nothing;
		}
		
		i++;
	}
	
	//log.Debug("%s recv pkg:%d" , _func_ , result_len);
	return results;	
}


/*--------------------------Static Func----------------------------*/
/*--------------------------------tcp_server-----------------------------*/
//go routine of listener
func tcp_server(pconfig *CommConfig , pserv *TcpServ) {
	var _func_ = "<tcp_server>";
	var log = pconfig.Log;
	
	log.Info("%s starting..." , _func_);
	//addr
	serv_addr , err := net.ResolveTCPAddr("tcp4", pserv.addr);
	if err != nil {
		log.Err("resolve addr:%s failed! err:%v", pserv.addr , err);
		return;
	}
	
	
	//listen
	listener , err := net.ListenTCP("tcp4", serv_addr);
	if err != nil {
		log.Err("%s listen at %s failed! err:%v" , _func_ , pserv.addr , err);
		return;
	}
	
	log.Info("%s listen at %s done!" , _func_ , pserv.addr);
	//accept
	for {
		//check exit
		select {
			case <- pserv.exit_ch:
			    log.Info("%s detect exit flg!" , _func_);
			    return;
			default:
			    //nothing    
		}
		
		//set deadline 1second
		listener.SetDeadline(time.Now().Add(1 * time.Second));
		      	
      	//acc
      	conn , err := listener.AcceptTCP();
      	if err != nil {
      		if net_err , ok := err.(net.Error); ok{
      			if net_err.Temporary() || net_err.Timeout() { //timeout no-err
      				//log.Debug("%s time out." , _func_);
      			} else {
      				log.Err("%s accept net-err fail! err:%v" , _func_ , err);
      				conn.Close();
      			}
      		} else {
      			log.Err("%s accept fail! err:%v" , _func_ , err);
      			conn.Close();
      		}
      	} else {     	
      	    //wrap connection
      	    log.Info("%s accept a new connection! peer:%s" , _func_ , conn.RemoteAddr().String());
      	
      	    //add client
      	    pserv.add_client(pconfig, conn);
      	}
      	
      	//close clients
      	pserv.close_clients(pconfig);
      		
	}
		
}


//detect and close closing clients
func (pserv *TcpServ) close_clients(pconfig *CommConfig) {
	var _func_ = "<tcp_serv.close_clients>"
	log := pconfig.Log;
	
	//check closing
	if len(pserv.close_ch) <= 0 {
		return;
	}
	
	//close
	for  {
		select {
			case idx := <- pserv.close_ch:    
			    pclient := pserv.client_list[idx];
			    close_client(pconfig, pclient);
			    pserv.client_list[idx] = nil;
			    pserv.curr_conn--;
			    log.Info("%s will close connection idx:%d curr_count:%d" , _func_ , idx , pserv.curr_conn);
			default:
			    return;   
		}
	}
		
}


func close_connection(pconfig *CommConfig , conn *net.TCPConn) {
	conn.Close();
}

//add a client to tcp_serv
func (pserv *TcpServ) add_client(pconfig *CommConfig , conn *net.TCPConn) {
	var _func_ = "<tcp_serv.add_client>";
	log := pconfig.Log;
	
	//check conn count
	if pserv.curr_conn >= pserv.max_conn {
		log.Err("%s fail! connection count:%d reached uplimit!" , _func_ , pserv.curr_conn);
		close_connection(pconfig, conn);
		return;
	}
	
	//search an empty index
	var i = 0;
	pserv.Lock();
	for i=0; i<pserv.max_conn; i++ {
		if pserv.client_list[i] == nil {
			break;
		}
	}
	pserv.Unlock();
	if i>= pserv.max_conn {
		log.Err("%s fail! no empy pos found! curr:%d max:%d" , _func_ , pserv.curr_conn , pserv.max_conn);
		close_connection(pconfig, conn);
		return;
	}
	
	//new client
	pclient := new_client(pconfig);
	if pclient == nil {
		log.Err("%s fail! new_client failed!" , _func_);
		close_connection(pconfig, conn);
		return;
	}
	pclient.index = i;
	pclient.conn = conn;
	pclient.stat = CLIENT_STAT_NORMAL;
	
	//add client
	pserv.Lock();
	pserv.client_list[i] = pclient;
	pserv.curr_conn++;
	pserv.Unlock();
	
	
	log.Info("%s at %s success! index:%d curr_count:%d" , _func_ , pserv.addr , i , pserv.curr_conn);
	go pclient.run(pconfig , pserv);
	return;
}


/*--------------------------------tcp_client-----------------------------*/
//create a new tcp_client
func new_client(pconfig *CommConfig) *tcp_client {
	var _func_ = "<new_client>";
	log := pconfig.Log;
	
	//new
	pclient := new(tcp_client);
	if pclient == nil {
		log.Err("%s fail! alloc tcp_client failed!" , _func_);
		return nil;
	}
	
	pclient.recv_cache = make([]byte , MAX_QUEUE_DATA*2); //pre alloc cache
	pclient.recv_cache = pclient.recv_cache[:0];	
	pclient.snd_cache = make([]byte , MAX_QUEUE_DATA*2);
	pclient.snd_cache = pclient.snd_cache[:0];
	//pclient.recv_queue = make(chan *queue_data , RECV_QUEUE_LEN);
	//pclient.snd_queue = make(chan *queue_data , SND_QUEUE_LEN);
	return pclient;
}

//close a tcp_client
func close_client(pconfig *CommConfig , pclient *tcp_client) {
	var _func_ = "<close_client>";
	log := pconfig.Log;
	
	//close connection
	err := pclient.conn.Close();
	if err != nil {
		log.Err("%s close conn err:%v idx:%d" , _func_ , err , pclient.index);
	}
	
	//clear data
	pclient.conn = nil;
	pclient.snd_cache = nil;
	pclient.recv_cache = nil;
	//pclient.snd_queue = nil;
	//pclient.recv_queue = nil;
	return;
}


//client go-routine
func (pclient *tcp_client) run(pconfig *CommConfig , pserv *TcpServ) {
	
	for {
		//exit-goroutine
		if pclient.stat == CLIENT_STAT_CLOSING {
			break;
		}
		
		//read bytes
		pclient.read(pconfig , pserv);
		
		//sleep
		time.Sleep(5 * time.Second);
	}
		
}



func (pclient *tcp_client) read(pconfig *CommConfig , pserv *TcpServ) {
	var _func_ = "<tcp_client.read>";
	var read_len = 0;
	var parsed int = 0;
	var err error;
	
	log := pconfig.Log;
	conn := pclient.conn;
	read_data := make([]byte , MAX_QUEUE_DATA);
	
	if pclient.stat == CLIENT_STAT_CLOSING {
		log.Info("%s in closing! index:%d" , _func_ , pclient.index);
		return;
	}
	
	//consist read
	for {
		read_data = read_data[:cap(read_data)];
		//set dead line 10ms
		conn.SetReadDeadline(time.Now().Add(5 * time.Second)); //test
		
		//read
		read_len , err = conn.Read(read_data);
		  //check err
		if err != nil { //parse err
		    close_conn := true;
		    //net err			
			if net_err , ok := err.(net.Error); ok {
				if net_err.Temporary() || net_err.Timeout() { //no data prepared
				    //log.Debug("%s no data" , _func_);
				    close_conn = false;
				} else { //other		
					log.Err("%s connection net.err other! will close conn! detail:%v index:%d" , _func_ , err , pclient.index);
				}
			} else if err == io.EOF { //read a closed connection
				log.Info("%s connection closed! index:%d" , _func_ , pclient.index);
			} else { //other
				log.Err("%s read meets an error! will close! err:%v index:%d" , _func_ , err , pclient.index);
			}
			
			//need close
			if close_conn {
			    pclient.stat = CLIENT_STAT_CLOSING;
			    pserv.close_ch <- pclient.index;
			}
			return;
		}
		
		//empty useless
		if read_len == 0 {
			log.Info("%s peer closed will close connection index:%d!" , _func_ , pclient.index);
			pclient.stat = CLIENT_STAT_CLOSING;
			pserv.close_ch <- pclient.index;			
			return;
		}
		
		//append readed
		log.Debug("%s read %d bytes" , _func_ , read_len);
		pclient.recv_cache = append(pclient.recv_cache , read_data[:read_len]...);
		
		
		//flush cache
		parsed = pclient.flush_recv_cache(pconfig , pserv);
		if parsed < 0 {
			log.Err("%s pkg data illegal! will close connection! idx:%d" , _func_ , pclient.index);
			pclient.stat = CLIENT_STAT_CLOSING;
			pserv.close_ch <- pclient.index;
			return;
		}
		log.Debug("%s flush %d pkgs! idx:%d" , _func_ , parsed , pclient.index);
		
	}
	
}

/*parse read bytes to queue-data
@return: -1:error 0:not ready else:handled count of pkg
*/
func (pclient *tcp_client) flush_recv_cache(pconfig *CommConfig , pserv *TcpServ) int {
	var _func_ = "<tcp_client.flush_recv_cache>";
	log := pconfig.Log;
	var tag uint8;
	var pkg_data []byte;
	var pkg_len int;
	var raw_data []byte;
	var parsed int;
	
	raw_data = pclient.recv_cache;
	for {
	    //unpack raw data
	    tag , pkg_data , pkg_len = local_net.UnPackPkg(raw_data);
	    
	    //error
	    if tag == 0xFF {
		    log.Err("%s error! pkg format illegal!" , _func_);
		    return -1;
	    }
	
	    //not- ready
	    if tag == 0 {
		    log.Debug("%s pkg not ready!" , _func_);
		    
		    //reset cache
            copyed := copy(pclient.recv_cache[:cap(pclient.recv_cache)] , raw_data[:]);
            if copyed != len(raw_data) {
            	log.Err("%s reset recv_cache failed! copy:%d raw:%d" , _func_ , copyed , len(raw_data));
            	return -1;
            }
            log.Debug("%s shrink cached %d bytes" , _func_ , copyed);		    
		    break;
	    }
	
	    //parse success
	    parsed++;
	
	    //put  queue
	    pdata := new(queue_data);
	    pdata.data = make([]byte , pkg_len);
	    copy(pdata.data , pkg_data);
	    pdata.idx = pclient.index;
	    pserv.recv_queue <- pdata;
	    log.Debug("%s success! tag:%x pkg_data:%s pkg_len:%d queue_len:%d" , _func_ , tag , string(pkg_data) , pkg_len , 
	    	len(pserv.recv_queue));
	
	    //forward
	    raw_data = raw_data[pkg_len:];
	    if len(raw_data) <= 0 {
		    pclient.recv_cache = pclient.recv_cache[:0]; //clear cache
		    log.Debug("%s no more data!" , _func_);
		    break;
	    }
	    
	}
	
	return parsed;
	//pclient.recv_cache = pclient.recv_cache[:0]; //reset
}

//send
func (pclient *tcp_client) send(pconfig *CommConfig , pserv *TcpServ) {
}

/*parse read bytes to queue-data
@return: -1:error 0:complete 1:send part of data
*/
func (pclient *tcp_client) flush_send_cache(pconfig *CommConfig) int {
	var _func_ = "<tcp_client.flush_send_cache>";
	var pkg_data []byte;
			
	log := pconfig.Log;
	pkg_data = pclient.snd_cache;
	conn := pclient.conn;

	//send data
	send_len , err := conn.Write(pkg_data);
	
	//check err
	if err != nil { //parse err
	    //net err			
		if net_err , ok := err.(net.Error); ok {
			if net_err.Temporary() || net_err.Timeout() { //no data prepared
			    //log.Debug("%s no data" , _func_);
			    return 1; //no data sended
			} else { //other		
				log.Err("%s connection net.err other! will close conn! detail:%v index:%d" , _func_ , err , pclient.index);
			}
		} else if err == io.EOF { //write a closed connection
			log.Info("%s connection closed! index:%d" , _func_ , pclient.index);
		} else { //other
			log.Err("%s write meets an error! will close! err:%v index:%d" , _func_ , err , pclient.index);
		}
			
		//need close
		if close_conn {
		    pclient.stat = CLIENT_STAT_CLOSING;
		    pserv.close_ch <- pclient.index;
		}
		return -1;
	}
		
	//check data
	if send_len < len(pkg_data) {
		log.Debug("%s not all data sended! send:%d should:%d" , _func_ , send_len , len(pkg_data));
		copy(pclient.snd_cache[:cap(pclient.snd_cache)] , pack_data[send_len:]);
		return 1;
	}  
			
	return 0;
}