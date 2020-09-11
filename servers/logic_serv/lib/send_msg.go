package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
)

func SendToServBytes(pconfig *Config  , target_serv int , buff []byte ) bool {
	var _func_ = "<SendToServ>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;

	ret := proc.Send(target_serv, buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_serv , ret);
		return false;
	}
	log.Debug("%s send to %d success!" , _func_ , target_serv);
	return true
}

func SendToServ(pconfig *Config  , target_serv int , pss_msg *ss.SSMsg) bool {
	var _func_ = "<SendToServ>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;

	//pack
	buff , err := ss.Pack(pss_msg)
	if err != nil {
		log.Err("%s pack failed! proto:%d err:%v" , _func_ , pss_msg.ProtoType , err)
		return false
	}

	//send
	ret := proc.Send(target_serv, buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_serv , ret);
		return false;
	}
	if pss_msg.ProtoType != ss.SS_PROTO_TYPE_HEART_BEAT_REQ {
		log.Debug("%s send to %d success!", _func_, target_serv);
	}
	return true
}



//send to connect server
func SendToConnect(pconfig *Config, pss_msg *ss.SSMsg) bool {
	var _func_ = "<SendToConnect>"
	log := pconfig.Comm.Log

	//select connect server
	target_id := pconfig.FileConfig.ConnServ
	if target_id <= 0 {
		log.Err("%s fail! target_id:%d illegal!" , _func_ , target_id)
	}

	//send
	return SendToServ(pconfig , target_id , pss_msg)
}
func SendToConnectBytes(pconfig *Config, buff []byte) bool {
	var _func_ = "<SendToConnect>"
	log := pconfig.Comm.Log

	//select connect server
	target_id := pconfig.FileConfig.ConnServ
	if target_id <= 0 {
		log.Err("%s fail! target_id:%d illegal!" , _func_ , target_id)
	}

	//send
	return SendToServBytes(pconfig , target_id , buff)
}


//send to db server
func SendToDb(pconfig *Config, pss_msg *ss.SSMsg) bool {
	var _func_ = "<SendToDb>"
	log := pconfig.Comm.Log

	//send to db server
	target_id := pconfig.FileConfig.DbServ
	if target_id <= 0 {
		log.Err("%s fail! target_id:%d illegal!" , _func_ , target_id)
	}

	//send
	return SendToServ(pconfig , target_id , pss_msg)
}
func SendToDbBytes(pconfig *Config, buff []byte) bool {
	var _func_ = "<SendToDb>"
	log := pconfig.Comm.Log

	//send to db server
	target_id := pconfig.FileConfig.DbServ
	if target_id <= 0 {
		log.Err("%s fail! target_id:%d illegal!" , _func_ , target_id)
	}

	//send
	return SendToServBytes(pconfig , target_id , buff)
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
	return SendToServBytes(pconfig , disp_serv , buff)
}
