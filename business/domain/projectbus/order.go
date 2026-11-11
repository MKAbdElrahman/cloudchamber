package projectbus

import "github.com/MKAbdElrahman/cloudchamber/business/sdk/order"

// DefaultOrderBy represents the default way we sort.
var DefaultOrderBy = order.NewBy(OrderByDateCreated, order.DESC)

// Set of fields that the results can be ordered by.
const (
	OrderByProjectID   = "a"
	OrderByName        = "b"
	OrderByDateCreated = "c"
)
