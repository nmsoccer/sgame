package lib

import "sgame/proto/ss"

func RecvTransLogicReq(pconfig *Config , preq *ss.MsgTransLogicReq , msg []byte , from int) {
	var _func_ = "<RecvTransLogicReq>";
	log := pconfig.Comm.Log;

	if preq == nil {
		log.Err("%s fail! req nil!" , _func_);
		return;
	}

	//dispatch
	switch preq.Cmd {
	case ss.TRANS_LOGIC_CMD_TRANS_CMD_KICK_OLD_ONLINE:
		RecvTransLogicKick(pconfig , preq);
	default:
		log.Err("%s illegal cmd:%d uid:%d" , _func_ , preq.Cmd , preq.Uid);
	}

}
