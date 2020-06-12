package cs

type CSLoginReq struct {
	Name string `json:"name"`
	Pass string `json:"pass"`
	Device string `json:"device"`
}

type CSLoginRsp struct {
	Result int `json:"result"`
	Basic UserBasic `json:"basic"`
}
