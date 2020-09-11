package lib

import "sgame/proto/ss"

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



//send to logic server
func SendToLogic(pconfig *Config , pss_msg *ss.SSMsg) bool {
	var _func_ = "<SendToLogic>";
	log := pconfig.Comm.Log;

	//logic
	if pconfig.FileConfig.LogicServ <= 0 {
		log.Err("%s fail! logic serv:%d illegal!" , _func_ , pconfig.FileConfig.LogicServ)
		return false
	}

	return SendToServ(pconfig , pconfig.FileConfig.LogicServ , pss_msg)
}