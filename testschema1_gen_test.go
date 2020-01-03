// Code generated by avrogen. DO NOT EDIT.

package avro_test

import "github.com/rogpeppe/avro"

type TestRecord struct {
	A int
	B int
}

// AvroRecord implements the avro.AvroRecord interface.
func (TestRecord) AvroRecord() avro.RecordInfo {
	return avro.RecordInfo{
		Schema: `{"fields":[{"default":42,"name":"A","type":{"type":"int"}},{"name":"B","type":{"type":"int"}}],"name":"TestRecord","type":"record"}`,
		Defaults: []func() interface{}{
			0: func() interface{} {
				return 42
			},
		},
	}
}

// TODO implement MarshalBinary and UnmarshalBinary methods?
