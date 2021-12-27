package qframe

import (
	"github.com/yistabraq/qframe/internal/grouper"
	"github.com/yistabraq/qframe/internal/icolumn"
	"github.com/yistabraq/qframe/internal/index"
	"github.com/yistabraq/qframe/qerrors"
	"github.com/yistabraq/qframe/types"
)

// GroupStats contains internal statistics for grouping.
// Clients should not depend on this for any type of decision making. It is strictly "for info".
// The layout may change if the underlying grouping mechanisms change.
type GroupStats grouper.GroupStats

// Grouper contains groups of rows produced by the QFrame.GroupBy function.
type Grouper struct {
	indices        []index.Int
	groupedColumns []string
	columns        []namedColumn
	columnsByName  map[string]namedColumn
	Err            error
	Stats          GroupStats
}

// Aggregation represents a function to apply to a column.
type Aggregation struct {
	// Fn is the aggregation function to apply.
	//
	// IMPORTANT: For pointer and reference types you must not assume that the data passed argument
	// to this function is valid after the function returns. If you plan to keep it around you need
	// to take a copy of the data.
	Fn types.SliceFuncOrBuiltInId

	// Column is the name of the column to apply the aggregation to.
	Column string

	// As can be used to specify the destination column name, if not given defaults to the
	// value of Column.
	As string
}

// Aggregate applies the given aggregations to all row groups in the Grouper.
//
// Time complexity O(m*n) where m = number of aggregations, n = number of rows.
func (g Grouper) Aggregate(aggs ...Aggregation) QFrame {
	if g.Err != nil {
		return QFrame{Err: g.Err}
	}

	// Loop over all groups and pick the first row in each of the groups.
	// This index will be used to populate the grouped by columns below.
	firstElementIx := make(index.Int, len(g.indices))
	for i, ix := range g.indices {
		firstElementIx[i] = ix[0]
	}

	newColumnsByName := make(map[string]namedColumn, len(g.groupedColumns)+len(aggs))
	newColumns := make([]namedColumn, 0, len(g.groupedColumns)+len(aggs))
	for i, colName := range g.groupedColumns {
		col := g.columnsByName[colName]
		col.pos = i
		col.Column = col.Subset(firstElementIx)
		newColumnsByName[colName] = col
		newColumns = append(newColumns, col)
	}

	var err error
	for _, agg := range aggs {
		col, ok := g.columnsByName[agg.Column]
		if !ok {
			return QFrame{Err: qerrors.New("Aggregate", unknownCol(agg.Column))}
		}

		newColumnName := agg.Column
		if agg.As != "" {
			newColumnName = agg.As
		}
		col.name = newColumnName

		_, ok = newColumnsByName[newColumnName]
		if ok {
			return QFrame{Err: qerrors.New(
				"Aggregate",
				"cannot aggregate on column that is part of group by or is already an aggregate: %s", newColumnName)}
		}

		if agg.Fn == "count" {
			// Special convenience case for "count" which would normally require a cast from
			// any other type of column to int before being executed.
			counts := make([]int, len(g.indices))
			for i, ix := range g.indices {
				counts[i] = len(ix)
			}

			col.Column = icolumn.New(counts)
		} else {
			col.Column, err = col.Aggregate(g.indices, agg.Fn)
			if err != nil {
				return QFrame{Err: qerrors.Propagate("Aggregate", err)}
			}
		}

		newColumnsByName[newColumnName] = col
		newColumns = append(newColumns, col)
	}

	return QFrame{columns: newColumns, columnsByName: newColumnsByName, index: index.NewAscending(uint32(len(g.indices)))}
}

// QFrames returns a slice of QFrame where each frame represents the content of one group.
//
// Time complexity O(n) where n = number of groups.
func (g Grouper) QFrames() ([]QFrame, error) {
	if g.Err != nil {
		return nil, g.Err
	}

	baseFrame := QFrame{columns: g.columns, columnsByName: g.columnsByName, index: index.Int{}}
	result := make([]QFrame, len(g.indices))
	for i, ix := range g.indices {
		result[i] = baseFrame.withIndex(ix)
	}
	return result, nil
}
