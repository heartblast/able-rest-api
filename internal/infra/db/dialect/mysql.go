package dialect

import "fmt"

type mysqlDialect struct{}

func (d mysqlDialect) Name() string { return "mysql" }

func (d mysqlDialect) Placeholder(_ int) string { return "?" }

func (d mysqlDialect) Pagination(limit, offset int) string {
	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

func (d mysqlDialect) CreateUserQuery() string {
	return `INSERT INTO users (name, email) VALUES (?, ?)`
}

func (d mysqlDialect) GetUserByIDQuery() string {
	return `SELECT id, name, email, created_at, updated_at FROM users WHERE id = ?`
}

func (d mysqlDialect) ListUsersQuery(limit, offset int) string {
	return `SELECT id, name, email, created_at, updated_at FROM users ORDER BY id` + d.Pagination(limit, offset)
}
