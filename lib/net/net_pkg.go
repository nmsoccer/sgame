/*
=====This is a GO API file for parse server-client transport pkg in SGame Framework=====
* more info:https://github.com/nmsoccer/sgame
=======================================================================================
NET-PKG
Tag   +    Len         +          Value
(1B)     (1|2|4B)                 (...)
|<-    head     ->|           |<- data ->|


*Tag: 1Byte
**Tag: 0 0 0 0 0 | 0 0 0  high-5bits:option , low-3bits: Bytes of len

*Len: 1 or 2 or 4Byte
**len [1,0xFF] :1Byte
**len (0xFF , 0xFFFF] :2Byte
**len (0xFFFF , 0xFFFFFFFF]: 4Byte

*/
package comm


import (
	"encoding/binary"
)

const (
	TAG_LEN = 1
	INT_MAX = 0x7FFFFFF0
	//pkg option
	PKG_OP_NORMAL = 0  //normal pkg
	PKG_OP_ECHO   = 1   //echo client <-> tcp-serv
	PKG_OP_VALID   = 2  //valid connection client-->server[validate] server-->client[enc_key if enc enable]
	PKG_OP_RSA_NEGO = 3  //encrypt by rsa_pub_key to negotiate des key client-->server[encrypted key] server-->client[result]
	PKG_OP_MAX    = 32 //max option value

	//VALID_KEY
	CONN_VALID_KEY = "c#s..x*.39&suomeI./().32&show+me_tHe_m0ney$"
)


/*
Pack pkg_data to pkg.
@pkg_option: ==0 normal pkg.  > 0 PKG_OP_XX means special pkg to server
@return: -1:failed -2:buff_len not enough >0:success(pkg_len)
*/
func PackPkg(pkg_buff []byte, pkg_data []byte, pkg_option uint8) int {
	//pack head
	head_len := pack_head(pkg_buff, uint32(len(pkg_data)), pkg_option)
	if head_len <= 0 {
		return -1
	}

	//pkg enough?
	if len(pkg_buff) < head_len+len(pkg_data) {
		return -2
	}

	//copy data
	pkg_buff = append(pkg_buff[:head_len], pkg_data...)
	return head_len + len(pkg_data)
}

/*
UnPackPkg from raw data
@return:(pkg_tag , pkg_data , pkg_len)
@pkg_tag 0xFF:error , 0:data not ready , else:success and valid tag of pkg
    if tag is valid then will return valid pkg_data and pkg_len
*/
func UnPackPkg(raw []byte) (uint8, []byte, int) {
	var tag uint8 = 0
	var data_len_u32 uint32 = 0
	var data_len int = 0
	var raw_len = len(raw)

	//Get Head
	head_len := unpack_head(raw, &tag, &data_len_u32)
	//error
	if head_len < 0 {
		return 0xFF, nil, 0
	}

	//not ready
	if head_len == 0 {
		return 0, nil, 0
	}

	data_len = int(data_len_u32)
	//Get Data
	if raw_len < (data_len + head_len) { //pkg-data not ready
		return 0, nil, data_len + head_len
	}

	pkg_data := raw[head_len : head_len+data_len]
	return tag, pkg_data, head_len + data_len
}

//Get pkg-option
//@return:PKG_OP_XX
func PkgOption(tag uint8) uint8 {
	return tag >> 3
}

//predict pkg-len according to data_len
//-1:if data_len illegal else pkg-len
func GetPkgLen(data_len int) int {
	if data_len < 0 || data_len >= INT_MAX {
		return -1
	}

	if data_len <= 0xFF {
		return TAG_LEN + 1 + data_len
	}

	if data_len <= 0xFFFF {
		return TAG_LEN + 2 + data_len
	}

	if data_len >= INT_MAX {
		return -1
	}

	return TAG_LEN + 4 + data_len
}

/*---------------------STATIC FUNCT------------------------*/
/*
unpack head from buff
@return 0:data not-ready, -1:failed , ELSE:success (head_len)
*/
func unpack_head(buff []byte, tag *uint8, data_lenth *uint32) int {
	buff_len := len(buff)
	if buff_len < TAG_LEN {
		return 0
	}

	//bytes of len
	var b_len uint8 = buff[0] & 0x07 //lowest 3bits
	if b_len != 1 && b_len != 2 && b_len != 4 {
		return -1
	}

	//1Byte
	if b_len == 0x01 {
		if buff_len < TAG_LEN+1 {
			return 0
		}

		*tag = buff[0]
		*data_lenth = uint32(buff[1])
		return TAG_LEN + 1
	}

	//2Byte
	if b_len == 0x02 {
		if buff_len < TAG_LEN+2 {
			return 0
		}

		*tag = buff[0]
		*data_lenth = uint32(binary.BigEndian.Uint16(buff[1:3]))
		return TAG_LEN + 2
	}

	//4Byte
	if buff_len < TAG_LEN+4 {
		return 0
	}

	*tag = buff[0]
	*data_lenth = binary.BigEndian.Uint32(buff[1:5])
	return TAG_LEN + 4
}

/*
pack head to buff
@return:  -1:failed else:success(head_len)
*/
func pack_head(buff []byte, data_lenth uint32, pkg_option uint8) int {
	buff_len := len(buff)
	if data_lenth == 0 {
		return -1
	}

	if pkg_option >= PKG_OP_MAX { //only 2^5
		return -1
	}

	//lenth:1Byte
	if data_lenth <= 0xFF {
		if buff_len < TAG_LEN+1 {
			return -1
		}

		buff[0] = 0x01 //tag
		buff[0] |= (pkg_option << 3)
		buff[1] = uint8(data_lenth)
		return 1 + TAG_LEN
	}

	//lenth:2Byte
	if data_lenth <= 0xFFFF {
		if buff_len < TAG_LEN+2 {
			return -1
		}

		buff[0] = 0x02 //tag
		buff[0] |= (pkg_option << 3)
		binary.BigEndian.PutUint16(buff[1:3], uint16(data_lenth))
		return TAG_LEN + 2
	}

	//lenth:4Byte
	if buff_len < TAG_LEN+4 {
		return -1
	}

	buff[0] = 0x04 //tag
	buff[0] |= (pkg_option << 3)
	binary.BigEndian.PutUint32(buff[1:5], data_lenth)
	return TAG_LEN + 4
}
