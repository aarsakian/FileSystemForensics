package filtermanager

import (
	metadata "github.com/aarsakian/FileSystemForensics/FS"
	"github.com/aarsakian/FileSystemForensics/filters"
)

type FilterManager struct {
	filters []filters.Filter
}

func (filterManager *FilterManager) Register(filter filters.Filter) {
	filterManager.filters = append(filterManager.filters, filter)
}

func (filterManager FilterManager) ApplyFilters(records []metadata.Record) []metadata.Record {
	for _, filter := range filterManager.filters {
		records = filter.Execute(records)
	}
	return records
}
