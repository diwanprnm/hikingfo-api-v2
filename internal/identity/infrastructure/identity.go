// Package identity/infrastructure is deliberately under one Go package
// (password.go, user_repo.go, session_repo.go, mailer.go, google.go) so the
// pgx helpers they share (nilOr/deref/classify/bioParam) live once. Each file
// documents the port it satisfies; the composition root wires them in cmd/api.
package infrastructure
