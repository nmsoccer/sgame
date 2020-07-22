package lib

import (
	"bytes"
	"compress/flate"
	"io"
	"sgame/proto/cs"
	"sgame/proto/ss"
	"sgame/servers/comm"
	"time"
	"compress/zlib"
)

type ZlibEnv struct {
	//for compress
	c_bf bytes.Buffer
	c_w *zlib.Writer
	//for uncompress
	u_reader *bytes.Reader
	u_bf bytes.Buffer
	u_zr io.ReadCloser
}

var m_zenv ZlibEnv;
var pzenv = &m_zenv;

func init() {
  pzenv.c_w = zlib.NewWriter(&pzenv.c_bf); //using default compress level
  pzenv.u_reader = bytes.NewReader(nil);
  pzenv.u_zr , _ = zlib.NewReader(pzenv.u_reader);
}


func ReadClients(pconfig *Config) int64 {
	var _func_ = "<ReadClients>"
	log := pconfig.Comm.Log

	//get results
	var results []*comm.ClientPkg = pconfig.TcpServ.Recv(pconfig.Comm) //comm.*ClientPkg

	//log.Debug("%s get results:%v len:%d" , _func_ , results , len(results));
	if results == nil || len(results) == 0 {
		return 0
	}

	start_ts := time.Now().UnixNano()
	//print
	for i := 0; i < len(results); i++ {
		//log.Debug("%s key:%v , type:%d , read:%v", _func_, results[i].ClientKey, results[i].PkgType, results[i].Data)
		HandleClientPkg(pconfig, results[i])
	}

	//diff
	diff := time.Now().UnixNano() - start_ts
	log.Debug("%s cost %dus pkg:%d", _func_, diff/1000, len(results))
	return diff
}

func HandleClientPkg(pconfig *Config, pclient *comm.ClientPkg) {
	var _func_ = "<HandleClientPkg>"
	var gmsg cs.GeneralMsg
	log := pconfig.Comm.Log

	//connection closed pkg
	if pclient.PkgType == comm.CLIENT_PKG_T_CONN_CLOSED {
		log.Info("%s connection closed! key:%v", _func_, pclient.ClientKey)
		//clear map
		uid, ok := pconfig.Ckey2Uid[pclient.ClientKey]
		if ok {
			log.Info("%s is already login. uid:%d notify logout to logic!", _func_, uid)
			SendLogoutReq(pconfig, uid, ss.USER_LOGOUT_REASON_LOGOUT_CONN_CLOSED)
			delete(pconfig.Ckey2Uid, pclient.ClientKey)
			//xxxx
			delete(pconfig.Uid2Ckey, uid)
		} else {
			log.Info("%s no need to upper post!", _func_)
		}
		return
	}

	//normal pkg
	//zib uncompress
	client_data := pclient.Data
	//before_len := len(client_data);
	if pconfig.FileConfig.ZlibOn == 1 {
		pzenv.u_reader.Reset(pclient.Data);
		pzenv.u_bf.Reset();
		err := pzenv.u_zr.(flate.Resetter).Reset(pzenv.u_reader , nil);
		//r , err := zlib.NewReader(pzenv.u_reader);
		if err != nil {
			log.Err("%s reset failed when uncompressing! err:%v c_key:%d" , _func_ , err , pclient.ClientKey);
			return;
		}
		io.Copy(&pzenv.u_bf , pzenv.u_zr);
		client_data = pzenv.u_bf.Bytes();
		/*
        b := bytes.NewReader(pclient.Data);
        var out bytes.Buffer;
        r , _ := zlib.NewReader(b);
        io.Copy(&out , r);
        client_data = out.Bytes();*/
	}
	//after_len := len(client_data);
    //log.Debug("%s uncompress %d->%d" , _func_ , before_len , after_len);

	//decode msg
	err := cs.DecodeMsg(client_data, &gmsg)
	if err != nil {
		log.Err("%s decode msg failed! err:%v", _func_, err)
		return
	}

	proto_id := gmsg.ProtoId
	var conv_err = true
	//convert
	switch proto_id {
	case cs.CS_PROTO_PING_REQ:
		pmsg, ok := gmsg.SubMsg.(*cs.CSPingReq)
		if ok {
			log.Debug("%s recv proto:%d success! v:%v", _func_, proto_id, *pmsg)
			SendPingReq(pconfig, pclient.ClientKey, pmsg)
			conv_err = false
		}
	case cs.CS_PROTO_LOGIN_REQ:
		pmsg, ok := gmsg.SubMsg.(*cs.CSLoginReq)
		if ok {
			log.Debug("%s recv proto:%d success! v:%v", _func_, proto_id, *pmsg)
			SendLoginReq(pconfig, pclient.ClientKey, pmsg)
			conv_err = false
		}
	case cs.CS_PROTO_LOGOUT_REQ:
		uid, exist := pconfig.Ckey2Uid[pclient.ClientKey]
		if !exist {
			log.Err("%s proto:%d but not login! key:%v", _func_, proto_id, pclient.ClientKey)
			return
		}
		_, ok := gmsg.SubMsg.(*cs.CSLogoutReq)
		if ok {
			log.Debug("%s recv proto:%d success! uid:%v", _func_, proto_id, uid)
			SendLogoutReq(pconfig, uid, ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_EXIT)
			conv_err = false
		}
	case cs.CS_PROTO_REG_REQ:
		pmsg , ok := gmsg.SubMsg.(*cs.CSRegReq)
		if ok {
			log.Debug("%s recv proto:%d success! v:%v", _func_, proto_id, *pmsg)
			SendRegReq(pconfig, pclient.ClientKey, pmsg)
			conv_err = false
		}
	default:
		log.Err("%s illegal proto:%d", _func_, proto_id)
		return
	}

	//log
	if conv_err {
		log.Err("%s conv proto:%d failed!", _func_, proto_id)
	}
}

func SendToClient(pconfig *Config, client_key int64, gmsg *cs.GeneralMsg) bool {
	var _func_ = "<SendToClient>"
	log := pconfig.Comm.Log

	//enc msg
	enc_data, err := cs.EncodeMsg(gmsg)
	if err != nil {
		log.Err("%s encode msg failed! key:%v err:%v", _func_, client_key, err)
		return false
	}

	//befor_len := len(enc_data);
	//zlib
    if pconfig.FileConfig.ZlibOn == 1 {
    	pzenv.c_bf.Reset();
    	pzenv.c_w.Reset(&pzenv.c_bf);
    	pzenv.c_w.Write(enc_data);
    	pzenv.c_w.Flush();
    	enc_data = pzenv.c_bf.Bytes();
    	/*
    	var b bytes.Buffer;
    	w := zlib.NewWriter(&b);
    	w.Write(enc_data);
    	w.Close();
    	enc_data =  b.Bytes();*/
	}
	//after_len := len(enc_data);
    //log.Debug("%s compressed %d -> %d" , _func_ , befor_len , after_len);

	//make pkg
	pclient := new(comm.ClientPkg)
	pclient.PkgType = comm.CLIENT_PKG_T_NORMAL;
	pclient.ClientKey = client_key
	pclient.Data = enc_data

	//Send
	ret := pconfig.TcpServ.Send(pconfig.Comm, pclient)
	if ret < 0 {
		log.Err("%s send to client ret:%d len:%d data:%v ", _func_, ret, len(enc_data), string(enc_data))
		return false;
	}
	return true
}

//server close connection positively
func CloseClient(pconfig *Config , client_key int64) bool {
	var _func_ = "<CloseClient>"
	log := pconfig.Comm.Log

    //makg pkg
    pclient := new(comm.ClientPkg)
    pclient.PkgType = comm.CLIENT_PKG_T_CLOSE_CONN;
    pclient.ClientKey = client_key;

    //Send
	ret := pconfig.TcpServ.Send(pconfig.Comm, pclient)
	log.Debug("%s send to client ret:%d", _func_, ret)
	return true
}