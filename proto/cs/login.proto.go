package cs

type CSLoginReq struct {
	Name   string `json:"name"`
	Pass   string `json:"pass"`
	Device string `json:"device"`
}

type CSLoginRsp struct {
	Result int        `json:"result"`
	Name   string     `json:"name"`
	Basic  UserBasic  `json:"basic"`
	Detail UserDetail `json:"user_detail"`
}

type CSLogoutReq struct {
	Uid int64 `json:"uid"`
}

type CSLogoutRsp struct {
	Result int    `json:"result"`
	Uid    int64  `json:"uid"`
	Msg    string `json:"msg"`
}

type CSRegReq struct {
	Name string `json:"name"`
	Pass string `json:"pass"`
	Sex  uint8  `json:"sex"`
	Addr string `json:"addr"`
}

type CSRegRsp struct {
	Result int    `json:result`
	Name   string `json:"name"`
}
