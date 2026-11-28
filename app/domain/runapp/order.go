package runapp

import "github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"

var orderByFields = map[string]string{
	"run_id":       runbus.OrderByRunID,
	"date_created": runbus.OrderByDateCreated,
	"date_updated": runbus.OrderByDateUpdated,
	"phase":        runbus.OrderByPhase,
}
