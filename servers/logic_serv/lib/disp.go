package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
)


func RecvDispMsg(pconfig *Config, pdisp *ss.MsgDisp) {
	var _func_ = "<RecvDispMsg>"
	log := pconfig.Comm.Log

	log.Debug("%s disp_proto:%d disp_from:%d ", _func_, pdisp.ProtoType, pdisp.FromServer)

	//extract disp
	pv, err := comm.ExDispMsg(pdisp)
	if err != nil {
		log.Err("%s extract disp msg failed! disp_proto:%d from_server:%d err:%v", _func_, pdisp.ProtoType, pdisp.FromServer, err)
		return
	}

	switch pdisp.ProtoType {
	case ss.DISP_PROTO_TYPE_HELLO:
		pmsg, ok := pv.(*ss.MsgDispHello)
		if !ok {
			break
		}
		log.Info("%s hello! content:%s", _func_, pmsg.Content)
		return
	case ss.DISP_PROTO_TYPE_KICK_DUPLICATE_USER:
		pmsg, ok := pv.(*ss.MsgDispKickDupUser)
		if !ok {
			break
		}
		RecvDupUserKick(pconfig, pmsg, int(pdisp.FromServer))
		return
	default:
		log.Err("%s unkown disp_proto:%d", _func_, pdisp.ProtoType)
	}

	log.Err("%s conver disp-msg fail! proto:%d", _func_, pdisp.ProtoType)
}