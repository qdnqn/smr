package configuration

type Configuration struct {
	Target      string       `default:"development" yaml:"target"`
	Root        string       `yaml:"root"`
	Domain      string       `yaml:"domain"`
	ExternalIP  string       `yaml:"externalIP"`
	Environment *Environment `yaml:"-"`
	Flags       Flags        `yaml:"-"`
}

type Flags struct {
	Daemon        bool
	DaemonSecured bool
	DaemonDomain  string
	Verbose       bool
	OptMode       bool
}

type Environment struct {
	HOMEDIR    string
	OPTDIR     string
	PROJECTDIR string
	PROJECT    string
	AGENTIP    string
}
