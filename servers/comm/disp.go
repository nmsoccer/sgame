package comm

import (
	"errors"
	"fmt"
	"sgame/proto/ss"
)


/*
Generate Disp Msg by arg. this function should be modified when new ss.DISP_PROTO_TYPE added
@disp_msg: Disp sub msg  like ss.MsgDispxxxx
@target:target server type
@method:choose spec target server method
@spec:spec target server will ignore @target and @method
@return:ss_msg , error
 */
func GenDispMsg(target ss.DISP_MSG_TARGET, method ss.DISP_MSG_METHOD, proto ss.DISP_PROTO_TYPE, spec int , sender int , disp_msg interface{}) (*ss.SSMsg , error) {
	var ss_msg = new(ss.SSMsg)
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_USE_DISP_PROTO
	body := new(ss.SSMsg_MsgDisp)
	body.MsgDisp = new(ss.MsgDisp)
	body.MsgDisp.ProtoType = proto
	body.MsgDisp.Method = method
	body.MsgDisp.Target = target
	body.MsgDisp.SpecServer = int32(spec)
	body.MsgDisp.FromServer = int32(sender)
	ss_msg.MsgBody = body

	//create dis_body
	//switch proto
	switch proto {
	case ss.DISP_PROTO_TYPE_HELLO:
		disp_body := new(ss.MsgDisp_Hello)
		pv , ok := disp_msg.(*ss.MsgDispHello)
		if !ok {
			return nil , errors.New("not MsgDispHello")
		}
		disp_body.Hello = pv
		body.MsgDisp.DispBody = disp_body
	case ss.DISP_PROTO_TYPE_KICK_DUPLICATE_USER:
		disp_body := new(ss.MsgDisp_KickDupUser)
		pv , ok := disp_msg.(*ss.MsgDispKickDupUser)
		if !ok {
			return nil , errors.New("not MsgDispKickDupUser")
		}
		disp_body.KickDupUser = pv
		body.MsgDisp.DispBody = disp_body
	default:
		return nil , errors.New(fmt.Sprintf("disp proto:%d not handled" , proto))
	}

	return ss_msg , nil
}

/*
Extract MsgDispxx sub msg from MsgDisp. this function should be modified when new ss.DISP_PROTO_TYPE added
@return:success: MsgDispxxxx and  nil if failed
 */
func ExDispMsg(pdisp *ss.MsgDisp) (interface{} , error) {
	switch pdisp.ProtoType {
	case ss.DISP_PROTO_TYPE_HELLO:
		disp_body , ok := pdisp.DispBody.(*ss.MsgDisp_Hello)
		if !ok {
			return nil , errors.New("not MsgDisp_Hello")
		}
		return disp_body.Hello , nil
	case ss.DISP_PROTO_TYPE_KICK_DUPLICATE_USER:
		disp_body , ok := pdisp.DispBody.(*ss.MsgDisp_KickDupUser)
		if !ok {
			return nil , errors.New("not MsgDisp_KickDupUser")
		}
		return disp_body.KickDupUser , nil
	default:
		break
	}

	return nil , errors.New(fmt.Sprintf("disp proto:%d not handled" , pdisp.ProtoType))
}