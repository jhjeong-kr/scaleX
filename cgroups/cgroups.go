package cgroups

import (
	"bytes"
	"io/ioutil"
	"fmt"
	"path"
	"strconv"
	"strings"

	"cperfc/config"
	"cperfc/log"
)

type SubSystemManager struct {
	SubSystem	[]string
	Path		map[string]string
}

const cgroupPath = "/sys/fs/cgroup"
var subSystemManager SubSystemManager

func init() {
}

func Initialize() {
	log.Println("Initializing Cgroup subsystems.")
	GetSubSystemManager().init()
	log.Printf("Supported Cgroup subsystems are: %s.", GetSubSystemManager().GetAllSubSystems())
}

func GetSubSystemManager() *SubSystemManager {
	return &subSystemManager
}

func (self *SubSystemManager)init() {
	self.Path = make(map[string]string)
	cgroupPath := getCgroupPath()
	entries, _ := ioutil.ReadDir(cgroupPath)
	for _, file := range entries {
		if file.IsDir() {
			if (file.Name() == config.CpuSetSubSystem) {
				self.SubSystem =  append(self.SubSystem, file.Name())
				self.Path[file.Name()] = path.Join(cgroupPath, file.Name())
			}
		}
	}
}

func (self *SubSystemManager)GetSubSystemPath(subSystem string) (path string, ok bool) {
	path, ok = self.Path[subSystem]
	return path, ok
}

func (self *SubSystemManager)GetAllSubSystems() []string {
	return self.SubSystem
}

func getCgroupPath() string {
	return cgroupPath
}

func getCpuSetPath() string {
	return path.Join(getCgroupPath(), config.CpuSetSubSystem)
}

func getCpuSetPathOfContainerType(constainerType string) string {
	return path.Join(getCpuSetPath(), constainerType)
}

func getCpuSetPathOfContainer(containerType string, containerId string) string {
	return path.Join(getCpuSetPathOfContainerType(containerType), containerId)
}

func getCpuSetPathOfDockerContainer(containerId string) string {
	return getCpuSetPathOfContainer(config.DockerName, containerId)
}

func GetCoreInfoOfContainer(containerType string, containerId string, which string) (string, error) {
	fullpath := path.Join(getCpuSetPathOfContainer(containerType, containerId), which)
	b, err := ioutil.ReadFile(fullpath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func GetCoreInfoOfDockerContainer(containerId string) (string, error) {
	return GetCoreInfoOfContainer(config.DockerName, containerId, "cpuset.cpus")
}

func GetCoreInfoOfLxcContainer(containerId string) (string, error) {
	return GetCoreInfoOfContainer(config.LxcName, containerId, "cpuset.cpus")
}

func GetParentContainer(subSystem string, cid string) []string {
	var parents []string
	var walk func(string)

	subdSystemPath, exist := GetSubSystemManager().GetSubSystemPath(subSystem)
	if !exist {
		return parents
	}
	walk = func(parentPath string) {
		children, _ := ioutil.ReadDir(parentPath)
		for _, child := range children {
			if !child.IsDir() {
				continue
			}
			if child.Name() == cid {
				parents = append(parents, parentPath)
			} else {
				walk(path.Join(parentPath, child.Name()))
			}
		}
	}
	walk(subdSystemPath)
	return parents
}

func GetContainerFullPath(subSystem string, cid string) []string {
	var fullPath []string

	parents := GetParentContainer(subSystem, cid)
	for _, parent := range parents {
		fullPath = append(fullPath, path.Join(parent, cid))
	}
	return parents
}

func IsContainerExist(cid string) bool {
	fullPath := GetContainerFullPath(config.CpuSetSubSystem, cid)
	return len(fullPath) >= 1
}

func GetContainerType(cid string) string {
	fullPath := GetParentContainer(config.CpuSetSubSystem, cid)
	if len(fullPath) > 0 {
		_, file := path.Split(fullPath[0])
		return file
	}
	return ""
}

func ResetCgroupInfo(cid string) {
	for _, subSystem := range GetSubSystemManager().GetAllSubSystems() {
		resetCgroupInfo(subSystem, GetContainerFullPath(subSystem, cid)[0])
	}
}

func resetCgroupInfo(subSystem string, fullPath string) {
	switch subSystem {
	case config.CpuSetSubSystem:
	}
}

func DecodeListFormat(expression string) []int {
	var list []int

	ranges := strings.Split(expression, ",")
	for _, numbers := range ranges {
		if strings.Contains(numbers, "-") {
			fromTo := strings.Split(numbers, "-")
			start, _ := strconv.Atoi(fromTo[0])
			end, _ := strconv.Atoi(fromTo[1])
			for number := start; number <= end; number++ {
				list = append(list, int(number))
			}
		} else {
			number, _ := strconv.Atoi(numbers)
			list = append(list, int(number))
		}
	}
	return list
}

func EncodeListFormat(list []int) string {
	const impossible = -10
	var expression bytes.Buffer

	makeRange := func(start int, end int, del string) string {
		if start == end {
			return fmt.Sprintf("%d%s", start, del)
		} else {
			return fmt.Sprintf("%d-%d%s", start, end, del)
		}
	}
	start := list[0]
	end := start
	for _, number := range list {
		if end + 1 >= number {
			end = number
		} else {
			expression.WriteString(makeRange(start, end, ","))
			start = number
			end = start
		}
	}
	expression.WriteString(makeRange(start, end, ""))
	return expression.String()
}
