package cperfc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	cAdvisorClient "github.com/google/cadvisor/client"
	cAdvisorInfo "github.com/google/cadvisor/info/v1"

	"cperfc/config"
	"cperfc/cgroups"
	"cperfc/log"
)

const numberOfRequest = 64

var cAdvisor *cAdvisorClient.Client
var loopController chan bool

func init() {
}

func StartMonitoring() {
	var err error

	log.Info("Initializing container monitoring tools.")
	cAdvisor, err = cAdvisorClient.NewClient(config.CAdvisorAddr)
	if err != nil {
		log.Error(err)
		log.Info("Terminates")
		finish(config.EXITCADVISOR)
	}
	_, err = cAdvisor.MachineInfo()
	if err != nil {
		log.Errorf("Failed to connect cAdvisor[%s].", config.CAdvisorAddr)
		log.Info("Terminates")
		finish(config.EXITCADVISOR)
	}
	loopController = make(chan bool)
	log.Infof("cAdvisor[%s] is running.", config.CAdvisorAddr)
	log.Info("Starting monitoring.")
	StartMainLoop()
}

func StartMainLoop() {
	close(loopController)
	loopController = make(chan bool)
	go mainLooper()
}

func mainLooper() {
	ticker := time.NewTicker(time.Duration(config.MainLoopInterval) * time.Second)
	defer ticker.Stop()
	loopSkipCount := 0
	for {
		select {
		case <- ticker.C:
			loopSkipCount = monitoring(loopSkipCount)
		case <- loopController:
			return
		}
	}
}

func StopMainLoop() {
	close(loopController)
	loopController = make(chan bool)
}

func monitoring(loopSkipCount int) int {
	var outBuffer bytes.Buffer

	defer func() {
		if len(outBuffer.String()) > 0 {
			log.Debug(strings.Replace(outBuffer.String(), "\n", "\n\t", -1))
		}
	}()
	manager := GetContainerManager()
	allRegisteredContainers := manager.GetAllContainers()
	outBuffer.WriteString(fmt.Sprintf("Container(%d) monitoring.", len(allRegisteredContainers)))
	if loopSkipCount > 0 {
		log.Warnf("cooldown for cAdvisor(%d)", loopSkipCount)
		return loopSkipCount - 1
	}
	machine, err := cAdvisor.MachineInfo()
	if err != nil {
		log.Error(fmt.Sprint(err))
		return config.LOOPSKIPCOUNT
	}
	machineCores := machine.NumCores
	freq := machine.CpuFrequency

	if len(allRegisteredContainers) == 0 {
		outBuffer.Reset()
	}
	request := cAdvisorInfo.ContainerInfoRequest{NumStats: numberOfRequest}
	for _, registeredContainer := range allRegisteredContainers {
		container, err := cAdvisor.ContainerInfo(path.Join("/", path.Join(registeredContainer.Type, registeredContainer.Id)), &request)
		if err != nil {
			outBuffer.WriteString(fmt.Sprintln(err))
			return config.LOOPSKIPCOUNT
		}
		ratio, duration, timestamp, err := CalcCPUUsage(container, false)
		if err == nil {
			registeredContainer.Timestamp = timestamp
			registeredContainer.CPUUsageLong = ratio
			actualCores := len(cgroups.DecodeListFormat(container.Spec.Cpu.Mask))
			if container.Namespace == config.DockerName {
				outBuffer.WriteString(fmt.Sprintf("\n%s(%s)", container.Name, container.Spec.Image))
			} else {
				outBuffer.WriteString(fmt.Sprintf("\n%s", container.Name))
			}
			outBuffer.WriteString(fmt.Sprintf("\n\t%.4f%% of %d(%s)/%d(0-%d) cores at %.2fGHz for %d seconds", ratio, actualCores, container.Spec.Cpu.Mask, machineCores, machineCores - 1, float64(freq) / 1000000, duration))
			ratio, duration, _, _ = CalcCPUUsage(container, true)
			registeredContainer.CPUUsageShort = ratio
			outBuffer.WriteString(fmt.Sprintf("\n\t%.4f%% of %d(%s)/%d(0-%d) cores at %.2fGHz for %d seconds", ratio, actualCores, container.Spec.Cpu.Mask, machineCores, machineCores - 1, float64(freq) / 1000000, duration))
			registeredContainer.CPUUsageLong = ratio
		}
	}
	return 0
}

func CalcCPUUsage(container *cAdvisorInfo.ContainerInfo, justNow bool) (ratio float64, duration int, timestamp time.Time, err error) {
	if len(container.Stats) >= 2 {
		var prevEvents *cAdvisorInfo.ContainerStats

		if justNow {
			prevEvents = container.Stats[len(container.Stats) - 2]
		} else {
			prevEvents = container.Stats[0]
		}
		currEvents := container.Stats[len(container.Stats) - 1]
		timeDelta := currEvents.Timestamp.Sub(prevEvents.Timestamp).Nanoseconds()
		usageDelta := currEvents.Cpu.Usage.Total - prevEvents.Cpu.Usage.Total
		actualCores := len(cgroups.DecodeListFormat(container.Spec.Cpu.Mask))
		usageRatio := float64(usageDelta) / (float64(timeDelta) * float64(actualCores))
		return usageRatio * 100, int(timeDelta / 1000 / 1000 / 1000), currEvents.Timestamp, nil
	}
	return 0, 0, time.Now(), errors.New("Not enough 'Stats'")
}

func GetContainerInfo(container Container) (cAdvisorInfo.ContainerInfo, error) {
	var info cAdvisorInfo.ContainerInfo

	request := cAdvisorInfo.ContainerInfoRequest{NumStats: 1}
	org, err := cAdvisor.ContainerInfo(path.Join("/", path.Join(container.Type, container.Id)), &request)
	info = *org
	info.Stats = nil
	return info, err
}

func JSONStructureToString(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func dumpJSONStructure(v interface{}) {
	bytes, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return
	}
	log.Println(string(bytes))
}
