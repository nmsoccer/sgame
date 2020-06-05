package table_desc

type GameConfig struct {
	Name string `json:"name"`
	Value string `json:"value"`
}

type GameConfigTable struct {
    Count int `json:"count"`
	Res []GameConfig `json:"res"`
}

type GameConfigJson struct {
	ConfigTable GameConfigTable `json:"game_config_table"` //each sheet
}