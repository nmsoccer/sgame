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

	if pmsg.ProtoId != comm.REPORT_PROTO_CMD_REQ {
		log.Err("%s illegal proto:%d" , _func_ , pmsg.ProcId);
		return;
	}


	switch pmsg.StrValue {
	case comm.CMD_RELOAD_CFG:
		var file_config FileConfig;
		ret := comm.LoadJsonFile(pconfig.ConfigFile , &file_config , pconfig.Comm);
		if ret {
			log.Info("%s reload_cfg success! proto:%d" , _func_ , pmsg.ProtoId);
			AfterReLoadConfig(pconfig , pconfig.FileConfig , &file_config);
			*pconfig.FileConfig = file_config; //override
			pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP , pmsg.IntValue , comm.CMD_STAT_SUCCESS , nil);
		} else {
			pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP , pmsg.IntValue , comm.CMD_STAT_FAIL , nil);
		}

	case comm.CMD_STOP_SERVER:
		pconfig.Comm.ChInfo <- comm.INFO_EXIT;
		log.Info("%s will stop server! proto:%d" , _func_ , pmsg.ProtoId);
		pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP , pmsg.IntValue , comm.CMD_STAT_SUCCESS , nil);
	default:
		pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP , pmsg.IntValue , comm.CMD_STAT_NOP , nil);
	}

	return;
}


