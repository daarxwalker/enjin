package keyvalue

import "embed"

// FS holds goose migrations with timestamp versions. Run them with a goose
// provider of their own (its own version table), so they never collide with
// the versions of the application migrations.
//
//go:embed *.sql
var FS embed.FS

// Tables lists the tables whose expired rows a cleaner.Cleaner should delete.
func Tables() []string {
	return []string{"enjin_cache", "enjin_csrf", "enjin_lock"}
}
