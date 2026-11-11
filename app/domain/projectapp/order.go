package projectapp

import "github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"

var orderByFields = map[string]string{
	"project_id":   projectbus.OrderByProjectID,
	"name":         projectbus.OrderByName,
	"date_created": projectbus.OrderByDateCreated,
}
