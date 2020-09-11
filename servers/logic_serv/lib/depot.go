package lib

import (
	"fmt"
	"sgame/proto/ss"
	"sgame/servers/comm"
)

var generate_item_seq uint16 = 1;

func InitUserDepot(pconfig *Config , pdepot *ss.UserDepot , uid int64) {
	var _func_ = "<InitUserDepot>";
    log := pconfig.Comm.Log;
    log.Info("%s uid:%d" , _func_ , uid);

    if pdepot.Items == nil {
    	pdepot.Items = make(map[int64] *ss.Item);
	}


    var instid int64 = 0;
    //put 10 items
    for i:=0; i<10; i++ {
        instid = comm.GenerateLocalId(int16(pconfig.ProcId & 0xFFFF) , &generate_item_seq);
        pdepot.Items[instid] = new(ss.Item);
        pdepot.Items[instid].Resid = int32(i+1001);
        pdepot.Items[instid].Count = 10;
        pdepot.Items[instid].Instid = instid;
        //log.Debug("%s <%d> item:%v" , _func_ , instid , pdepot.Items[instid]);
        pconfig.NetLog.Log("|" , fmt.Sprintf("%d|%s|ItemFlow|%d|%d|%d|%d|%d" , pconfig.ProcId , pconfig.ProcName ,
        	uid , instid , pdepot.Items[instid].Resid , pdepot.Items[instid].Count , 1))
        pdepot.ItemsCount++;
	}

    log.Info("%s finish! uid:%d depot:%v" , _func_ , uid , *pdepot);
    return;
}
