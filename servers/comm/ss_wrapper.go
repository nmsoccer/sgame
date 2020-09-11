package comm

import (
	"errors"
	"fmt"
	"sgame/proto/ss"
)

/*
Fill SSMsg By ProtoType and MsgBody This is a helper for wrapper pkg
@pmsg: ss.Msg*** defined in SSMsg.msg_body
@pss_msg: fill info of this ss_msg
@return: error
*/
func FillSSPkg(ss_msg *ss.SSMsg , proto ss.SS_PROTO_TYPE , pmsg interface{}) error {
	ss_msg.ProtoType = proto

	switch proto {
	case ss.SS_PROTO_TYPE_HEART_BEAT_REQ:
		body := new(ss.SSMsg_HeartBeatReq)
		pv , ok := pmsg.(*ss.MsgHeartBeatReq)
		if !ok {
			return errors.New("not MsgHeartBeatReq")
		}
		body.HeartBeatReq = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_PING_REQ:
		body := new(ss.SSMsg_PingReq)
		pv , ok := pmsg.(*ss.MsgPingReq)
		if !ok {
			return errors.New("not MsgPingReq")
		}
		body.PingReq = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_PING_RSP:
		body := new(ss.SSMsg_PingRsp)
		pv , ok := pmsg.(*ss.MsgPingRsp)
		if !ok {
			return errors.New("not MsgPingRsp")
		}
		body.PingRsp = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_LOGIN_REQ:
		body := new(ss.SSMsg_LoginReq)
		pv , ok := pmsg.(*ss.MsgLoginReq)
		if !ok {
			return errors.New("not MsgLoginReq")
		}
		body.LoginReq = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_LOGIN_RSP:
		body := new(ss.SSMsg_LoginRsp)
		pv , ok := pmsg.(*ss.MsgLoginRsp)
		if !ok {
			return errors.New("not MsgLoginRsp")
		}
		body.LoginRsp = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_LOGOUT_REQ:
		body := new(ss.SSMsg_LogoutReq)
		pv , ok := pmsg.(*ss.MsgLogoutReq)
		if !ok {
			return errors.New("not MsgLogoutReq")
		}
		body.LogoutReq = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_LOGOUT_RSP:
		body := new(ss.SSMsg_LogoutRsp)
		pv , ok := pmsg.(*ss.MsgLogoutRsp)
		if !ok {
			return errors.New("not MsgLogoutRsp")
		}
		body.LogoutRsp = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_REG_REQ:
		body := new(ss.SSMsg_RegReq)
		pv , ok := pmsg.(*ss.MsgRegReq)
		if !ok {
			return errors.New("not MsgRegReq")
		}
		body.RegReq = pv
		ss_msg.MsgBody = body
	case ss.SS_PROTO_TYPE_REG_RSP:
		body := new(ss.SSMsg_RegRsp)
		pv , ok := pmsg.(*ss.MsgRegRsp)
		if !ok {
			return errors.New("not MsgRegRsp")
		}
		body.RegRsp = pv
		ss_msg.MsgBody = body
	default:
		return errors.New(fmt.Sprintf("disp proto:%d not handled" , proto))
	}

	return nil
}
