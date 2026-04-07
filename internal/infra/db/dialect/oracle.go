package dialect

import "fmt"

type oracleDialect struct{}

func (d oracleDialect) Name() string { return "oracle" }

func (d oracleDialect) Placeholder(index int) string {
	return fmt.Sprintf(":%d", index)
}

func (d oracleDialect) Pagination(limit, offset int) string {
	return fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
}

func (d oracleDialect) CreateUserQuery() string {
	return `INSERT INTO users (id, name, email, created_at, updated_at) VALUES (users_seq.NEXTVAL, :1, :2, SYSTIMESTAMP, SYSTIMESTAMP)`
}

func (d oracleDialect) GetUserByIDQuery() string {
	return `SELECT id, name, email, created_at, updated_at FROM users WHERE id = :1`
}

func (d oracleDialect) ListUsersQuery(limit, offset int) string {
	return `SELECT id, name, email, created_at, updated_at FROM users ORDER BY id` + d.Pagination(limit, offset)
}
