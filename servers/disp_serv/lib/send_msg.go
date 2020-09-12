package lib

import "sgame/proto/ss"

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