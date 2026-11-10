package commondb

import (
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
)

// OrderByFields maps ordering keys to database columns.
var OrderByFields = map[string]string{
	pipelinejobbus.OrderByJobID:       "job_id",
	pipelinejobbus.OrderByDateCreated: "date_created",
	pipelinejobbus.OrderByStatus:      "status",
}

// OrderByClause builds an ORDER BY clause.
func OrderByClause(orderBy order.By) (string, error) {
	by, exists := OrderByFields[orderBy.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", orderBy.Field)
	}

	return " ORDER BY " + by + " " + orderBy.Direction, nil
}
