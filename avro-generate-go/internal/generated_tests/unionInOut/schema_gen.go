// Code generated by avrogen. DO NOT EDIT.

package unionInOut

import "github.com/rogpeppe/avro"

type PrimitiveUnionTestRecord struct {
	UnionField interface{}
}

// AvroRecord implements the avro.AvroRecord interface.
func (PrimitiveUnionTestRecord) AvroRecord() avro.RecordInfo {
	return avro.RecordInfo{
		Schema: `{"fields":[{"default":1234,"name":"UnionField","type":["int","long","float","double","string","boolean","null"]}],"name":"PrimitiveUnionTestRecord","type":"record"}`,
		Defaults: []func() interface{}{
			0: func() interface{} {
				return 1234
			},
		},
		Unions: [][]interface{}{
			0: {new(int), new(int64), new(float32), new(float64), new(string), new(bool), nil},
		},
	}
}

// TODO implement MarshalBinary and UnmarshalBinary methods?
