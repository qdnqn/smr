package static

import (
	_ "embed"
)

const ROOTDIR string = "smr"
const CONFIGDIR string = "config"
const PROJECT = "smr"

const SMR string = "smr"

const DEFAULT_LOG_LEVEL = "info"

var STRUCTURE = []string{
	"config",
	"persistent",
	"persistent/smr",
}

const SMR_ENDPOINT_NAME = "smr-agent"

const SMR_AGENT_URL = "smr-agent.cluster.private:1443"
const SMR_AGENT_DOMAIN = "smr-agent.cluster.private"
const SMR_LOCAL_DOMAIN string = "cluster.private"

//go:embed resources/git/version
var SMR_VERSION string
