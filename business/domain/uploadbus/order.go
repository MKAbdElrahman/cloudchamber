package uploadbus

import "github.com/MKAbdElrahman/cloudchamber/business/sdk/order"

// DefaultOrderBy represents the default way we sort.
var DefaultOrderBy = order.NewBy(OrderByDateCreated, order.DESC)

// Set of fields that the results can be ordered by.
const (
	OrderByObjectKey   = "object_key"
	OrderByOwnerID     = "owner_id"
	OrderByStatus      = "status"
	OrderByDateCreated = "date_created"
	OrderByDateUpdated = "date_updated"
)
