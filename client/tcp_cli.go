package main 

import (
	"fmt"
	"net"
	"os"
	local_net "sgame/lib/net"
)

func main() {
	fmt.Println("start client ...");
	server_addr := "localhost:18909";
		
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
	//ws := make([]byte , 128);
	pack_buff := make([]byte , 128);
	
	for ;; {
		rs = rs[:cap(rs)];
		fmt.Printf("please input:>>");
		n , _ := os.Stdin.Read(rs);
		rs = rs[:n-1]; //trip last \n
		
		pkg_len := local_net.PackPkg(pack_buff, rs);
		fmt.Printf("read %d bytes and packed:%d\n", n , pkg_len);
		
		n , _ = conn.Write(pack_buff[:pkg_len]);
		fmt.Printf("write:%d\n", n);
		
		//n , _ = conn.Read(ws);
		//fmt.Printf("\nread:%s bytes:%d\n", string(ws[:n]) , n);
	}

	return;
}

