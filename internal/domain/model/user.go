package model

import "time"

// User는 사용자 도메인 엔티티다.
type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
