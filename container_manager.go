package cperfc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"os"
	"time"

	cAdvisorInfo "github.com/google/cadvisor/info/v1"

	"cperfc/log"
)

type CgroupCPU struct {
	Shares			string			`json:"shares"`			// format: refer to 'cgroup' man page
	ThreshMin		int				`json:"thresh_min"`
	ThreshMax		int				`json:"thresh_max"`
	Cooltime		time.Time		`json:"cooltime"`
}

type CgroupCPUSet struct {
	CPUS			string			`json:"cpus"`			// format: refer to 'cgroup' man page
	ThreshMin		int				`json:"thresh_min"`
	ThreshMax		int				`json:"thresh_max"`
	MinCores		int				`json:"min_cores"`
	MaxCores		int				`json:"max_cores"`
	Cooltime		time.Time		`json:"cooltime"`
}

type CgroupInfo struct {
	CPUSet			CgroupCPUSet	`json:"cpuset"`
	CPU				CgroupCPU		`json:"cpu"`
}

type Container struct {
    Id      		string			`json:"id"`
	Type			string			`json:"type"`
	Path			string			`json:"path"`
	CgroupCurrent	CgroupInfo		`json:"cgroup_cur"`
	CgroupRequest	CgroupInfo		`json:"cgroup_req"`
	CAdvisorInfo	cAdvisorInfo.ContainerInfo		`json:"cAdvisor"`
	CPUUsageShort	float64				`json:"cpu_usage_short"`
	CPUUsageLong	float64				`json:"cpu_usage_long"`
	Timestamp		time.Time		`json:"Timestamp"`
}

type ContainerManager struct {
	Containers		map[string]*Container
}

var containerManager ContainerManager

func init() {
}

func NewContainerManager() {
	var outBuffer bytes.Buffer

	log.Println("Initializing registered containers.")
	manager := GetContainerManager()
	ok, msg := manager.load()
	if ok {
		outBuffer.WriteString(fmt.Sprintf("%d containers are under control.", len(manager.GetAllContainers())))
		for _, container := range manager.GetAllContainers() {
			outBuffer.WriteString(fmt.Sprintf("\n\t/%s", path.Join(container.Type, container.Id)))
		}
		log.Info(outBuffer.String())
	} else {
		outBuffer.WriteString(msg)
		log.Warn(outBuffer.String())
	}
}

func GetContainerManager() *ContainerManager {
	return &containerManager
}

func (self *ContainerManager)load() (ret bool, msg string) {
	const metaName = "registered"
	Exists := func (name string) bool {
    	_, err := os.Stat(name)
    	return !os.IsNotExist(err)
	}

	self.Containers = make(map[string]*Container)
	file, err := os.Open(metaName)
	if err != nil {
		if Exists(metaName) {
			return false, "Failed to load the previous container metadata"
		} else {
			return true, "No registered containers"
		}
	}
	defer file.Close()
	json.NewDecoder(file).Decode(&self.Containers)
	return true, ""
}

func (self *ContainerManager)store() (ret bool, msg string) {
	file, err := os.Create("registered")
	if err != nil {
		return false, "Failed to save container metadata"
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	encoder.Encode(self.Containers)
	return true, ""
}

func (self *ContainerManager)GetAllContainers() map[string]*Container {
	return self.Containers
}

func (self *ContainerManager)GetContainers(cid string) (*Container, bool) {
	container, exist := self.Containers[cid]
	return container, exist
}

func (self *ContainerManager)AddContainer(container *Container) bool {
	if self.IsContainerRegistered(container.Id) {
		return false
	}
	copied := *container
	self.Containers[container.Id] = &copied
	self.store()
	return true
}

func (self *ContainerManager)RemoveContainer(id string) bool {
	if !self.IsContainerRegistered(id) {
		return false
	}
	delete(self.Containers, id)
	self.store()
	return true
}

func (self *ContainerManager)IsContainerRegistered(id string) bool {
	_, exist := self.Containers[id]
	return exist
}
