/*
 * simple_cli.go
 *
 * This is a Demo Client Connect SGame Server Using Go.
 * Test Ping Proto.
 *
 * Build:  go build simple_cli.go
           ./simple_cli -p <port>
 * More Info:https://github.com/nmsoccer/sgame/wiki/mulit-connect
 * Created on: 2020.8.6
 * Author: nmsoccer
*/

package main

import (
	"flag"
	"fmt"
	"net"
	lnet "sgame/lib/net"
	//"sgame/proto/cs"
	"strconv"
	"time"
)


const (
	BUFF_LEN = 10 * 1024
)


var host = "127.0.0.1";
var port = flag.Int("p", 0, "server port");

func ValidConnec(conn *net.TCPConn) bool {
	//pack
	pkg_buff := make([]byte , 128)
	pkg_len := lnet.PackPkg(pkg_buff , []byte(lnet.CONN_VALID_KEY) , lnet.PKG_OP_VALID)
	if pkg_len <= 0 {
		fmt.Printf("valid connection pack failed! pkg_len:%d\n" , pkg_len)
		return false
	}

	//send
	_, err := conn.Write(pkg_buff[:pkg_len])
	if err != nil {
		fmt.Printf("send valid pkg failed! err:%v\n", err)
		return false
	}
	fmt.Printf("send valid success! pkg_len:%d valid_key:%s\n", pkg_len , lnet.CONN_VALID_KEY)


	//read
	read_buff := make([]byte , BUFF_LEN);
	n, err := conn.Read(read_buff)
	if err != nil {
		fmt.Printf("read failed! err:%v\n", err)
		return false;
	}


	//Unpack
	tag , pkg_data, pkg_len := lnet.UnPackPkg(read_buff[:n])
	if(tag==0xFF){
		fmt.Printf("unpack failed!\n");
		return false;
	}
	if(tag == 0xEF){
		fmt.Printf("pkg_buff not enough!\n");
		return false;
	}
	if(tag == 0){
		fmt.Printf("data not ready!\n");
		return false;
	}
	fmt.Printf("<< from server:data_len:%d pkg_len:%d tag:%d option:%d\n" , len(pkg_data) , pkg_len ,
		tag , lnet.PkgOption(tag));

	return true
}



func main() {
	flag.Parse();
	if *port <= 0 {
		flag.PrintDefaults();
		return;
	}

	pkg_buff := make([]byte , BUFF_LEN);
	fmt.Printf("start client...\n");
	//addr
	server_addr := host + ":" + strconv.Itoa(*port);
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

	var cmd string
	curr_ts := time.Now().UnixNano()/1000000;
	//Valid Conn
	if ValidConnec(conn) == false {
		fmt.Printf("valid conn failed!")
		return
	}


	//Test Ping
	/*
	 * create json request refer sgame/proto/cs/: api.go and ping.proto.go
	 */
	//Encode  cmd
	cmd = fmt.Sprintf("{\"proto\":1 , \"sub\":{\"ts\":%d}}" , curr_ts);
	enc_data := []byte(cmd);

	//Pack
	pkg_len := lnet.PackPkg(pkg_buff, enc_data, 0)
	if pkg_len < 0 {
		fmt.Printf("pack failed!\n")
		return
	}

	//send
	_, err = conn.Write(pkg_buff[:pkg_len])
	if err != nil {
		fmt.Printf("send cmd pkg failed!  err:%v\n",  err)
	} else {
		fmt.Printf(">> send cmd:%s \n" , cmd)
	}

	//read
	read_buff := make([]byte , BUFF_LEN);
	n, err := conn.Read(read_buff)
	if err != nil {
		fmt.Printf("read failed! err:%v\n", err)
		return;
	}

    fmt.Printf("recv:%v\n" , read_buff[:n])
	//Unpack
	tag , pkg_data, pkg_len := lnet.UnPackPkg(read_buff[:n])
	if(tag==0xFF){
		fmt.Printf("unpack failed!\n");
		return;
	}
	if(tag == 0xEF){
		fmt.Printf("pkg_buff not enough!\n");
		return;
	}
	if(tag == 0){
		fmt.Printf("data not ready!\n");
		return;
	}
	fmt.Printf("<< from server:%s data_len:%d pkg_len:%d tag:%d option:%d\n" , string(pkg_data) ,  len(pkg_data) , pkg_len ,
		tag , lnet.PkgOption(tag));

	return;
}


