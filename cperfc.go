package cperfc

import (
    "os"
	"os/user"
    "time"

	"cperfc/config"
	"cperfc/cgroups"
	log "cperfc/log"
)

func init() {
}

func Run() int {
    log.Logging()

	if u, _ := user.Current(); u.Gid != "0" {
		log.Error("Please run with root permissions")
        log.Info("Terminates.")
		finish(config.EXITNONROOT)
	}

	cgroups.Initialize()
	NewContainerManager()
	RESTfulAPIServe()
	StartMonitoring()

    for {
		time.Sleep(time.Second)
	}
    log.Info("Terminates.")
	return config.EXITNORMAL
}

func finish(returnCode int) {
	os.Exit(returnCode)
}
