package commands

import (
	"fmt"
	"github.com/simplecontainer/smr/pkg/bootstrap"
	"github.com/simplecontainer/smr/pkg/logger"
	"github.com/simplecontainer/smr/pkg/manager"
	"github.com/simplecontainer/smr/pkg/startup"
	"github.com/simplecontainer/smr/pkg/static"
	"io"
	"os"
)

func Create() {
	Commands = append(Commands, Command{
		name: "create",
		condition: func(*manager.Manager) bool {
			if os.Args[2] == "" {
				logger.Log.Warn("please specify project name")
				return false
			} else {
				return true
			}
		},
		functions: []func(*manager.Manager, []string){
			func(mgr *manager.Manager, args []string) {
				_, err := bootstrap.CreateProject(os.Args[2], mgr.Config)

				if err != nil {
					panic(err)
				}

				var out io.Writer
				out, err = os.OpenFile(fmt.Sprintf("%s/%s/%s/%s/config.yaml", mgr.Config.Environment.HOMEDIR, static.SMR, os.Args[2], static.CONFIGDIR), (os.O_WRONLY | os.O_CREATE), 0644)

				if err != nil {
					panic(err)
				}

				target := "development"
				if os.Getenv("ENVIRONMENT") != "" {
					target = os.Getenv("ENVIRONMENT")
				}

				domain := "localhost"
				if os.Getenv("DOMAIN") != "" {
					domain = os.Getenv("DOMAIN")
				}

				externalIP := "127.0.0.1"
				if os.Getenv("EXTERNALIP") != "" {
					externalIP = os.Getenv("EXTERNALIP")
				}

				mgr.Config.Target = target
				mgr.Config.Root = mgr.Config.Environment.PROJECTDIR
				mgr.Config.Domain = domain
				mgr.Config.ExternalIP = externalIP

				err = startup.Save(mgr.Config, out)

				if err != nil {
					panic(err)
				}
			},
		},
		depends_on: []func(*manager.Manager, []string){
			func(mgr *manager.Manager, args []string) {},
		},
	})
}
