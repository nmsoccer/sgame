package proto

import (
  "fmt"
  "errors"
  "google.golang.org/protobuf/proto"
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