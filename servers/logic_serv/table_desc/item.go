package table_desc

//item table
type ItemConfig struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Price int `json:"price"`
	ItemType int `json:"type"`
	Pos []int `json:"pos"`
	CanSell int `json:"can_sell"`
}

type ItemTable struct {
    Count int `json:"count"`
	Res []ItemConfig `json:"res"`
}

//item timing table
type ItemTimingCfg struct {
	Id int `json:"id"`
	ExpireTime string `json:"expire_time"`
	Show int `json:"show"`
}

type ItemTimingTable struct {
	Count int `json:"count"`
	Res []ItemTimingCfg `json:"res"`
}


//Conf Json File
type ItemJson struct {
	ItemTimingTable ItemTimingTable `json:"item_timing_table"` //timing sheet
	ItemTable ItemTable `json:"item_table"` //item sheet
}