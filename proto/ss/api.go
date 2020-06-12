package ss

import (
  "fmt"
  "errors"
  "google.golang.org/protobuf/proto"
)

const (
	MAX_SS_MSG_SIZE=(200*1024) //200k
)


func Pack(i interface{}) ([]byte, error) {
	fmt.Printf("try to pack...\n");
	switch i.(type) {
		case proto.Message:
			m , _ := i.(proto.Message);
			return proto.Marshal(m);
		
		default:
			break;	
	}
	
	return nil , errors.New("this type not support!");
}


func UnPack(b []byte, i interface{}) error {
	fmt.Printf("try to unpack...\n");
	switch i.(type) {
		case proto.Message:
		    m , _ := i.(proto.Message);
		    return proto.Unmarshal(b, m);
		default:
			break;    
	}
	
	return errors.New("this type not support");
}