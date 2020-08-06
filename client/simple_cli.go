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

/*
 * simple_cli.go
 *
 * This is a Demo Client Connect SGame Server Using Go.
 * Test Ping Proto.
 *
 * Build:  go build simple_cli.go
 * More Info:https://github.com/nmsoccer/sgame/wiki/mulit-connect
 *  Created on: 2020.8.6
 *      Author: nmsoccer
 */


const (
	BUFF_LEN = 10 * 1024
)


var host = "127.0.0.1";
var port = flag.Int("p", 0, "server port");


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
	curr_ts := time.Now().Unix();
	//Test Ping
	/*
	 * create json request refer sgame/proto/cs/: api.go and ping.proto.go
	 */


	/*Encode Method 1 using csLib
	var gmsg cs.GeneralMsg
	gmsg.ProtoId = cs.CS_PROTO_PING_REQ
	psub := new(cs.CSPingReq)
	psub.TimeStamp = curr_ts;
	gmsg.SubMsg = psub

	//encode
	enc_data, err := cs.EncodeMsg(&gmsg)
	if err != nil {
		fmt.Printf("encode failed! err:%v\n", err)
		return
	}
	cmd = string(enc_data)
	*/

	//Encode Method 2 manually
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
		fmt.Printf("<< send cmd:%s \n" , cmd)
	}

	//read
	read_buff := make([]byte , BUFF_LEN);
	n, err := conn.Read(read_buff)
	if err != nil {
		fmt.Printf("read failed! err:%v\n", err)
		return;
	}


	//Unpack
	tag , pkg_data, pkg_len := lnet.UnPackPkg(read_buff[:n])
	fmt.Printf(">> from server:%s data_len:%d pkg_len:%d tag:%d\n" , string(pkg_data) ,  len(pkg_data) , pkg_len ,
		tag);

	/* more if wanna Decode json
	var gmsg2 cs.GeneralMsg
	err = cs.DecodeMsg(pkg_data, &gmsg2)
	if err != nil {
		fmt.Printf("decode failed! err:%v\n", err)
		return;
	}

	//print
	if gmsg2.ProtoId != cs.CS_PROTO_PING_RSP {
		fmt.Printf("proto:%d not valid!\n" , gmsg2.ProtoId);
		return;
	}
	prsp, ok := gmsg.SubMsg.(*cs.CSPingRsp)
	if ok {
		fmt.Printf("ping: ts:%d\n", prsp.TimeStamp)
	}
	 */

	time.Sleep(1 * time.Second);
	return;
}


