package cs


/*
type User struct {
	UserId   int64  `json:"user_id"`
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	ItemList []Item `json:"item_list"`
	MiscInfo []byte `json:"misc_info"`
}
 */

type UserBasic struct {
	Uid   int64  `json:"uid"`
	Name  string `json:"name"`
	Addr  string `json:"addr"`
	Sex   uint8  `json:"sex"` //1:male 2:female
	Level int32  `json:"level"`
}


type Item struct {
	ResId int32 `json:"res_id"`
	Instid int64 `json:"instid"`
	Count int32 `json:"count"`
	Attr int64 `json:"attr"`
}

type UserDepot struct {
	ItemsCount int32 `json:"item_count"`
	Items map[int64]*Item `json:"items"`
}


type UserDetail struct {
	Exp int32 `json:"exp"`
	Depot *UserDepot `json:"user_depot"`
}
