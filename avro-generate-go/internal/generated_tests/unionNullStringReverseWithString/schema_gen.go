// Code generated by avrogen. DO NOT EDIT.

package unionNullStringReverseWithString

import "github.com/rogpeppe/avro"

type R struct {
	OptionalString *string
}

// AvroRecord implements the avro.AvroRecord interface.
func (R) AvroRecord() avro.RecordInfo {
	return avro.RecordInfo{
		Schema: `{"fields":[{"name":"OptionalString","type":["string","null"]}],"name":"R","type":"record"}`,
		Unions: [][]interface{}{
			0: {new(string), nil},
		},
	}
}

// TODO implement MarshalBinary and UnmarshalBinary methods?
