package scenario

import "errors"

var (
	ErrScenarioNotFound   = errors.New("scenario not found")
	ErrScenarioNotPending = errors.New("scenario is not pending review")
)
