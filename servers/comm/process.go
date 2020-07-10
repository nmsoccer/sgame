package comm

import (
  "os"
  "os/exec"
	"time"

	//"path/filepath"
  "syscall"
  "strconv"
)

const (
    TEMP_DIR_PREFIX = "/tmp/sg_run_2020"
    REWRITE_LOCK_FILE_TICK = (3600 * 6); //seconds maintain lock file in /tmp
)


func maintain_lock_file(pconfig *CommConfig) {
	var _func_ = "<maintain_lock_file>";
	log := pconfig.Log;

	for {
		time.Sleep(REWRITE_LOCK_FILE_TICK * time.Second);

		//seek
		_, err := pconfig.LockFile.Seek(0, 0);
		if err != nil {
			log.Err("%s seek failed! err:%v" , _func_ , err);
			continue;
		}

		//rewrite
		my_self := os.Getpid();
		_ , err = pconfig.LockFile.Write([]byte(strconv.Itoa(my_self)));
		if err != nil {
			log.Info("%s write my self procid:%d to lock file %s failed! err:%v\n" , _func_ , my_self ,
				pconfig.LockFile.Name() , err);
		} else {
			log.Info("%s to %s success" , _func_ , pconfig.LockFile.Name());
		}

	}

}



//uniq process lock file
func LockUniqFile(pconfig *CommConfig , name_space string , proc_id int , proc_name string) bool{
	var _func_ = "<LockFile>";
    var target_dir	= TEMP_DIR_PREFIX + "/" + name_space;
    var log = pconfig.Log;
    
    //mkdir
    err := os.MkdirAll(target_dir, 0755);
    if err != nil {
    	log.Err("%s mkdir:%s failed! err:%v\n", _func_ , target_dir , err);
    	return false;
    }
    
    //open file
    var tmp_file = target_dir + "/" + proc_name + "_" + strconv.Itoa(proc_id) + ".pid";
    file , err := os.OpenFile(tmp_file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744);
    if err != nil {
    	log.Err("%s create tmp file:%s failed! err:%v\n", _func_ , tmp_file , err);
    	return false;
    }
    //defer file.Close();
    
    //try lock file
    err = syscall.Flock(int(file.Fd()) , syscall.LOCK_EX| syscall.LOCK_NB);
    if err != nil {
    	log.Err("%s lock %s failed! check other process exist! err:%v\n", _func_ , tmp_file , err);
    	last_proc := make([]byte , 24);
    	n , err := file.Read(last_proc);
    	if err == nil && n>0 {
    		last_proc = last_proc[0:n];
    		log.Err("old proc :%s", string(last_proc));
    	}
    	return false;
    }
    
    //write self id
    my_self := os.Getpid();
    _ , err = file.Write([]byte(strconv.Itoa(my_self)));
    if err != nil {
    	log.Info("%s write my self procid:%d to %s failed! err:%v\n" , _func_ , my_self , tmp_file , err);
    } else {
    	log.Info("%s lock %s success" , _func_ , tmp_file);
    }
    pconfig.LockFile = file;

    //timing rewrite
    go maintain_lock_file(pconfig);
    return true;
}

//unlock
func UnlockUniqFile(pconfig *CommConfig , name_space string , proc_id int , proc_name string) bool {
	var _func_ = "<UnlockUniqFile>";
	var target_dir	= TEMP_DIR_PREFIX + "/" + name_space;
	var tmp_file = target_dir + "/" + proc_name + "_" + strconv.Itoa(proc_id) + ".pid";
	var log = pconfig.Log;

	/*
	//open
	file , err := os.OpenFile(tmp_file, os.O_RDWR , 0);
    if err != nil {
    	log.Err("%s create tmp file:%s failed! err:%v\n", _func_ , tmp_file , err);
    	return false;
    }
    defer file.Close();*/
	file := pconfig.LockFile;
	
	//unlock
	err := syscall.Flock(int(file.Fd()) , syscall.LOCK_UN);
	if err != nil {
		log.Err("%s unlock %s failed! err:%v\n", _func_ , tmp_file , err);
		return false;
	}
	log.Info("%s unlock %s success! \n", _func_ , tmp_file);
	return true;
}




//daemon process
func Daemonize() {
	
    if os.Getppid() != 1 {
    	//file_path , _ := filepath.Abs(os.Args[0]);
    	cmd := exec.Command(os.Args[0], os.Args[1:]...);
    	cmd.Stdin = os.Stdin;
    	cmd.Stdout = os.Stdout;
    	cmd.Stderr = os.Stderr;
    	cmd.SysProcAttr = &syscall.SysProcAttr{};
    	cmd.SysProcAttr.Setsid = true;
    	
    	err := cmd.Start();
    	if err != nil {
    		cmd.Process.Release();
    		os.Exit(0);
    	}
    	os.Exit(0);
    }  	
		
}


