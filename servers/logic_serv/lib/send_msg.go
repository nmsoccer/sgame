package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
)

//@v only support []byte and *ss.SSMsg
func SendToServ(pconfig *Config  , target_serv int , v interface{}) bool {
	var _func_ = "<SendToServ>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;

	var buff []byte = nil
	var pss_msg *ss.SSMsg = nil
	var ok bool = false
	var err error = nil

	//check type
	switch v.(type) {
	case []byte:
		buff , ok = v.([]byte)
		if !ok {
			log.Err("%s conv to []byte failed!" , _func_)
			return false
		}
	case *ss.SSMsg:
		pss_msg , ok = v.(*ss.SSMsg)
		if !ok {
			log.Err("%s conv to *ss.SSMsg failed!" , _func_)
			return false
		}
		//pack
		buff , err = ss.Pack(pss_msg)
		if err != nil {
			log.Err("%s pack failed! proto:%d err:%v" , _func_ , pss_msg.ProtoType , err)
			return false
		}

	default:
		log.Err("%s fail! illegal v type" , _func_)
		return false
	}

	//send
	ret := proc.Send(target_serv, buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_serv , ret);
		return false;
	}
	if pss_msg!=nil && pss_msg.ProtoType != ss.SS_PROTO_TYPE_HEART_BEAT_REQ {
		log.Debug("%s send to %d success!", _func_, target_serv);
	}
	return true
}



//send to connect server
func SendToConnect(pconfig *Config, v interface{}) bool {
	var _func_ = "<SendToConnect>"
	log := pconfig.Comm.Log

	//select connect server
	target_id := pconfig.FileConfig.ConnServ
	if target_id <= 0 {
		log.Err("%s fail! target_id:%d illegal!" , _func_ , target_id)
	}

	//send
	return SendToServ(pconfig , target_id , v)
}


//send to db server
func SendToDb(pconfig *Config, v interface{}) bool {
	var _func_ = "<SendToDb>"
	log := pconfig.Comm.Log

	//send to db server
	target_id := pconfig.FileConfig.DbServ
	if target_id <= 0 {
		log.Err("%s fail! target_id:%d illegal!" , _func_ , target_id)
	}

	//send
	return SendToServ(pconfig , target_id , v)
}


//send to disp hash
//if hash_v>0 use hash first; or use rand method
func SendToDisp(pconfig *Config, hash_v int64, buff []byte) bool {
	var _func_ = "<SendToDispHash>"
	log := pconfig.Comm.Log

	//SELECT DISP
	//BY HASH
	disp_serv := -1
	if hash_v > 0 {
		disp_serv = comm.SelectProperServ(pconfig.Comm, comm.SELECT_METHOD_HASH, hash_v, pconfig.FileConfig.DispServList,
			pconfig.Comm.PeerStats, comm.PERIOD_HEART_BEAT_DEFAULT/1000)
		if disp_serv <= 0 {
			log.Err("%s fail! no proper disp by hash found! key:%d candidate:%v", _func_, hash_v, pconfig.FileConfig.DispServList)
		}
	}

	//BY RAND
	if disp_serv <= 0 {
		disp_serv = comm.SelectProperServ(pconfig.Comm, comm.SELECT_METHOD_RAND, 0, pconfig.FileConfig.DispServList,
			pconfig.Comm.PeerStats, comm.PERIOD_HEART_BEAT_DEFAULT/1000)
		if disp_serv <= 0 {
			log.Err("%s fail! no proper disp by rand found! key:%d candidate:%v", _func_, hash_v, pconfig.FileConfig.DispServList)
			return false
		}
	}


	//Send
	return SendToServ(pconfig , disp_serv , buff)
}
