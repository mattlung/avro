// Code generated by generatetestcode.go; DO NOT EDIT.

package simpleFixed

import (
	"testing"

	"github.com/rogpeppe/avro/avro-generate-go/internal/testutil"
)

var test = testutil.RoundTripTest{
	InDataJSON: `{
                "F": "abcde"
            }`,
	OutDataJSON: `{
                "F": "abcde"
            }`,
	InSchema: `{
                "name": "R",
                "type": "record",
                "fields": [
                    {
                        "name": "F",
                        "type": {
                            "name": "five",
                            "type": "fixed",
                            "size": 5
                        }
                    }
                ]
            }`,
	OutSchema: `{
                "name": "R",
                "type": "record",
                "fields": [
                    {
                        "name": "F",
                        "type": {
                            "name": "five",
                            "type": "fixed",
                            "size": 5
                        }
                    }
                ]
            }`,
	GoType: new(R),
}

func TestGeneratedCode(t *testing.T) {
	test.Test(t)
}