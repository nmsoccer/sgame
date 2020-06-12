package cs


type Item struct {
	ItemId int `json:"item_id"`
	Grid int  `json:"grid"`
}

type User struct {
	UserId int64  `json:"user_id"`
	Name string  `json:"name"`
	Addr string  `json:"addr"`
	ItemList []Item `json:"item_list"`
	MiscInfo []byte `json:"misc_info"`
}

type UserBasic struct {
	Uid int64 `json:"uid"`
	Addr string `json:"addr"`
	Sex uint8 `json:"sex"`
}