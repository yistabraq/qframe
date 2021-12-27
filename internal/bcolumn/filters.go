package bcolumn

import (
	"github.com/yistabraq/qframe/filter"
	"github.com/yistabraq/qframe/internal/index"
)

var filterFuncs = map[string]func(index.Int, []bool, bool, index.Bool){
	filter.Eq:  eq,
	filter.Neq: neq,
}

var filterFuncs2 = map[string]func(index.Int, []bool, []bool, index.Bool){
	filter.Eq:  eq2,
	filter.Neq: neq2,
}
