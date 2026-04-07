package dialect

import "fmt"

type postgresDialect struct{}

func (d postgresDialect) Name() string { return "postgres" }

func (d postgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (d postgresDialect) Pagination(limit, offset int) string {
	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

func (d postgresDialect) CreateUserQuery() string {
	return `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, created_at, updated_at`
}

func (d postgresDialect) GetUserByIDQuery() string {
	return `SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`
}

func (d postgresDialect) ListUsersQuery(limit, offset int) string {
	return `SELECT id, name, email, created_at, updated_at FROM users ORDER BY id` + d.Pagination(limit, offset)
}
