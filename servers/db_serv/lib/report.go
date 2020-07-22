package lib

import "sgame/servers/comm"


func RecvReportCmd(arg interface{}) {
	var _func_ = "<RecvReportCmd>"
	pconfig , ok := arg.(*Config);
	if !ok {
		return;
	}
    log := pconfig.Comm.Log;

    //handle cmd
    var pmsg *comm.ReportMsg;
    for i:=0; i<comm.MAX_RECV_PER_TICK; i++ {
        pmsg = pconfig.ReportServ.Recv();
        if pmsg == nil {
        	break;
		}

		//Handle
		log.Info("%s proto:%d int_v:%d str_v:%s" , _func_ , pmsg.ProtoId , pmsg.IntValue , pmsg.StrValue);
        HandleReportCmd(pconfig , pmsg);
	}


}

func HandleReportCmd(pconfig *Config , pmsg *comm.ReportMsg) {
	var _func_ = "HandleReportCmd";
	log := pconfig.Comm.Log;

	switch pmsg.ProtoId {
	case comm.REPORT_PROTO_CMD_RELOAD:
		if pmsg.StrValue == "reload_cfg" {
			var file_config FileConfig;
			ret := comm.LoadJsonFile(pconfig.ConfigFile , &file_config , pconfig.Comm);
			if ret {
				log.Info("%s reload_cfg success! proto:%d" , _func_ , pmsg.ProtoId);
				*pconfig.FileConfig = file_config; //override
				pconfig.ReportServ.Report(comm.REPORT_PROTO_RELOAD_RSP , pmsg.IntValue , comm.RELOAD_STAT_SUCCESS , nil);
			} else {
				pconfig.ReportServ.Report(comm.REPORT_PROTO_RELOAD_RSP , pmsg.IntValue , comm.RELOAD_STAT_FAIL , nil);
			}
		} else { //just back
			pconfig.ReportServ.Report(comm.REPORT_PROTO_RELOAD_RSP , pmsg.IntValue , comm.RELOAD_STAT_NOP , nil);
		}

	default:
		log.Err("%s illegal proto:%d" , _func_ , pmsg.ProcId);
	}

	return;
}


