package cs


type CSPingReq struct {
    TimeStamp int64 `json:"ts"`
}

type CSPingRsp struct {
	TimeStamp int64 `json:"ts"`
	
}