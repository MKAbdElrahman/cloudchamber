package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

//go:embed sql/messagedb
var messageDBFS embed.FS

// installMessageDB creates the Message DB 1.3.0 schema when it is missing.
// The scripts are idempotent only for a database that has never installed them,
// so this runs once, before the migration that copies existing steps.
func installMessageDB(ctx context.Context, db *sqlx.DB) error {
	var installed bool
	if err := db.QueryRowxContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE n.nspname = 'message_store' AND p.proname = 'write_message'
		)`).Scan(&installed); err != nil {
		return fmt.Errorf("message store probe: %w", err)
	}
	if !installed {
		var files []string
		err := fs.WalkDir(messageDBFS, "sql/messagedb", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".sql") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return fmt.Errorf("message store files: %w", err)
		}
		sort.Strings(files)

		for _, path := range files {
			if strings.HasSuffix(path, "zz-cloudchamber.sql") {
				continue
			}
			if err := execMessageSQL(ctx, db, path); err != nil {
				return err
			}
		}
	}
	return execMessageSQL(ctx, db, "sql/messagedb/zz-cloudchamber.sql")
}

func execMessageSQL(ctx context.Context, db *sqlx.DB, path string) error {
	body, err := messageDBFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("exec %s: %w", path, err)
	}
	return nil
}
