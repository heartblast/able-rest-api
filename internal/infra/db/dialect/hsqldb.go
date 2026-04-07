package dialect

import "fmt"

type hsqldbDialect struct{}

func (d hsqldbDialect) Name() string { return "hsqldb" }

func (d hsqldbDialect) Placeholder(_ int) string { return "?" }

func (d hsqldbDialect) Pagination(limit, offset int) string {
	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

func (d hsqldbDialect) CreateUserQuery() string {
	return `INSERT INTO users (name, email, created_at, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
}

func (d hsqldbDialect) GetUserByIDQuery() string {
	return `SELECT id, name, email, created_at, updated_at FROM users WHERE id = ?`
}

func (d hsqldbDialect) ListUsersQuery(limit, offset int) string {
	return `SELECT id, name, email, created_at, updated_at FROM users ORDER BY id` + d.Pagination(limit, offset)
}
