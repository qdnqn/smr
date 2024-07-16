package status

import (
	"github.com/hmdsefi/gograph"
	"time"
)

type Status struct {
	State                  StatusState
	LastReadiness          bool
	LastReadinessTimestamp time.Time
	StateMachine           gograph.Graph[StatusState] `json:"-"`
	Reconciling            bool
	LastUpdate             time.Time
}

type StatusState struct {
	state    string
	category int8
}

const CATEGORY_PRERUN = 0
const CATEGORY_WHILERUN = 1
const CATEGORY_POSTRUN = 2
const CATEGORY_END = 3

const STATUS_CREATED string = "created"
const STATUS_PREPARE string = "prepare"
const STATUS_DEPENDS_CHECKING string = "depends_checking"
const STATUS_DEPENDS_SOLVED string = "depends_solved"
const STATUS_READINESS_CHECKING string = "readiness_check"
const STATUS_READY string = "readiness_ready"
const STATUS_RUNNING string = "running"
const STATUS_DEAD string = "dead"
const STATUS_BACKOFF string = "backoff"

const STATUS_DEPENDS_FAILED string = "depends_failed"
const STATUS_READINESS_FAILED string = "readiness_failed"
const STATUS_INVALID_CONFIGURATION string = "invalid_configuration"

const STATUS_START string = "start"
const STATUS_KILL string = "kill"
const STATUS_PENDING_DELETE string = "pending_delete"
