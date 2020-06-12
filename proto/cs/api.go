package cs

import (
  "encoding/json"
  "errors"
)

/*
This is a cs-proto using json format
*/

/*
CS PROTO ID
*/
const (
  //proto start	
  CS_PROTO_START=0	
  CS_PROTO_PING_REQ=1
  CS_PROTO_PING_RSP=2
  CS_PROTO_LOGIN_REQ=3
  CS_PROTO_LOGIN_RSP=4
  //PS:new proto added should modify 'proto2msg' function
  //proto end
  CS_PROTO_END=5
  
  
    	
)

/*
* GeneralMsg 
*/
type GeneralMsg struct {
	ProtoId int `json:"proto"`
	SubMsg interface{} `json:"sub"`
}

type ProtoHead struct {
	ProtoId int `json:"proto"`
	Sub interface{} `json:"-"`
}

/*
* Encode GeneralMsg
* @return encoded_bytes , error
*/
func EncodeMsg(pmsg *GeneralMsg) ([]byte , error) {	
	//proto
	if pmsg.ProtoId<=CS_PROTO_START || pmsg.ProtoId>=CS_PROTO_END {
		return nil , errors.New("proto_id illegal");
	}
		
	//encode
	return json.Marshal(pmsg);	
}

/*
* Decode GeneralMsg
* @return 
*/
func DecodeMsg(data []byte , pmsg *GeneralMsg) error {
	var proto_head ProtoHead;
	var err error;
	
	//decode proto
	err = json.Unmarshal(data, &proto_head);
	if err != nil {
		return err;
	}
	
	//switch proto
	proto_id := proto_head.ProtoId;
	psub , err := proto2msg(proto_id); 
	if err != nil {
		return err;
	}	
	pmsg.SubMsg = psub;
		
	//decode
	err = json.Unmarshal(data, pmsg);
	if err != nil {
		return err;
	}
	
	return nil;
}

/*-----------------------------------STATIC--------------------*/
/*
* get real msg pointer by proto
*/
func proto2msg(proto_id int) (interface{} , error) {
	var pmsg interface{};
	
	//refer
    switch proto_id {
		case CS_PROTO_PING_REQ:
		    pmsg = new(CSPingReq);
		case CS_PROTO_PING_RSP:
		    pmsg = new(CSPingRsp);
		case CS_PROTO_LOGIN_REQ:
		    pmsg = new(CSLoginReq);
		case CS_PROTO_LOGIN_RSP:
		    pmsg = new(CSLoginRsp);       
		default:
		    return nil , errors.New("proto illegal!");
	}
    
    
    //return
    return pmsg , nil;	
}
