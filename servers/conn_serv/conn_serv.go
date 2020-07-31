package main

import (
	"flag"
	"fmt"
	"sgame/servers/conn_serv/lib"
)

var config lib.Config
var pconfig *lib.Config = &config

//option
var name_space = flag.String("N", "", "name space in proc_bridge sys")
var proc_id = flag.Int("p", 0, "proc id in proc_bridge sys")
var config_file = flag.String("f", "", "config file")
var proc_name = flag.String("P", "", "proc name ")
var daemonize = flag.Bool("D" , false , "run in daemonize mode")

func init() {
}

func parse_flag() bool {
	//check flag
	flag.Parse()
	if len(*name_space) <= 0 || *proc_id <= 0 || len(*config_file) <= 0 || len(*proc_name) <= 0 {
		flag.PrintDefaults()
		return false
	}
	pconfig.ProcId = *proc_id
	pconfig.NameSpace = *name_space
	pconfig.ConfigFile = *config_file
	pconfig.ProcName = *proc_name
	pconfig.Daemon = *daemonize
	return true
}

func main() {
	//parse flag
	if parse_flag() == false {
		return
	}

	//comm set
	if lib.CommSet(pconfig) == false {
		fmt.Printf("comm set failed!\n")
		return
	}

	//self set
	if lib.LocalSet(pconfig) == false {
		fmt.Printf("self set failed!\n")
		return
	}

	//start server
	lib.ServerStart(pconfig)
}
