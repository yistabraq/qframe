package template

import (
	"encoding/json"

	"github.com/yistabraq/qframe/qerrors"

	"github.com/yistabraq/qframe/internal/column"
	"github.com/yistabraq/qframe/internal/index"
	"github.com/yistabraq/qframe/types"
)

// This file contains definitions for data and functions that need to be added
// manually for each data type.

// TODO: Probably need a more general aggregation pattern, int -> float (average for example)
var aggregations = map[string]func([]genericDataType) genericDataType{}

func (c Column) DataType() types.DataType {
	return types.None
}

// Functions not generated but needed to fulfill interface
func (c Column) AppendByteStringAt(buf []byte, i uint32) []byte {
	return nil
}

func (c Column) ByteSize() int {
	return 0
}

func (c Column) Equals(index index.Int, other column.Column, otherIndex index.Int) bool {
	return false
}

func (c Column) Filter(index index.Int, comparator interface{}, comparatee interface{}, bIndex index.Bool) error {
	return nil
}

func (c Column) FunctionType() types.FunctionType {
	return types.FunctionTypeBool
}

func (c Column) Marshaler(index index.Int) json.Marshaler {
	return nil
}

func (c Column) StringAt(i uint32, naRep string) string {
	return ""
}

func (c Column) Append(cols ...column.Column) (column.Column, error) {
	return nil, qerrors.New("Append", "Not implemented")
}

func (c Comparable) Compare(i, j uint32) column.CompareResult {
	return column.Equal
}

func (c Comparable) Hash(i uint32, seed uint64) uint64 {
	return 0
}
