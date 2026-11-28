package rundb

import (
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
)

// orderByFields maps a runbus ordering key to its database column name.
var orderByFields = map[string]string{
	runbus.OrderByRunID:       "run_id",
	runbus.OrderByDateCreated: "date_created",
	runbus.OrderByDateUpdated: "date_updated",
	runbus.OrderByPhase:       "phase",
}

// orderByClause builds an ORDER BY clause for the given ordering.
func orderByClause(orderBy order.By) (string, error) {
	by, exists := orderByFields[orderBy.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", orderBy.Field)
	}

	return " ORDER BY " + by + " " + orderBy.Direction, nil
}
