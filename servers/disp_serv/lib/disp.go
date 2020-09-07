package lib

import "sgame/proto/ss"

func RecvDispMsg(pconfig *Config , pdisp *ss.MsgDisp , from int , msg []byte) {
	var _func_ = "<RecvDispMsg>"
	log := pconfig.Comm.Log

	//Dispatch to Target
	log.Debug("%s disp_proto:%d from:%d disp_from:%d target:%d spec:%d method:%d" , _func_ , pdisp.ProtoType , from , pdisp.FromServer , pdisp.Target ,
		pdisp.SpecServer , pdisp.Method)

	//to spec server
	if pdisp.SpecServer > 0 {
		SendToServ(pconfig , int(pdisp.SpecServer) , msg)
	}

}
