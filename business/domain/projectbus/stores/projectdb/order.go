package projectdb

import (
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
)

// OrderByFields maps a projectbus ordering key to its database column name.
var OrderByFields = map[string]string{
	projectbus.OrderByProjectID:   "project_id",
	projectbus.OrderByName:        "name",
	projectbus.OrderByDateCreated: "date_created",
}

// OrderByClause builds an ORDER BY clause for the given ordering.
func OrderByClause(orderBy order.By) (string, error) {
	by, exists := OrderByFields[orderBy.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", orderBy.Field)
	}

	return " ORDER BY " + by + " " + orderBy.Direction, nil
}
