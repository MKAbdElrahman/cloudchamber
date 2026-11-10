package commondb

import (
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
)

// OrderByFields maps ordering keys to database columns.
var OrderByFields = map[string]string{
	uploadbus.OrderByObjectKey:   "object_key",
	uploadbus.OrderByOwnerID:     "owner_id",
	uploadbus.OrderByStatus:      "status",
	uploadbus.OrderByDateCreated: "date_created",
	uploadbus.OrderByDateUpdated: "date_updated",
}

// OrderByClause builds an ORDER BY clause.
func OrderByClause(orderBy order.By) (string, error) {
	by, exists := OrderByFields[orderBy.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", orderBy.Field)
	}

	return " ORDER BY " + by + " " + orderBy.Direction, nil
}
