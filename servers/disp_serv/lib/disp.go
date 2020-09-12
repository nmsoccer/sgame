package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
)

func RecvDispMsg(pconfig *Config , pdisp *ss.MsgDisp , from int , msg []byte) {
	var _func_ = "<RecvDispMsg>"
	log := pconfig.Comm.Log

	//Dispatch to Target
	log.Debug("%s disp_proto:%d from:%d disp_from:%d target:%d spec:%d method:%d" , _func_ , pdisp.ProtoType , from , pdisp.FromServer , pdisp.Target ,
		pdisp.SpecServer , pdisp.Method)

	//to spec server
	if pdisp.Method == ss.DISP_MSG_METHOD_SPEC {
		if pdisp.SpecServer <= 0 {
			log.Err("%s fail! spec method but spec server not set!" ,_func_)
			return
		}
		SendToServ(pconfig , int(pdisp.SpecServer) , msg)
		return
	}

	//Dispatch target
	switch pdisp.Target {
	case ss.DISP_MSG_TARGET_LOGIC_SERVER:
		DispToLogicServ(pconfig , pdisp , msg)
	case ss.DISP_MSG_TARGET_CHAT_SERVER:
		//sgame has no chat-server just for instruction
	default:
		log.Err("%s target:%d can not handle!" , _func_ , pdisp.Target)
	}

}

func DispToLogicServ(pconfig *Config , pdisp *ss.MsgDisp , msg []byte) {
	var _func_ = "<DispToLogicServ>"
	var target_serv int
	log := pconfig.Comm.Log

	//method
	switch pdisp.Method {
	case ss.DISP_MSG_METHOD_RAND:
		target_serv = comm.SelectProperServ(pconfig.Comm , comm.SELECT_METHOD_RAND , 0 , pconfig.FileConfig.LogicServList , pconfig.Comm.PeerStats ,
			comm.PERIOD_HEART_BEAT_DEFAULT/1000)
	case ss.DISP_MSG_METHOD_HASH:
		target_serv = comm.SelectProperServ(pconfig.Comm , comm.SELECT_METHOD_HASH , pdisp.HashV , pconfig.FileConfig.LogicServList , pconfig.Comm.PeerStats ,
			comm.PERIOD_HEART_BEAT_DEFAULT/1000)
	default:
		log.Err("%s method:%d illegal! proto:%d" , _func_ , pdisp.Method , pdisp.ProtoType)
		return
	}

	//check
	if target_serv <= 0 {
		log.Err("%s fail! no proper target found! method:%d proto:%d hash:%d" , _func_ , pdisp.Method , pdisp.ProtoType ,
			pdisp.HashV)
	}


	//send to target
	log.Debug("%s send to:%d method:%d hash:%d" , _func_ , target_serv , pdisp.Method , pdisp.HashV)
	SendToServ(pconfig , target_serv , msg)
}
