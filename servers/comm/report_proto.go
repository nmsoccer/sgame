package comm

import (
	"encoding/json"
)

//report proto
const (
	MAX_REPORT_MSG_LEN = 10*1024
    REPORT_PROTO_SERVER_START = 1 //server start time
    REPORT_PROTO_SERVER_HEART = 2 //server heartbeat
    REPORT_PROTO_RELOAD_CFG = 3 //server reload cfg time
    REPORT_PROTO_RELOAD_TABLE = 4 //server reload table time
    REPORT_PROTO_CONN_NUM = 5 //server connection number.
    REPORT_PROTO_SYNC_SERVER = 6 //sync basic server information
    REPORT_PROTO_SERVER_STOP = 7 //server stop time
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


/*-----------------------------------STATIC--------------------*/
/*
* get real msg pointer by proto
*/
func proto2msg(proto_id int) (interface{} , error) {
	var pmsg interface{};
	
	//refer
    switch proto_id {
    	case  REPORT_PROTO_SYNC_SERVER:
    	    pmsg = new(SyncServerMsg);
		default:
		    return nil , nil;
	}
        
    //return
    return pmsg , nil;	
}