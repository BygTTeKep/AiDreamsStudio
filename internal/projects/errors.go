package projects

import "errors"

var (
	ErrProjectNotFound = errors.New("Project not found")
	ErrCreateProject   = errors.New("Error create project")
)
