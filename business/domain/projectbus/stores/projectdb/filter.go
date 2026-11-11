package projectdb

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
)

// ApplyFilter appends a WHERE clause for the fields set on the filter.
func ApplyFilter(filter projectbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string

	if filter.ID != nil {
		data["project_id"] = *filter.ID
		wc = append(wc, "project_id = :project_id")
	}

	if filter.OwnerID != nil {
		data["owner_id"] = *filter.OwnerID
		wc = append(wc, "owner_id = :owner_id")
	}

	if filter.Name != nil {
		data["name"] = fmt.Sprintf("%%%s%%", filter.Name.String())
		wc = append(wc, "name LIKE :name")
	}

	if len(wc) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(wc, " AND "))
	}
}
