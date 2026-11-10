package pipelinejobapp

import "github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"

var orderByFields = map[string]string{
	"job_id":       pipelinejobbus.OrderByJobID,
	"date_created": pipelinejobbus.OrderByDateCreated,
	"status":       pipelinejobbus.OrderByStatus,
}
