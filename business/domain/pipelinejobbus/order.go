package pipelinejobbus

import "github.com/MKAbdElrahman/cloudchamber/business/sdk/order"

// DefaultOrderBy represents the default ordering for pipeline jobs.
var DefaultOrderBy = order.NewBy(OrderByDateCreated, order.DESC)

// Set of fields that can be used for ordering.
const (
	OrderByJobID       = "job_id"
	OrderByDateCreated = "date_created"
	OrderByStatus      = "status"
)
