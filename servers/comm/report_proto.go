package comm

import (
	"encoding/json"
)

//report proto
const (
	MAX_REPORT_MSG_LEN = 10*1024
	//sys defined proto here
	REPORT_PROTO_MONITOR = 1 //monitor information from top cmd
	REPORT_PROTO_RESERVED = 10000 //reserved proto id
	//user defined proto starts here
    REPORT_PROTO_SERVER_START = 10001 //server start time
    REPORT_PROTO_SERVER_HEART = 10002 //server heartbeat
    REPORT_PROTO_CONN_NUM = 10003 //server connection number.
    REPORT_PROTO_SYNC_SERVER = 10004 //sync basic server information
    REPORT_PROTO_SERVER_STOP = 10005 //server stop time
    REPORT_PROTO_CMD_REQ = 10006 //cmd to reload cfg
    REPORT_PROTO_CMD_RSP = 10007 //reload rsp

    //Cmd Stat
	CMD_STAT_NONE = ""
	CMD_STAT_ING = "ing"
	CMD_STAT_SUCCESS = "done"
	CMD_STAT_FAIL = "fail"
	CMD_STAT_NOP = "nop" //no operation

	//Report Cmd
	CMD_CMD_NONE = ""
	CMD_RELOAD_CFG = "reload_cfg"
    CMD_RELOAD_TAB = "reload_table"
    CMD_STOP_SERVER = "stop_server"
    CMD_LOG_LEVEL = "log_level" //chg log filt level
    CMD_LOG_DEGREE = "log_degree" //chg log time degree
    CMD_LOG_ROTATE = "log_rotate" //chg log rotate
    CMD_LOG_SIZE = "log_size" //chg log size
	CMD_START_GPROF = "start_gprof"
	CMD_END_GRPOF = "end_gprof"
)

//report msg
type ReportMsg struct {
	ProtoId int `json:"proto"`
	ProcId int `json:"proc_id"`
	IntValue int64 `json:"intv"` //for normal int value
	StrValue string `json:"strv"` //for normal string value
	Sub interface{} `json:sub` //for complex info
}

type ReportHead struct {
	ProtoId int `json:"proto"`
	ProcId int `json:"proc_id"`
	IntValue int64 `json:"intv"` //for normal int value
	StrValue string `json:"strv"` //for normal string value
	Sub interface{} `json:"-"` //for complex info
}


func EncodeReportMsg(pmsg *ReportMsg) ([]byte , error) {
	return json.Marshal(pmsg);
}

func DecodeReportMsg(data []byte , pmsg *ReportMsg) error{
	var proto_head ReportHead;
	var err error;
	
	//decode proto
	err = json.Unmarshal(data, &proto_head);
	if err != nil {
		return err;
	}
	
	//switch proto	
	psub , err := proto2msg(proto_head.ProtoId); 
	if err != nil {
		return err;
	}	
	pmsg.Sub = psub;
	
	//direct return
	if psub == nil {
		pmsg.ProtoId = proto_head.ProtoId;
		pmsg.ProcId = proto_head.ProcId;
		pmsg.StrValue = proto_head.StrValue;
		pmsg.IntValue = proto_head.IntValue;
		return nil;
	}
	
		
	//decode
	err = json.Unmarshal(data, pmsg);
	if err != nil {
		return err;
	}
	
	return nil;
}

/*-----------------------------------SubMsg--------------------*/
type SyncServerMsg struct{
	StartTime int64	
}

type CmdExtraMsg struct {
	ExtraValue string	`json:"ext_value"`
}



/*-----------------------------------STATIC--------------------*/
/*
* get real msg pointer by proto
*/
func proto2msg(proto_id int) (interface{} , error) {
	var pmsg interface{};
	
	//refer
    switch proto_id {
    case  REPORT_PROTO_SYNC_SERVER:
        pmsg = new(SyncServerMsg)
	case REPORT_PROTO_CMD_REQ:
		pmsg = new(CmdExtraMsg)
	default:
	    return nil , nil;
	}
        
    //return
    return pmsg , nil;	
}