package dependency

import (
	"context"
)

type Dependency struct {
	Name     string
	Timeout  string
	Ctx      context.Context
	Function func() error
	Cancel   context.CancelFunc
}

type State struct {
	State int8
}

const CHECKING = 0
const SUCCESS = 1
const FAILED = 2

type Result struct {
	Data string
}
