package main 

import (
	"fmt"
	"net"
	"os"
	lnet "sgame/lib/net"
	"sgame/proto/cs"
	"time"
	"flag"
	"strconv"
)

const (
  CMD_PING="ping"
  CMD_LOGIN="login"
  
  BUFF_LEN=(10*1024)
)

var cmd_map map[string]string;
var buff_len int = 1024;
var exit_ch chan bool;

//flag
var host = flag.String("h", "127.0.0.1", "server ip");
var port = flag.Int("p", 0, "server port");
var option = flag.Int("m", 0, "method 1:interace 2:command");
var cmd = flag.String("c", "", "cmd");

func init() {
	cmd_map = make(map[string]string);
	//init cmd_map
	cmd_map[CMD_PING] = "ping to server";
	cmd_map[CMD_LOGIN]="login";
}

func show_cmd() {
	fmt.Printf("----cmd----\n");
	for cmd , info := range cmd_map {
		fmt.Printf("[%s] %s\n", cmd , info);
	}
}


func RecvPkg(conn *net.TCPConn) {
	read_buff := make([]byte , 1024);	
	for {
		time.Sleep(10 * time.Millisecond);
		if len(exit_ch) > 0 {
			break;
		}
				
		read_buff = read_buff[:cap(read_buff)];		
		//read
		n , err := conn.Read(read_buff);
		if err != nil {
			fmt.Printf("read failed! err:%v\n", err);
			break;
		}

		//unpack
		_ , pkg_data , _ := lnet.UnPackPkg(read_buff[:n]);
		//fmt.Printf("read tag:%d pkg_data:%v pkg_len:%d pkg_option:%d\n", tag , pkg_data , pkg_len , lnet.PkgOption(tag));
		
		//decode
		var gmsg cs.GeneralMsg;
		err = cs.DecodeMsg(pkg_data, &gmsg);
		if err != nil {
			fmt.Printf("decode failed! err:%v\n", err);
			continue;
		}
		
		//switch rsp
		switch gmsg.ProtoId {
			case cs.CS_PROTO_PING_RSP:
			  prsp , ok := gmsg.SubMsg.(*cs.CSPingRsp);
			  if ok {
			    curr_ts := time.Now().UnixNano()/1000;	
			    fmt.Printf("ping:%v ms\n", (curr_ts-prsp.TimeStamp)/1000);
			  }
			  return;
			default:
			  fmt.Printf("illegal proto:%d\n", gmsg.ProtoId);
		}
				
	}
}

//send pkg to server
func SendPkg(conn *net.TCPConn , cmd string) {
	var gmsg cs.GeneralMsg;
	var err error;
	var enc_data []byte;
	
    pkg_buff := make([]byte , BUFF_LEN);        
    //encode msg
	switch cmd {
	    case CMD_PING:
	        fmt.Printf("ping...\n");
	        gmsg.ProtoId = cs.CS_PROTO_PING_REQ;
	        psub := new(cs.CSPingReq);
	        psub.TimeStamp = time.Now().UnixNano()/1000;
	        gmsg.SubMsg = psub;
	        	        
	    case CMD_LOGIN:
	        fmt.Printf("login...\n");
	        gmsg.ProtoId = cs.CS_PROTO_LOGIN_REQ;
	        psub := new(cs.CSLoginReq);
	        psub.Name = "cs";
	        psub.Device = "onepluse9";
	        psub.Pass = "123";
	        gmsg.SubMsg = psub;    
	    default:
	        fmt.Printf("illegal cmd:%s\n", cmd);    
	        return;	        	
	}
	
	//encode
	enc_data , err = cs.EncodeMsg(&gmsg);
	if err != nil {
	    fmt.Printf("encode %s failed! err:%v\n", cmd , err);
	    return
	}
	
	//pack 
	pkg_len := lnet.PackPkg(pkg_buff, enc_data, 0);
	if pkg_len < 0 {
		fmt.Printf("pack cmd:%s failed!\n", cmd);
		return;
	}
	
	//send	
	 _ , err = conn.Write(pkg_buff[:pkg_len]);
	 if err != nil {
	     fmt.Printf("send cmd pkg failed! cmd:%s err:%v\n", cmd , err);
	 } else {
	 	fmt.Printf("send cmd:%s success! pkg:%v pkg_len:%d json:%s\n", cmd , pkg_buff[:pkg_len] , pkg_len , string(enc_data));
	 }
}





func main() {
	flag.Parse();
	if *port <= 0 || (*option != 1 && *option != 2) {
		flag.PrintDefaults();
		show_cmd();
		return;
	}
	 	
	fmt.Println("start client ...");
	//server_addr := "localhost:18909";
	server_addr := *host + ":" + strconv.Itoa(*port);
	
	//init
	exit_ch = make(chan bool , 1);
		
	//connect
	tcp_addr  , err := net.ResolveTCPAddr("tcp4", server_addr);
    if err != nil {
    	fmt.Printf("resolve addr:%s failed! err:%s\n", server_addr  ,err);
    	return;
    }

    conn , err := net.DialTCP("tcp4",  nil , tcp_addr);
    if err != nil {
    	fmt.Printf("connect %s failed! err:%v\n", server_addr , err);
    	return;
    }
	defer conn.Close();
	
	rs := make([]byte , 128);
	pack_buff := make([]byte , 128);	
	//read
	go RecvPkg(conn);
	
	//check option
	switch *option {
	case 1:
	    for ;; {
		    rs = rs[:cap(rs)];
		    fmt.Printf("please input:>>");
		    n , _ := os.Stdin.Read(rs);
		    rs = rs[:n-1]; //trip last \n
		
		    if string(rs) == "exit" {
			    fmt.Println("byte...");
			    exit_ch <- true;
			    break;
		    }
		
		    pkg_len := lnet.PackPkg(pack_buff, rs , (uint8)(*option));
		    //fmt.Printf("read %d bytes and packed:%d\n", n , pkg_len);		
		    n , _ = conn.Write(pack_buff[:pkg_len]);
		    time.Sleep(50 * time.Millisecond);
	    }
	    
	case 2:
	    if len(*cmd) <= 0 {
	    	show_cmd();
	    	break;
	    }
	    SendPkg(conn, *cmd);
	    time.Sleep(2*time.Second);
	default:
	    fmt.Printf("option:%d nothing tod!\n", *option);        
	}

	return;
}

