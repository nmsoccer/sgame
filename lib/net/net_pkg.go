package comm

import (
	"encoding/binary"
)

const(
  TAG_LEN=1
)


/* NET-PKG
Tag   +    Len         +          Value
(1B)     (1|2|4B)              (...)
|<-    head     ->|        |<- data ->|


*Tag: 1Byte
**Tag: 0 0 0 0 0 | 0 0 0  low 3bits: Bytes of len

*Len: 1 or 2 or 4Byte
**len [1,0xFF] :1Byte
**len (0xFF , 0xFFFF] :2Byte
**len (0xFFFF , 0xFFFFFFFF]: 4Byte

exam: 
0: num = 0xFE --> [0xFE]
1. num = 0x0102 --> [0x02],[0x01]
low 8bits pack to low 1Byte, high 8bits pack to high 1Byte
2. num = 0x01020304  --> [0x03,0x04] , [0x01 , 0x02]
low 16bits pack to net-endian  at low 2Bytes; high 16bits pack to net-endian at high two bytes; 
*/

/*
unpack head from buff
@return 0:data not-ready, -1:failed , ELSE:success (head_len)
*/
func UnPackHead(buff []byte , tag *uint8 , data_lenth *uint32) int {
	buff_len := len(buff);
	if buff_len < TAG_LEN {
		return 0;
	}
	
	//bytes of len
	var b_len uint8 = buff[0] & 0x07; //lowest 3bits
	if b_len != 1 && b_len != 2 && b_len !=4 {
		return -1;
	}
	
	//1Byte
	if b_len == 0x01 {
		if buff_len < TAG_LEN + 1 {
			return 0;
		}
		
		*tag = buff[0];
		*data_lenth = uint32(buff[1]);
		return TAG_LEN + 1;
	}
	
	//2Byte
	if b_len == 0x02 {
		if buff_len < TAG_LEN + 2 {
			return 0;
		}
		
		*tag = buff[0];
		*data_lenth = uint32(binary.BigEndian.Uint16(buff[1:3]));
		return TAG_LEN + 2;
	}
	
	//4Byte
	if buff_len < TAG_LEN + 4 {
		return 0;
	}
	
	*tag = buff[0];
	*data_lenth = binary.BigEndian.Uint32(buff[1:5]);
	return TAG_LEN + 4;
}


/*
pack head to buff
@return:  -1:failed else:success(head_len)
*/
func PackHead(buff []byte , data_lenth uint32) int {
	buff_len := len(buff);
	if data_lenth ==0 {
		return -1;
	}
	
	//lenth:1Byte
    if data_lenth <= 0xFF {
    	if buff_len < TAG_LEN+1 {
    		return -1;
    	}
    	
    	buff[0] = 0x01; //tag
    	buff[1] = uint8(data_lenth);
    	return 1+TAG_LEN;
    }
    
    //lenth:2Byte
    if data_lenth <= 0xFFFF {
    	if buff_len < TAG_LEN+2 {
    		return -1;
    	}
    	
    	buff[0] = 0x02; //tag
    	binary.BigEndian.PutUint16(buff[1:3], uint16(data_lenth));
    	return TAG_LEN+2;
    }
    
    //lenth:4Byte
    if buff_len < TAG_LEN+4 {
    	return -1;
    }
    
    buff[0] = 0x04; //tag
    binary.BigEndian.PutUint32(buff[1:5], data_lenth);
    return TAG_LEN+4;	
}


/*UnPackPkg from raw data
@return:(pkg_tag , pkg_data , pkg_len)
@pkg_tag 0xFF:error , 0:data not ready , else:success and valid tag of pkg
    if tag is valid then will return valid pkg_data and pkg_len
*/
func UnPackPkg(raw []byte) (uint8 , []byte , int){
    var tag uint8 = 0;
    var data_len_u32 uint32 = 0;
    var data_len int = 0;
    var raw_len = len(raw);
    
    //Get Head
    head_len := UnPackHead(raw, &tag, &data_len_u32);
      //error
    if head_len < 0 {
    	return 0xFF , nil , 0;
    }
    
      //not ready
    if head_len == 0 {
    	return 0 , nil , 0;
    }
    
    data_len = int(data_len_u32);
    //Get Data       
    if raw_len < (data_len + head_len) { //pkg-data not ready
    	return 0 , nil , 0;
    }
    
    pkg_data := raw[head_len:head_len+data_len];
    return tag , pkg_data , head_len+data_len;      	
}



/*
Pack pkg_data to pkg
@return: -1:failed else:success(pkg_len)
*/
func PackPkg(pkg_buff []byte , pkg_data []byte) int {
    //pack head
    head_len := PackHead(pkg_buff , uint32(len(pkg_data)));
    if head_len < 0 {
    	return -1;
    }
    
    //pkg enough?
    if len(pkg_buff) < head_len + len(pkg_data) {
    	return -1;
    }
    
    //copy data
    pkg_buff = append(pkg_buff[:head_len] , pkg_data...);
    return head_len + len(pkg_data);
}

