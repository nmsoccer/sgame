package lib

import (
    "sgame/proto"
    "time"
)

const (
    MESSAGE_LEN = 200*1024 //200k
)

type Msg struct{
	sender int
	msg []byte	
}

var pmsg *Msg;

func init() {
	pmsg = new(Msg);
	pmsg.msg = make([]byte , MESSAGE_LEN);
}


func RecvMsg(pconfig *Config) int64 {
	var _func_ = "RecvMsg";
	var start_time int64;	
	
	//init
	pmsg.msg = pmsg.msg[0:cap(pmsg.msg)];
	var recv int;
	var log = pconfig.Comm.Log;
	var proc = pconfig.Comm.Proc;
	var msg = pmsg.msg;
	var handle_pkg = 0;
	
	start_time = time.Now().UnixNano();
	//keep recving
    for {
    	msg = msg[0:cap(msg)];
        recv = proc.Recv(msg, cap(msg), &(pmsg.sender));
        if recv < 0 { //no package
        	break;
        }
        
        handle_pkg++;
        //unpack
        msg = msg[0:recv];
        //log.Debug("recved:%d sender:%d v:%v" , recv , pmsg.sender , msg);
        var ss_req = new(proto.SSMsgReq);
        err := proto.UnPack(msg, ss_req);
        if err != nil {
    	    log.Err("unpack failed! err:%v" , err);
    	    continue;
        } 
    	//log.Debug("unpack success! v:%v and %v" , *ss_req , *(ss_req.GetHeartBeat()));

		//dispatch          
        switch ss_req.ProtoType {
        	case proto.SS_PROTO_TYPE_HEART_BEAT_REQ:
        	    RecvHeartBeatReq(pconfig, ss_req.GetHeartBeat(), pmsg.sender);
        	default:
        	    log.Err("%s fail! unknown proto type:%v" , _func_ , ss_req.ProtoType);
        }  
    }
    
    //return
    if handle_pkg == 0 {
    	return 0;
    } else {
    	return (time.Now().UnixNano() - start_time)/1000000; //millisec
    }
}