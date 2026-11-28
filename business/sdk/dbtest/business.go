package dbtest

import (
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/auditbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/auditbus/extensions/auditotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/auditbus/stores/auditdb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus/extensions/runotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus/stores/rundb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/extensions/useraudit"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/extensions/userotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/stores/usercache"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/stores/userdb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/delegate"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/jmoiron/sqlx"
)

// BusDomain represents all the business domain apis needed for testing.
type BusDomain struct {
	Delegate *delegate.Delegate
	Audit    auditbus.ExtBusiness
	User     userbus.ExtBusiness
	Run      runbus.ExtBusiness
}

func newBusDomains(log *logger.Logger, db *sqlx.DB) BusDomain {
	delegate := delegate.New(log)

	auditOtelExt := auditotel.NewExtension()
	auditStorage := auditdb.NewStore(log, db)
	auditBus := auditbus.NewBusiness(log, auditStorage, auditOtelExt)

	userOtelExt := userotel.NewExtension()
	userAuditExt := useraudit.NewExtension(auditBus)
	userStorage := usercache.NewStore(log, userdb.NewStore(log, db), time.Hour)
	userBus := userbus.NewBusiness(log, delegate, userStorage, userOtelExt, userAuditExt)

	runOtelExt := runotel.NewExtension()
	runStorage := rundb.NewStore(log, db)
	runBus := runbus.NewBusiness(log, runStorage, runOtelExt)

	return BusDomain{
		Delegate: delegate,
		Audit:    auditBus,
		User:     userBus,
		Run:      runBus,
	}
}
