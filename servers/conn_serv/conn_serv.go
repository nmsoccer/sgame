package main

import (
    ll "sgame/servers/conn_serv/lib"
    "flag"
    "fmt"
)


var config ll.Config;
var pconfig *ll.Config = &config;

//option
var name_space = flag.String("N", "", "name space in proc_bridge sys");
var proc_id = flag.Int("p", 0, "proc id in proc_bridge sys");
var config_file = flag.String("f", "", "config file");


func init() {
}

func parse_flag() bool {
    //check flag
	flag.Parse();
	if len(*name_space) <=0 || *proc_id<=0 || len(*config_file)<=0 {
		flag.PrintDefaults();
		return false;
	}
	pconfig.ProcId = *proc_id;
	pconfig.NameSpace = *name_space;
    pconfig.ConfigFile = *config_file;
    return true;	
}

func main() {		
	//parse flag
	if parse_flag() == false {
		return;
	}
	
    //comm set
	if ll.CommSet(pconfig) == false {
		fmt.Printf("comm set failed!\n");
		return;
	}
	
	//self set
	if ll.SelfSet(pconfig) == false {
		fmt.Printf("self set failed!\n");
		return;
	}
    
        
    //start server
    ll.ServerStart(pconfig);
}