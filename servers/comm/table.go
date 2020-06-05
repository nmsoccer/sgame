package comm

type TableMap map[string] interface{};

const (
  TablePath="./table"	
  HeadTableFile="__head.json"
)
type HeadTable struct {
	ReloadList []string `json:"reload_list"`
}

func LoadTableFiles(table_map TableMap , pconfig *CommConfig) bool{
	return load_table_files(table_map, pconfig);
}

func ReLoadTableFiles(table_map TableMap , pconfig *CommConfig) bool{
	return reload_table_files(table_map, pconfig);
}



/*----------------------------Inner Func------------------------------*/
func load_table_files(table_map TableMap , pconfig *CommConfig) bool{
	var _func_ = "<load_table_files>";	
	var log = pconfig.Log;
	
	//load table file
	for file_name , data := range table_map {		
		//load data
		ok := load_table_file(file_name, data , pconfig);
		if !ok {
			log.Err("%s load %s failed!" , _func_ , file_name);
			return false;
		}
		
	}
	
	//finish
	log.Info("%s finish!" , _func_);
	return true;
}



//load table files
//which:0: load all tables 1:reload table
func reload_table_files(table_map TableMap , pconfig *CommConfig) bool{
	var _func_ = "<reload_table_files>";
	var head_table HeadTable;
	
	log := pconfig.Log;
	//load head table
	if load_head_table_file(&head_table , pconfig) != true {
	    log.Err("%s load head failed!" , _func_);
	    return false;	
	}
	
	//load-list
	var load_list  = head_table.ReloadList;
	for _ , file_name := range load_list {
		//check data
		data , ok := table_map[file_name];
		if !ok {
			log.Err("%s failed! %s has no struct!" , _func_ , file_name);
			return false;
		}
		
		//load data
		ok = load_table_file(file_name, data , pconfig);
		if !ok {
			log.Err("%s load %s failed!" , _func_ , file_name);
			return false;
		}
		
	}
	
	//finish
	log.Info("%s finish!" , _func_);
	return true;
}


func load_head_table_file(phead *HeadTable , pconfig *CommConfig) bool{
	var _func_ = "<load_head_table_file>";
	log := pconfig.Log;
	var head_table_path string = TablePath + "/" + HeadTableFile;
	ret := LoadJsonFile(head_table_path, phead , pconfig);
	if ret != true {
		log.Err("%s %s failed!" , _func_ , head_table_path);
		return false;
	}
	//log.Info("%s %s success! content:%v" , _func_ , head_table_path , *phead);
	return true;
}

func load_table_file(file_name string , data interface{} , pconfig *CommConfig) bool {
	var _func_ = "<load_table_file>";
	log := pconfig.Log;
	var file_path = TablePath + "/" + file_name;
	ret := LoadJsonFile(file_path, data , pconfig);
	if ret != true {
		log.Err("%s %s failed!" , _func_ , file_path);
		return false;
	}
	//log.Info("%s %s success! content:%v" , _func_ , file_path , data);
	return true;
}

