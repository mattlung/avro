package avro_test

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/heetch/avro"
)

var compatStringTests = []struct {
	m avro.CompatMode
	s string
}{
	{0, "NONE"},
	{avro.Backward, "BACKWARD"},
	{avro.Forward, "FORWARD"},
	{avro.Full, "FULL"},
	{avro.BackwardTransitive, "BACKWARD_TRANSITIVE"},
	{avro.ForwardTransitive, "FORWARD_TRANSITIVE"},
	{avro.FullTransitive, "FULL_TRANSITIVE"},
	{1 << 10, "UNKNOWN"},
}

func TestCompatString(t *testing.T) {
	c := qt.New(t)
	for _, test := range compatStringTests {
		c.Run(test.s, func(c *qt.C) {
			c.Assert(test.m.String(), qt.Equals, test.s)
		})
	}
}
