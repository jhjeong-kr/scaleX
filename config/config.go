package config

import (
	"flag"
	"os"
)

const (
	EXITNORMAL = 0
	EXITNONROOT = 1
	EXITPORT = 80
	EXITCADVISOR = 2
	EXITLOG = 3
)
const LOOPSKIPCOUNT = 5

const DockerName = "docker"
const LxcName = "lxc"
const CpuSetSubSystem = "cpuset"

var LogFormat = "text"
var LogLevel = "info"
var CAdvisorAddr = "http://localhost:8080"
var ListeningPort = 8088
var MainLoopInterval = 10

func init() {
}

func ParseCommandLine() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flag.StringVar(&LogFormat, "logformat", LogFormat, "log format = {text, json}")
	flag.StringVar(&LogLevel, "loglevel", LogLevel, "log level = {info, warning, fatal, error, panic, debug}")
	flag.IntVar(&MainLoopInterval, "interval", MainLoopInterval, "interval for monitoring in second")
	flag.IntVar(&ListeningPort, "port", ListeningPort, "port for RESTful API serving")
	flag.StringVar(&CAdvisorAddr, "cadvisor", CAdvisorAddr, "address to cAdvisor API server")
	flag.Parse()
}
