package main

import (
	"bytes"
	"compress/zlib"
	"flag"
	"fmt"
	"io"
	"net"
	lnet "sgame/lib/net"
	"sgame/proto/cs"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CMD_PING   = "ping"
	CMD_LOGIN  = "login"
	CMD_LOGOUT = "logout"
	CMD_REG    = "reg"

	BUFF_LEN = (10 * 1024)
)

var cmd_map map[string]string
var zlib_enalbe = true
var exit_ch chan bool = make(chan bool , 1)
var wg sync.WaitGroup


//flag
var help = flag.Bool("h" , false , "show help")
var host *string = flag.String("a", "127.0.0.1", "server ip")
var port = flag.Int("p", 0, "server port")
var cmd = flag.String("c", "", "cmd")
var keep = flag.Int("k" , 0 , "keepalive seconds if method=2")
var quiet = flag.Bool("q" , false , "quiet if method=2");
var total_count     = flag.Int("N" , 0 , "Total Number of requests");
var concurrent_count    = flag.Int("C" , 0 , "cocurrent per second")

func init() {
	cmd_map = make(map[string]string)
	//init cmd_map
	cmd_map[CMD_PING] = "ping to server"
	cmd_map[CMD_LOGIN] = "login <name> <pass>"
	cmd_map[CMD_LOGOUT] = "logout"
	cmd_map[CMD_REG] = "register <name> <pass> <sex:1|2> <addr>"
}

func v_print(format string , arg ...interface{}) {
	if ! * quiet {
		fmt.Printf(format , arg...);
	}
}


func show_cmd() {
	fmt.Printf("----cmd----\n")
	for cmd, info := range cmd_map {
		fmt.Printf("[%s] %s\n", cmd, info)
	}
}

func RecvPkg(conn *net.TCPConn) bool {
	read_buff := make([]byte, BUFF_LEN)
	for {
		time.Sleep(10 * time.Millisecond)

		read_buff = read_buff[:cap(read_buff)]
		//read
		n, err := conn.Read(read_buff)
		if err != nil {
			fmt.Printf("read failed! err:%v\n", err)
			return false;
		}

		//unpack
		_, pkg_data, _ := lnet.UnPackPkg(read_buff[:n])
		//fmt.Printf("read tag:%d pkg_data:%v pkg_len:%d pkg_option:%d\n", tag , pkg_data , pkg_len , lnet.PkgOption(tag));

		//uncompress
		if zlib_enalbe {
			b := bytes.NewReader(pkg_data);
			var out bytes.Buffer;
			r , err := zlib.NewReader(b);
			if err != nil {
				fmt.Printf("uncompress data failed! err:%v" , err);
				return false;
			}
			io.Copy(&out , r);
			pkg_data = out.Bytes();
		}


		//decode
		var gmsg cs.GeneralMsg
		err = cs.DecodeMsg(pkg_data, &gmsg)
		if err != nil {
			fmt.Printf("decode failed! err:%v\n", err)
			return false;
		}

		//switch rsp
		switch gmsg.ProtoId {
		case cs.CS_PROTO_PING_RSP:
			prsp, ok := gmsg.SubMsg.(*cs.CSPingRsp)
			if ok {
				curr_ts := time.Now().UnixNano() / 1000
				v_print("ping:%v ms\n", (curr_ts-prsp.TimeStamp)/1000)
				return true;
			}
		case cs.CS_PROTO_LOGIN_RSP:
			prsp, ok := gmsg.SubMsg.(*cs.CSLoginRsp)
			if ok {
				if prsp.Result == 0 {
					v_print("login result:%d name:%s\n", prsp.Result, prsp.Name)
					v_print("uid:%v sex:%d addr:%s level:%d Exp:%d ItemCount:%d\n", prsp.Basic.Uid, prsp.Basic.Sex, prsp.Basic.Addr,
						prsp.Basic.Level, prsp.Detail.Exp , prsp.Detail.Depot.ItemsCount)
					for instid , pitem := range prsp.Detail.Depot.Items {
						v_print("[%d] res:%d count:%d attr:%d\n" , instid , pitem.ResId , pitem.Count , pitem.Attr);
					}
				}
				return true;
			}
		case cs.CS_PROTO_LOGOUT_RSP:
			prsp, ok := gmsg.SubMsg.(*cs.CSLogoutRsp)
			if ok {
				v_print("logout result:%d msg:%s\n", prsp.Result, prsp.Msg)
                return true;
			}
		case cs.CS_PROTO_REG_RSP:
			prsp , ok := gmsg.SubMsg.(*cs.CSRegRsp)
			if ok {
				v_print("reg result:%d name:%s\n", prsp.Result, prsp.Name);
                return true;
			}
		default:
			fmt.Printf("illegal proto:%d\n", gmsg.ProtoId)
			return false;
		}


        break;
	}

	return false;
}

//send pkg to server
func SendPkg(conn *net.TCPConn, cmd string) bool {
	var gmsg cs.GeneralMsg
	var err error
	var enc_data []byte

	pkg_buff := make([]byte, BUFF_LEN)
	//parse cmd and arg
	args := strings.Split(cmd , " ");


	//encode msg
	switch args[0] {
	case CMD_PING:
		v_print("ping...\n")

		gmsg.ProtoId = cs.CS_PROTO_PING_REQ
		psub := new(cs.CSPingReq)
		psub.TimeStamp = time.Now().UnixNano() / 1000
		gmsg.SubMsg = psub

	case CMD_LOGIN:
		if len(args) != 3 {
			show_cmd();
			return false;
		}
		gmsg.ProtoId = cs.CS_PROTO_LOGIN_REQ
		psub := new(cs.CSLoginReq)
		psub.Name = args[1]
		psub.Device = "onepluse9"
		psub.Pass = args[2]
		gmsg.SubMsg = psub
		v_print("login...name:%s pass:%s\n", psub.Name, psub.Pass);
	case CMD_LOGOUT:
		v_print("logout...\n")

		gmsg.ProtoId = cs.CS_PROTO_LOGOUT_REQ
		psub := new(cs.CSLogoutReq)
		psub.Uid = 0
		gmsg.SubMsg = psub
	case CMD_REG: // register <name> <pass> <sex:1|2> <addr>
		if len(args) != 5 {
			show_cmd();
			return false;
		}
		gmsg.ProtoId = cs.CS_PROTO_REG_REQ
		psub := new(cs.CSRegReq)
		psub.Name = args[1];
		psub.Pass = args[2];
		sex_v , _ := strconv.Atoi(args[3]);
		psub.Sex = uint8(sex_v);
		psub.Addr = args[4];
		v_print("reg... name:%s pass:%s sex:%d addr:%s\n", psub.Name, psub.Pass, psub.Sex, psub.Addr);

		gmsg.SubMsg = psub;
	default:
		fmt.Printf("illegal cmd:%s\n", cmd)
		return false
	}

	//encode
	enc_data, err = cs.EncodeMsg(&gmsg)
	if err != nil {
		fmt.Printf("encode %s failed! err:%v\n", cmd, err)
		return false
	}

	//compress
	if zlib_enalbe {
		var b bytes.Buffer;
		w := zlib.NewWriter(&b);
		w.Write(enc_data);
		w.Close();
		enc_data = b.Bytes();
	}

	//pack
	pkg_len := lnet.PackPkg(pkg_buff, enc_data, 0)
	if pkg_len < 0 {
		fmt.Printf("pack cmd:%s failed!\n", cmd)
		return false
	}

	//send
	_, err = conn.Write(pkg_buff[:pkg_len])
	if err != nil {
		fmt.Printf("send cmd pkg failed! cmd:%s err:%v\n", cmd, err)
		return false
	}
	v_print("send cmd:%s success! pkg:%v pkg_len:%d \n", cmd, pkg_buff[:pkg_len], pkg_len)
	return true;
}

//single case
func TestCase(idx int , host string , port int , cmd string) {
	start_us := time.Now().UnixNano()/1000;
	server_addr := host + ":" + strconv.Itoa(port)
	//connect
	tcp_addr, err := net.ResolveTCPAddr("tcp4", server_addr)
	if err != nil {
		fmt.Printf("[%d] resolve addr:%s failed! err:%s\n", idx , server_addr, err)
		return
	}

	conn, err := net.DialTCP("tcp4", nil, tcp_addr)
	if err != nil {
		fmt.Printf("[%d] connect %s failed! err:%v\n", idx , server_addr, err)
		return
	}
	defer conn.Close()


	//send pkg
	ok := SendPkg(conn , cmd);
	if !ok {
		fmt.Printf("[%d] send pkg failed!\n" , idx);
		return
	}


    //recv pkg
    ok = RecvPkg(conn);
	end_us := time.Now().UnixNano()/1000;
    if !ok {
    	//fmt.Printf("[%d] recv pkg failed!\n" , idx);
		fmt.Printf("cmd:-1|start:%d|end:%d|cost:%d\n" , start_us , end_us , (end_us-start_us));
	}

    //success
	fmt.Printf("cmd:0|start:%d|end:%d|cost:%d\n" , start_us , end_us , (end_us-start_us));
	//keep alive
	time.Sleep(time.Duration(*keep) * time.Second);
	wg.Done()
}




func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("main panic! err:%v" , err);
		}
	}()
	flag.Parse()
	if *port <= 0 || *total_count<=0 || *concurrent_count<=0{
		flag.PrintDefaults()
		show_cmd()
		return
	}
	if *help {
		show_cmd();
		return;
	}

	wg.Add(*total_count);
	sleep_us := time.Duration(1000000 / *concurrent_count);
	fmt.Errorf("start client ... slee_us:%d" , sleep_us);

	for i:=0; i<*total_count; i++ {
		time.Sleep(sleep_us * time.Microsecond);
		go TestCase(i , *host , *port , *cmd);
	}

	//start_us := time.Now().UnixNano()/1000;
	//end_us := time.Now().UnixNano()/1000;

    wg.Wait();
	v_print("press finish!\n");
}
