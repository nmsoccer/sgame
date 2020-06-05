package lib

import(
    "sgame/servers/comm"
    tables "sgame/servers/logic_serv/table_desc"
)


//TABLE_FILE_NAME
const (
    TAB_FILE_ITEM="item.json"
    TAB_FILE_GAME_CONFIG="game_config.json"  
)


func RegistTableMap(pconfig *Config) bool {
	var _func_ = "<RegistTableMap>";
	var log = pconfig.Comm.Log;
	
	pconfig.TableMap = make(comm.TableMap);		
	tab_map := pconfig.TableMap;	
	
	/*table/xx.json <-> table_desc/xx.go*/	
	//item.json
	tab_map[TAB_FILE_ITEM] = new(tables.ItemJson);
	if _ , ok := tab_map[TAB_FILE_ITEM]; !ok {
		log.Err("%s failed! new %s failed!" , _func_ , TAB_FILE_ITEM);
		return false;
	}
	
	//game_config.json
	tab_map[TAB_FILE_GAME_CONFIG] = new(tables.GameConfigJson);
	if _ , ok := tab_map[TAB_FILE_GAME_CONFIG]; !ok {
		log.Err("%s failed! new %s failed!" , _func_ , TAB_FILE_GAME_CONFIG);
		return false;
	}
	
	
	return true;
}

