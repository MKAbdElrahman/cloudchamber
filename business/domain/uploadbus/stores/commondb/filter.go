package commondb

import (
	"bytes"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
)

// ApplyFilter appends a WHERE clause to buf based on filter.
func ApplyFilter(filter uploadbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string

	if filter.ObjectKey != nil {
		data["object_key"] = *filter.ObjectKey
		wc = append(wc, "object_key = :object_key")
	}

	if filter.OwnerID != nil {
		data["owner_id"] = *filter.OwnerID
		wc = append(wc, "owner_id = :owner_id")
	}

	if filter.Status != nil {
		data["status"] = filter.Status.String()
		wc = append(wc, "status = :status")
	}

	if filter.SourceRef != nil {
		data["source_ref"] = *filter.SourceRef
		wc = append(wc, "source_ref = :source_ref")
	}

	if len(wc) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(wc, " AND "))
	}
}
