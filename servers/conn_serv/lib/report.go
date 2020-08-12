package lib

import (
	lib_log "sgame/lib/log"
	"sgame/servers/comm"
	"strconv"
)

func RecvReportCmd(arg interface{}) {
	var _func_ = "<RecvReportCmd>"
	pconfig, ok := arg.(*Config)
	if !ok {
		return
	}
	log := pconfig.Comm.Log

	//handle cmd
	var pmsg *comm.ReportMsg
	for i := 0; i < comm.MAX_RECV_PER_TICK; i++ {
		pmsg = pconfig.ReportServ.Recv()
		if pmsg == nil {
			break
		}

		//Handle
		log.Info("%s proto:%d int_v:%d str_v:%s", _func_, pmsg.ProtoId, pmsg.IntValue, pmsg.StrValue)
		HandleReportCmd(pconfig, pmsg)
	}

}

func HandleReportCmd(pconfig *Config, pmsg *comm.ReportMsg) {
	var _func_ = "HandleReportCmd"
	log := pconfig.Comm.Log

	if pmsg.ProtoId != comm.REPORT_PROTO_CMD_REQ {
		log.Err("%s illegal proto:%d", _func_, pmsg.ProcId)
		return
	}

	//switch cmd
	switch pmsg.StrValue {
	case comm.CMD_RELOAD_CFG:
		pconfig.ReportCmd = pmsg.StrValue
		pconfig.ReportCmdToken = pmsg.IntValue
		pconfig.Comm.ChInfo <- comm.INFO_RELOAD_CFG

	case comm.CMD_START_GPROF, comm.CMD_END_GRPOF:
		pconfig.ReportCmd = pmsg.StrValue
		pconfig.ReportCmdToken = pmsg.IntValue
		pconfig.Comm.ChInfo <- comm.INFO_PPROF

	case comm.CMD_STOP_SERVER:
		pconfig.Comm.ChInfo <- comm.INFO_EXIT
		log.Info("%s will stop server! proto:%d", _func_, pmsg.ProtoId)
		pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP, pmsg.IntValue, comm.CMD_STAT_SUCCESS, nil)
	case comm.CMD_LOG_LEVEL, comm.CMD_LOG_DEGREE, comm.CMD_LOG_ROTATE, comm.CMD_LOG_SIZE:
		var result = -1
		for {
			//get extra
			pextra, ok := pmsg.Sub.(*comm.CmdExtraMsg)
			if !ok {
				log.Err("%s %s no extra info found!", _func_, pmsg.StrValue)
				break
			}
			log.Info("%s <%s> extra:%s", _func_, pmsg.StrValue, pextra.ExtraValue)

			//get value
			value, err := strconv.Atoi(pextra.ExtraValue)
			if err != nil {
				log.Err("%s %s extra:%s illegal!", _func_, pmsg.StrValue, pextra.ExtraValue)
				break
			}

			//chg attr
			var ret int = -1
			if pmsg.StrValue == comm.CMD_LOG_LEVEL {
				ret = pconfig.Comm.Log.ChgAttr(lib_log.LOG_ATTR_LEVEL, value)
			} else if pmsg.StrValue == comm.CMD_LOG_DEGREE {
				ret = pconfig.Comm.Log.ChgAttr(lib_log.LOG_ATTR_DEGREEE, value)
			} else if pmsg.StrValue == comm.CMD_LOG_ROTATE {
				ret = pconfig.Comm.Log.ChgAttr(lib_log.LOG_ATTR_ROTATE, value)
			} else { //size
				ret = pconfig.Comm.Log.ChgAttr(lib_log.LOG_ATTR_SIZE, value)
			}
			if ret != 0 {
				log.Err("%s %s chg attr to %d failed!", _func_, pmsg.StrValue, value)
				break
			}
			log.Info("%s %s extra:%d success!", _func_, pmsg.StrValue, value)
			result = 0
			break
		}

		//send back
		if result < 0 {
			pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP, pmsg.IntValue, comm.CMD_STAT_FAIL, nil)
		} else {
			pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP, pmsg.IntValue, comm.CMD_STAT_SUCCESS, nil)
		}
	default:
		pconfig.ReportServ.Report(comm.REPORT_PROTO_CMD_RSP, pmsg.IntValue, comm.CMD_STAT_NOP, nil)
	}

	return
}
