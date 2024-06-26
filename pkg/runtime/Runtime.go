package runtime

import (
	"fmt"
	"github.com/qdnqn/smr/pkg/static"
	"github.com/qdnqn/smr/pkg/utils"
	"github.com/spf13/viper"
	"log"
	"net"
	"os"
)

func GetRuntimeInfo() *Runtime {
	HOMEDIR, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}

	OPTDIR := "/opt/smr"

	if viper.GetBool("optmode") {
		if _, err := os.Stat(OPTDIR); err != nil {
			if err = os.Mkdir(OPTDIR, os.FileMode(0750)); err != nil {
				panic(err.Error())
			}
		}
	}

	if viper.GetString("project") == "" {
		// TODO: Try to find context file and parse it from there
		// TODO: If context files is missing or invalid get the default
		viper.Set("project", static.ROOTDIR)
	}

	return &Runtime{
		HOMEDIR:    HOMEDIR,
		OPTDIR:     OPTDIR,
		PROJECT:    fmt.Sprintf("%s", viper.GetString("project")),
		PROJECTDIR: fmt.Sprintf("%s/%s/%s", HOMEDIR, static.ROOTDIR, viper.GetString("project")),
		PASSWORD:   utils.RandString(32),
		AGENTIP:    GetOutboundIP(),
	}
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
