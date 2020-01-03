// Code generated by avrogen. DO NOT EDIT.

package arrayDefault

import "github.com/rogpeppe/avro"

type R struct {
	ArrayOfInt []int `json:"arrayOfInt"`
}

// AvroRecord implements the avro.AvroRecord interface.
func (R) AvroRecord() avro.RecordInfo {
	return avro.RecordInfo{
		Schema: `{"fields":[{"default":[2,3,4],"name":"arrayOfInt","type":{"items":"int","type":"array"}}],"name":"R","type":"record"}`,
		Defaults: []func() interface{}{
			0: func() interface{} {
				return []int{2, 3, 4}
			},
		},
	}
}

// TODO implement MarshalBinary and UnmarshalBinary methods?
