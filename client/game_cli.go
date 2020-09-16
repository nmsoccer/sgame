package main

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	lnet "sgame/lib/net"
	"sgame/proto/cs"
	"strconv"
	"strings"
	"time"
)

const (
	CMD_PING   = "ping"
	CMD_LOGIN  = "login"
	CMD_LOGOUT = "logout"
	CMD_REG    = "reg"

	BUFF_LEN = (10 * 1024)

	METHOD_INTERFACE = 1
	METHOD_COMMAND = 2
)

var cmd_map map[string]string
var zlib_enalbe = true
//var buff_len int = 1024;
var exit_ch chan bool = make(chan bool , 1)
//var enc_type = lnet.NET_ENCRYPT_DES_ECB //des encrypt
var enc_type int8 = -1
var enc_block cipher.Block
var enc_key []byte

//flag
var help = flag.Bool("h" , false , "show help")
var host *string = flag.String("a", "127.0.0.1", "server ip")
var port = flag.Int("p", 0, "server port")
var method = flag.Int("m", 0, "method 1:interace 2:command")
var cmd = flag.String("c", "", "cmd")
var keep = flag.Int("k" , 0 , "keepalive seconds if method=2")
var quiet = flag.Bool("q" , false , "quiet");

var tcp_conn  *net.TCPConn

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

func ValidConnection(conn *net.TCPConn) bool {
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
	v_print("send valid success! pkg_len:%d valid_key:%s\n", pkg_len , lnet.CONN_VALID_KEY)
	return true
}

func RsaNegotiateDesKey(conn *net.TCPConn , inn_key []byte , rsa_pub []byte) bool {
	var _func_ = "<RsaNegotiateDesKey>"
	//encrypt by rsa
	encoded , err := lnet.RsaEncrypt(inn_key , rsa_pub)
	if err != nil {
		v_print("%s failed! err:%v\n" , _func_ , err)
		return false
	}


	//pack
	pkg_buff := make([]byte , len(encoded) + 10)
	pkg_len := lnet.PackPkg(pkg_buff , encoded , lnet.PKG_OP_RSA_NEGO)
	if pkg_len <= 0 {
		fmt.Printf("valid connection pack failed! pkg_len:%d\n" , pkg_len)
		return false
	}

	//send
	_, err = conn.Write(pkg_buff[:pkg_len])
	if err != nil {
		fmt.Printf("send valid pkg failed! err:%v\n", err)
		return false
	}
	v_print("send RsaEnc success! pkg_len:%d inn_key:%s\n", pkg_len , string(inn_key))
	return true
}


func RecvConnSpecPkg(tag uint8 , data []byte) {
	var _func_ = "<RecvConnSpecPkg>"
	var err error
	//pkg option
	pkg_option := lnet.PkgOption(tag)
	switch pkg_option {
	case lnet.PKG_OP_ECHO:
		v_print("%s echo pkg! content:%s" , _func_ , string(data))
	case lnet.PKG_OP_VALID:
		enc_type = int8(data[0])
		v_print("%s valid pkg! enc_type:%d content:%s data:%v\n" , _func_ , enc_type , string(data) , data)
		if enc_type == lnet.NET_ENCRYPT_DES_ECB {
			enc_key = make([]byte , 8)
			copy(enc_key , data[1:9])
			enc_block , err = des.NewCipher(enc_key)
			if err != nil {
				v_print("%s new des block failed! err:%v" , _func_ , err)
			}
			v_print("enc_key:%s\n" , string(enc_key))
			break
		}
		if enc_type == lnet.NET_ENCRYPT_AES_CBC_128 {
			enc_key = make([]byte , 16)
			copy(enc_key , data[1:17])
			enc_block , err = aes.NewCipher(enc_key)
			if err != nil {
				v_print("%s new aes block failed! err:%v" , _func_ , err)
			}
			v_print("enc_key:%s\n" , string(enc_key))
			break
		}
		if enc_type == lnet.NET_ENCRYPT_RSA {
			rsa_pub_key := make([]byte , len(data)-1)
			copy(rsa_pub_key , data[1:])
			v_print("%s rsa_pub_key:%s\n" , _func_ , string(rsa_pub_key))

			//RSA ENC
			enc_key = []byte("12345678")
			ok := RsaNegotiateDesKey(tcp_conn , enc_key , rsa_pub_key)
			if !ok {
				enc_key = enc_key[:0] //clear
			}
		}
	case lnet.PKG_OP_RSA_NEGO:
		v_print("%s rsa_nego pkg! result:%s\n" , _func_ , string(data))
		if bytes.Compare(data[:2] , []byte("ok")) == 0 {
			enc_block , err = des.NewCipher(enc_key)
			if err != nil {
				v_print("%s new des block by rsa-nego failed! err:%v" , _func_ , err)
			}
		} else {
			v_print("%s nego des key failed for:%s" , _func_ , string(data))
		}
	default:
		v_print("%s unkonwn option:%d\n" , _func_ , pkg_option)
	}
}



func RecvPkg(conn *net.TCPConn) {
	var _func_ = "<RecvPkg>"
	read_buff := make([]byte, BUFF_LEN)
	for {
		time.Sleep(10 * time.Millisecond)

		read_buff = read_buff[:cap(read_buff)]
		//read
		n, err := conn.Read(read_buff)
		if err != nil {
			fmt.Printf("read failed! err:%v\n", err)
			os.Exit(0);
		}

		//unpack
		tag, pkg_data, pkg_len := lnet.UnPackPkg(read_buff[:n])
		if lnet.PkgOption(tag) != lnet.PKG_OP_NORMAL {
			v_print("read spec pkg. tag:%d pkg_len:%d pkg_option:%d\n", tag, pkg_len , lnet.PkgOption(tag));
			RecvConnSpecPkg(tag , pkg_data)
			continue
		}

		//decrypt
		if enc_type != lnet.NET_ENCRYPT_NONE {
			if enc_block == nil || len(enc_key)<=0 {
				v_print("%s new encrypt block nil! enc_type:%d" , _func_ , enc_type)
				return
			}
			switch enc_type {
			case lnet.NET_ENCRYPT_DES_ECB:
				pkg_data, err = lnet.DesDecrypt(enc_block, pkg_data, enc_key)
				if err != nil {
					v_print("%s des decrypt failed! err:%v", _func_, err)
					return
				}
			case lnet.NET_ENCRYPT_AES_CBC_128:
				pkg_data, err = lnet.AesDecrypt(enc_block, pkg_data, enc_key)
				if err != nil {
					v_print("%s des decrypt failed! err:%v", _func_, err)
					return
				}
			case lnet.NET_ENCRYPT_RSA:
				pkg_data, err = lnet.DesDecrypt(enc_block, pkg_data, enc_key)
				if err != nil {
					v_print("%s rsa_des decrypt failed! err:%v", _func_, err)
					return
				}
			default:
				v_print("%s illegal enc_type:%d" ,_func_ , enc_type)
				return
			}

		}

		//uncompress
		if zlib_enalbe {
			b := bytes.NewReader(pkg_data);
			var out bytes.Buffer;
			r , err := zlib.NewReader(b);
			if err != nil {
				fmt.Printf("uncompress data failed! err:%v" , err);
				if *method == METHOD_COMMAND {
					exit_ch <- false;
					return;
				}
				continue;
			}
			io.Copy(&out , r);
			pkg_data = out.Bytes();
		}


		//decode
		var gmsg cs.GeneralMsg
		err = cs.DecodeMsg(pkg_data, &gmsg)
		if err != nil {
			fmt.Printf("decode failed! err:%v\n", err)
			if *method == METHOD_COMMAND {
				exit_ch <- false;
				return;
			}
			continue
		}

		//switch rsp
		switch gmsg.ProtoId {
		case cs.CS_PROTO_PING_RSP:
			prsp, ok := gmsg.SubMsg.(*cs.CSPingRsp)
			if ok {
				curr_ts := time.Now().UnixNano() / 1000
				v_print("ping:%v ms crr_ts:%d req:%d\n", (curr_ts-prsp.TimeStamp)/1000 , curr_ts , prsp.TimeStamp)

				if *method == METHOD_COMMAND {
					exit_ch <- true;
					return;
				}
			}
		case cs.CS_PROTO_LOGIN_RSP:
			prsp, ok := gmsg.SubMsg.(*cs.CSLoginRsp)
			if ok {
				if prsp.Result == 0 {
					v_print("login result:%d name:%s role_name:%s\n", prsp.Result, prsp.Name , prsp.Basic.Name)
					v_print("uid:%v sex:%d addr:%s level:%d Exp:%d ItemCount:%d\n", prsp.Basic.Uid, prsp.Basic.Sex, prsp.Basic.Addr,
						prsp.Basic.Level, prsp.Detail.Exp , prsp.Detail.Depot.ItemsCount)
                    for instid , pitem := range prsp.Detail.Depot.Items {
                    	v_print("[%d] res:%d count:%d attr:%d\n" , instid , pitem.ResId , pitem.Count , pitem.Attr);
					}
				} else {
					v_print("login result:%d name:%s\n", prsp.Result, prsp.Name)
				}
				if *method == METHOD_COMMAND {
					exit_ch <- true;
					return;
				}
			}
		case cs.CS_PROTO_LOGOUT_RSP:
			prsp, ok := gmsg.SubMsg.(*cs.CSLogoutRsp)
			if ok {
				v_print("logout result:%d msg:%s\n", prsp.Result, prsp.Msg)

				if *method == METHOD_COMMAND {
					exit_ch <- true;
					return;
				}
			}
		case cs.CS_PROTO_REG_RSP:
			prsp , ok := gmsg.SubMsg.(*cs.CSRegRsp)
			if ok {
				v_print("reg result:%d name:%s\n", prsp.Result, prsp.Name);

				if *method == METHOD_COMMAND {
					exit_ch <- true;
					return;
				}
			}
		default:
			fmt.Printf("illegal proto:%d\n", gmsg.ProtoId)
		}

		if *method == METHOD_COMMAND {
			exit_ch <- false;
			return;
		}

	}
}

//send pkg to server
func SendPkg(conn *net.TCPConn, cmd string) {
	var _func_ = "<SendPkg>"
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
			return;
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
			return;
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
		return
	}

	//encode
	enc_data, err = cs.EncodeMsg(&gmsg)
	if err != nil {
		fmt.Printf("encode %s failed! err:%v\n", cmd, err)
		return
	}

	//compress
	if zlib_enalbe {
		var b bytes.Buffer;
		w := zlib.NewWriter(&b);
		w.Write(enc_data);
		w.Close();
		enc_data = b.Bytes();
	}

	//encrypt
	if enc_type != lnet.NET_ENCRYPT_NONE {
		if enc_block == nil || len(enc_key)<=0 {
			v_print("%s new encrypt block nil! enc_type:%d" , _func_ , enc_type)
			return
		}
		switch enc_type {
		case lnet.NET_ENCRYPT_DES_ECB:
			enc_data, err = lnet.DesEncrypt(enc_block, enc_data, enc_key)
			if err != nil {
				v_print("%s des encrypt failed! err:%v", _func_, err)
				return
			}
		case lnet.NET_ENCRYPT_AES_CBC_128:
			enc_data, err = lnet.AesEncrypt(enc_block, enc_data, enc_key)
			if err != nil {
				v_print("%s des encrypt failed! err:%v", _func_, err)
				return
			}
		case lnet.NET_ENCRYPT_RSA:
			enc_data, err = lnet.DesEncrypt(enc_block, enc_data, enc_key)
			if err != nil {
				v_print("%s rsa_des encrypt failed! err:%v", _func_, err)
				return
			}
		default:
			v_print("%s illegal enc_type:%d" ,_func_ , enc_type)
			return
		}

	}


	//pack
	pkg_len := lnet.PackPkg(pkg_buff, enc_data, 0)
	if pkg_len < 0 {
		fmt.Printf("pack cmd:%s failed!\n", cmd)
		return
	}

	//send
	_, err = conn.Write(pkg_buff[:pkg_len])
	if err != nil {
		fmt.Printf("send cmd pkg failed! cmd:%s err:%v\n", cmd, err)
	} else {
		v_print("send cmd:%s success! \n", cmd)
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("main panic! err:%v" , err);
		}
	}()
	flag.Parse()
	if *port <= 0 || (*method != METHOD_INTERFACE && *method != METHOD_COMMAND) {
		flag.PrintDefaults()
		show_cmd()
		return
	}
	if *help {
		show_cmd();
		return;
	}

	v_print("start client ...\n")
	start_us := time.Now().UnixNano()/1000;
	//server_addr := "localhost:18909";
	server_addr := *host + ":" + strconv.Itoa(*port)

	//init
	//exit_ch = make(chan bool, 1)

	//connect
	tcp_addr, err := net.ResolveTCPAddr("tcp4", server_addr)
	if err != nil {
		fmt.Printf("resolve addr:%s failed! err:%s\n", server_addr, err)
		return
	}

	conn, err := net.DialTCP("tcp4", nil, tcp_addr)
	if err != nil {
		fmt.Printf("connect %s failed! err:%v\n", server_addr, err)
		return
	}
	tcp_conn = conn
	defer conn.Close()

	//valid connection
	ok := ValidConnection(conn)
	if !ok {
		fmt.Printf("valid connection error!\n")
		return
	}


	rs := make([]byte, 128)
	//pack_buff := make([]byte , 128);
	//read
	go RecvPkg(conn)

	//check option
	switch *method {
	case METHOD_INTERFACE:
		for {
			rs = rs[:cap(rs)]
			fmt.Printf("please input:>>")
			n, _ := os.Stdin.Read(rs)
			rs = rs[:n-1] //trip last \n

			if string(rs) == "exit" {
				fmt.Println("byte...")
				exit_ch <- true
				break
			}

			//pkg_len := lnet.PackPkg(pack_buff, rs , (uint8)(*option));
			//fmt.Printf("read %d bytes and packed:%d\n", n , pkg_len);
			//n , _ = conn.Write(pack_buff[:pkg_len]);
			SendPkg(conn, string(rs))
			time.Sleep(50 * time.Millisecond)
		}

	case METHOD_COMMAND:
		if len(*cmd) <= 0 {
			flag.PrintDefaults();
			show_cmd()
			break
		}
		//start_us := time.Now().UnixNano()/1000;
		SendPkg(conn, *cmd)
		//time.Sleep(2 * time.Second)
		v := <- exit_ch;
		end_us := time.Now().UnixNano()/1000;
		if v { //success
			fmt.Printf("cmd:0|start:%d|end:%d|cost:%d\n" , start_us , end_us , (end_us-start_us));
		} else {
			fmt.Printf("cmd:-1|start:%d|end:%d|cost:%d\n" , start_us , end_us , (end_us-start_us));
		}

		//keep alive
		time.Sleep(time.Duration(*keep) * time.Second);

	default:
		fmt.Printf("option:%d nothing tod!\n", *method)
	}

	return
}
