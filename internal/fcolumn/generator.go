package fcolumn

import (
	"bytes"

	"github.com/yistabraq/qframe/filter"
	"github.com/yistabraq/qframe/internal/maps"
	"github.com/yistabraq/qframe/internal/template"
)

//go:generate qfgenerate -source=ffilter -dst-file=filters_gen.go
//go:generate qfgenerate -source=fdoc -dst-file=doc_gen.go

func spec(name, operator, templateStr string) template.Spec {
	return template.Spec{
		Name:     name,
		Template: templateStr,
		Values:   map[string]interface{}{"name": name, "dataType": "float64", "operator": operator}}
}

func colConstComparison(name, operator string) template.Spec {
	return spec(name, operator, template.BasicColConstComparison)
}

func colColComparison(name, operator string) template.Spec {
	return spec(name, operator, template.BasicColColComparison)
}

func GenerateFilters() (*bytes.Buffer, error) {
	// If adding more filters here make sure to also add a reference to them
	// in the corresponding filter map so that they can be looked up.
	return template.GenerateFilters("fcolumn", []template.Spec{
		colConstComparison("lt", filter.Lt),
		colConstComparison("lte", filter.Lte),
		colConstComparison("gt", filter.Gt),
		colConstComparison("gte", filter.Gte),
		colConstComparison("eq", "=="), // Go eq ("==") differs from qframe eq ("=")
		colConstComparison("neq", filter.Neq),
		colColComparison("lt2", filter.Lt),
		colColComparison("lte2", filter.Lte),
		colColComparison("gt2", filter.Gt),
		colColComparison("gte2", filter.Gte),
		colColComparison("eq2", "=="), // Go eq ("==") differs from qframe eq ("=")
		colColComparison("neq2", filter.Neq),
	})
}

func GenerateDoc() (*bytes.Buffer, error) {
	return template.GenerateDocs(
		"fcolumn",
		maps.StringKeys(filterFuncs0, filterFuncs1, filterFuncs2),
		maps.StringKeys(aggregations))
}
