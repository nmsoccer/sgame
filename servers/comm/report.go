package comm

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
	"math/rand"
)


const (
	//if manger count > 1 or no use
	REPORT_METHOD_SEQ = 1 //report status to manager which is latest valid
    REPORT_METHOD_ALL = 2 //report status to all manager 
    REPORT_METHOD_MOD = 3 //report status to proc_id%manager_count to manager
    REPORT_METHOD_RAND = 4 //report status to a rand manager
    
    //MSG CHAN BUFFER
    REPROT_MSG_CHAN_SIZE = 100
    //MAX SEND&RECV
    MAX_SEND_PER_TICK = 10
    MAX_RECV_PER_TICK = 10
)


type manager_server struct {
	conn *net.UDPConn
	addr string	
}

type ReportServ struct {
	sync.Mutex
	pconfig *CommConfig
	method int8
	proc_id int
	proc_name string
    server_list []*manager_server
    exit_ch chan bool
    msg_send chan *ReportMsg
    msg_recv  chan *ReportMsg
	monitor_interval int
}


/*
Start a Report Go-Routine
* @proc_id process proc id
* @proc_name process proc_name
* @manage_addr manage servers addr, it could be many servers. addr like: "ip:port"
* @moniter_inv monitor interval of monitor goroutine. -1:no monitor =0:no sleep(at least 1 second sleep) >0 sleep seconds
*/
func StartReport(pconfig *CommConfig , proc_id int , proc_name string , manage_addr []string , method int8 , monitor_inv int) *ReportServ{
	var _func_ = "<StartReport>"
	log := pconfig.Log;
	
	//init env
	pserv := new(ReportServ);
	pserv.pconfig = pconfig;
	pserv.proc_id = proc_id;
	pserv.proc_name = proc_name;
	pserv.method = method;
	pserv.monitor_interval = monitor_inv;
	pserv.server_list = make([]*manager_server , len(manage_addr));
	for i:=0; i<len(pserv.server_list); i++ {
		pserv.server_list[i] = new(manager_server);
		pserv.server_list[i].addr = manage_addr[i];
	}
	pserv.exit_ch = make(chan bool , 2);
	pserv.msg_send = make(chan *ReportMsg , REPROT_MSG_CHAN_SIZE);
	pserv.msg_recv = make(chan *ReportMsg , REPROT_MSG_CHAN_SIZE);
	
	//start report_serv
	log.Info("%s success! proc_id:%d proc_name:%s addr:%v method:%d" , _func_ , pserv.proc_id , pserv.proc_name , pserv.server_list , pserv.method);
	pserv.connect_all();
	pserv.serve();
	go pserv.monitor();
	return pserv;
}

func (pserv *ReportServ) Close() {
	pserv.exit_ch <- true;
}

/*
* Report Msg To Manager
* @proto:Refer REPORT_PROTO_XX
* @v_msg: used for complex information. should be defined in report_proto.go
*/
func (pserv *ReportServ) Report(proto int , v_int int64 , v_str string , v_msg interface{}) bool {
    var	_func_ = "<ReportServ.Report>";
    log := pserv.pconfig.Log;
    
    //check chan
    if len(pserv.msg_send) >= cap(pserv.msg_send) {
    	log.Err("%s faild! send channel full:%d vs %d!" , _func_ , len(pserv.msg_send) , cap(pserv.msg_send));
    	return false;
    }
    
    //check manager
    if len(pserv.server_list) <= 0 {
    	log.Err("%s failed! no manager server valid!" , _func_);
    	return false;
    }


    //report msg
    var preport = new(ReportMsg);
    preport.ProtoId = proto;
    preport.ProcId = pserv.proc_id;
    preport.IntValue = v_int;
    preport.StrValue = v_str;
    preport.Sub = v_msg;
    
    //send   
    pserv.msg_send <- preport;
    return true;
}


func (pserv *ReportServ) Recv() *ReportMsg {
	if len(pserv.msg_recv) <= 0 {
		return nil;
	}

	return <- pserv.msg_recv;
}

//set monitor interval
func (pserv *ReportServ) SetMonitor(interval int) {
    pserv.Lock();
    pserv.monitor_interval = interval;
    pserv.Unlock();
}

/*---------------------------------STATIC FUNC------------------------------*/
//serve
func (pserv *ReportServ) serve() {
	go func() {
		var _func_ = "serv.serv";
		log := pserv.pconfig.Log;
	    for {
		    time.Sleep(10 * time.Millisecond);
		    //check exit
		    if len(pserv.exit_ch) > 0 {
		    	log.Info("%s exit!" , _func_);
		    	break;
		    }
		    
		    //send report
		    pserv.send();
		    
		    //recv report
		    pserv.recv();
	    }
	}();
}

func (pserv *ReportServ) monitor() {
	var _func_ = "<monitor>"
	var stdout bytes.Buffer;
	var err error;
	var interv int;
	pid := os.Getpid();

	log := pserv.pconfig.Log;

	exe_cmd := fmt.Sprintf("top -b -p %d -n 1" , pid);
    for {
    	interv = pserv.monitor_interval;
    	//<0 do nothing  just check per 10seconds
    	if interv < 0 {
    		time.Sleep(10 * time.Second)
    		continue;
		}

		//get top info
        stdout.Reset();
		cmd := exec.Command("bash" , "-c" , exe_cmd);
		cmd.Stdout = &stdout;
		cmd.Stderr = nil;
    	err = cmd.Run();
    	if err != nil {
    		log.Err("%s run command:%s fail! err:%v" , _func_ , exe_cmd , err);
		} else {
            pserv.Report(REPORT_PROTO_MONITOR , 0 , stdout.String() , nil);
		}

		//sleep
		if interv == 0 {
			interv = 1; //at least 1second sleep
		}
		time.Sleep(time.Duration(interv) * time.Second);

	}
}



//connect all
func (pserv *ReportServ) connect_all() {
	var _func_ = "serv.connect_all";
	log := pserv.pconfig.Log;
	
	//connect all
	for i:=0; i<len(pserv.server_list); i++ {
		serv_addr , err := net.ResolveUDPAddr("udp", pserv.server_list[i].addr);
		if err != nil {
			log.Err("%s resolve addr:%s failed! err:%v" , _func_ , pserv.server_list[i].addr , err);
			continue;
		}
		
		//dial
		conn , err := net.DialUDP("udp", nil , serv_addr);
		if err != nil {
			log.Err("%s dial manager failed! addr:%s err:%v" , _func_ , pserv.server_list[i].addr , err);
			continue;
		}
		
		//save
		pserv.server_list[i].conn = conn;
		log.Info("%s dial manager %s local:%s success!" , _func_ , pserv.server_list[i].addr ,
			conn.LocalAddr().String());
	}
	
}

//send report
func (pserv *ReportServ) send() {
	var _func_ = "report_serv.send";
	log := pserv.pconfig.Log;
	
		
	//set max
	var max_send  = len(pserv.msg_send);
	if max_send > MAX_SEND_PER_TICK {
		max_send = MAX_SEND_PER_TICK;
	}
	
	if max_send <= 0 {
		return;
	}
	//send
	var pmsg *ReportMsg;
	var count int;
	for  {
		if count >= max_send {
			break;
		}
		
		//get msg
		select {
			case pmsg = <- pserv.msg_send:
			    break;
			default:
			    return;
		}
		
		//check serv
		if len(pserv.server_list) <= 0 {
			count += 1;
			continue;
		}
				
		//enc msg
		enc_data , err := EncodeReportMsg(pmsg);
		if err != nil {
			log.Err("%s enc msg failed! proto:%d i_v:%d s_v:%s err:%v" , _func_ , pmsg.ProtoId , pmsg.IntValue , pmsg.StrValue , err);
			count += 1;
			continue;
		}
			
		//send
		switch pserv.method {
		case REPORT_METHOD_SEQ:	
		    _ , err = pserv.server_list[0].conn.Write(enc_data);
		    if err != nil {
			    //log.Err("%s send to %s failed! err:%v" , _func_ , pserv.server_list[0].addr , err);
		    } else {
			    //log.Debug("%s send to %s success! send:%d" , _func_ , pserv.server_list[0].addr , n);
		    }
		case REPORT_METHOD_ALL:
		    for i:=0; i<len(pserv.server_list); i++ {
		    	_ , err = pserv.server_list[i].conn.Write(enc_data);
		    	if err != nil {
			        //log.Err("%s send to %s failed! err:%v" , _func_ , pserv.server_list[i].addr , err);
		        } else {
			        //log.Debug("%s send to %s success! send:%d" , _func_ , pserv.server_list[i].addr , n);
		        }
		    }
		case REPORT_METHOD_MOD:
		    pos := pserv.proc_id % len(pserv.server_list);
		    _ , err = pserv.server_list[pos].conn.Write(enc_data);
		    if err != nil {
			    //log.Err("%s send to %s failed! err:%v" , _func_ , pserv.server_list[pos].addr , err);
		    } else {
			    //log.Debug("%s send to %s success! send:%d" , _func_ , pserv.server_list[pos].addr , n);
		    }
		    
		case REPORT_METHOD_RAND:
		    pos := rand.Int() % len(pserv.server_list);
		    _ , err = pserv.server_list[pos].conn.Write(enc_data);
		    if err != nil {
			    //log.Err("%s send to %s failed! err:%v" , _func_ , pserv.server_list[pos].addr , err);
		    } else {
			    //log.Debug("%s send to %s success! send:%d" , _func_ , pserv.server_list[pos].addr , n);
		    }        
		default:
		    //nothing                
		} 
		
		
		count += 1;
	}
	
	
}

//recv report
func (pserv *ReportServ) recv() {
	var _func_ = "<report_serv.recv>";
	log := pserv.pconfig.Log;

    //set max
	var max_recv  = cap(pserv.msg_recv) - len(pserv.msg_recv);
	if max_recv > MAX_RECV_PER_TICK {
		max_recv = MAX_RECV_PER_TICK;
	}
	
	if max_recv <= 0 {
		log.Err("%s warning recv channel full! len:%d" , _func_ , len(pserv.msg_recv));
		return;
	}

    //recv    
	var count int;
	var buff = make([]byte , MAX_REPORT_MSG_LEN);
	for i:=0; i<len(pserv.server_list); i++ {
		pmanager := pserv.server_list[i];
        for {
    	    if count >= max_recv {
    		    break;
    	    }
    	    buff = buff[:cap(buff)];
    	    //set dead line 
		    pmanager.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); //set timeout
    	
    	    //recv
    	    n , err := pmanager.conn.Read(buff);
    	    if err != nil {
    		    break;
    	    }
    	    if n == 0 {
    		    log.Debug("%s addr %s no pkg!" , _func_ , pmanager.addr);
    		    break;
    	    }
    	
    	    //decode
    	    pmsg := new(ReportMsg);
    	    err = DecodeReportMsg(buff[:n], pmsg);
    	    if err != nil {
    		    log.Err("%s decode from %s failed! err:%v" , _func_ , pmanager.addr , err);
    		    count += 1;
    		    continue;
    	    }
    	
    	    //append
    	    //log.Debug("%s from:%s recv proto:%d iv:%d sv:%s" , _func_ , pmanager.addr , pmsg.ProtoId , pmsg.IntValue , pmsg.StrValue);
    	    pserv.msg_recv <- pmsg;
    	    count += 1;
        }
	}

}
