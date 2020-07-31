package main 

import (
	"fmt"
	"net"
	"os"
	local_net "sgame/lib/net"
	"sgame/proto/cs"
	"time"
	"flag"
	"strconv"
)

var buff_len int = 1024;
//var exit_ch chan bool;

//flag
//var host = flag.String("h", "127.0.0.1", "server ip");
//var port = flag.Int("p", 0, "server port");
//var option = flag.Int("t", 0, "option 1:echo 2:stat");

func RecvClient(conn *net.TCPConn) {
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
		tag , pkg_data , _ := local_net.UnPackPkg(read_buff[:n]);
		//fmt.Printf("tag:%d pkg_data:%s pkg_len:%d pkg_option:%d\n", tag , string(pkg_data) , pkg_len , local_net.PkgOption(tag));
		switch local_net.PkgOption(tag) {
			case local_net.PKG_OP_ECHO:
			    fmt.Printf("[%s]\n", string(pkg_data));
			case local_net.PKG_OP_STAT:
			    fmt.Printf("%s\n", string(pkg_data));    
			default:
		}
		
		
	}
}



func main() {
	flag.Parse();
	if *port <= 0 || *option < local_net.PKG_OP_NORMAL || *option > local_net.PKG_OP_MAX {
		flag.PrintDefaults();
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
	go RecvClient(conn);
	
	//check option
	switch *option {
	case local_net.PKG_OP_ECHO , local_net.PKG_OP_NORMAL:
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
		
		    pkg_len := local_net.PackPkg(pack_buff, rs , (uint8)(*option));
		    //fmt.Printf("read %d bytes and packed:%d\n", n , pkg_len);		
		    n , _ = conn.Write(pack_buff[:pkg_len]);
		    time.Sleep(50 * time.Millisecond);
	    }
	    
	case local_net.PKG_OP_STAT:
	    copyed := copy(rs , []byte(local_net.FETCH_STAT_KEY));
	    rs = rs[:copyed];
	    pkg_len := local_net.PackPkg(pack_buff, rs, local_net.PKG_OP_STAT);
	    _ , _ = conn.Write(pack_buff[:pkg_len]);
	    time.Sleep(5 * time.Second);    
	}

	return;
}

