package projects

import "time"

type Project struct {
	ID          string
	Name        string
	UserCreated int
	CreatedAt   time.Time
}
