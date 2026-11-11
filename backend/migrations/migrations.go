// Package migrations embeds the SQL schema files. The server applies them in
// filename order at start-up, so each file must be safe to re-run
// (CREATE ... IF NOT EXISTS, etc.).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
