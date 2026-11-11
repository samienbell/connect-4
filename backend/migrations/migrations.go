// Package migrations embeds the SQL schema files. At start-up the server
// applies, in filename order, each file not yet recorded in the
// schema_migrations table. Never edit a file once it has been deployed; add a
// new one instead.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
