package cperfc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"strconv"

	"github.com/gorilla/mux"

	"cperfc/config"
	"cperfc/cgroups"
	"cperfc/log"
)

type SimpleResult struct {
	Result			bool			`json:"result"`
	Desc			string			`json:"description"`
}

func init() {
}

func RESTfulAPIServe() {
	log.Printf("Trying to initialize RESTful API port(%d).", config.ListeningPort)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", restfulIndex)
	router.HandleFunc("/control/{controlMsg}", restfulControl)
	router.HandleFunc("/api/process/getcontainer/{pid}", restfulProcessGetContainer)
	router.HandleFunc("/api/container/register/{cid}", restfulContainerRegister)
	router.HandleFunc("/api/container/unregister/{cid}", restfulContainerUnregister)
	router.HandleFunc("/api/container/isregistered/{cid}", restfulContainerIsRegistered)
	router.HandleFunc("/api/container/status/{cid}", restfulContainerStatus)
	router.HandleFunc("/api/container/set/cpu/{cid}", restfulContainerSetCPU)
	router.HandleFunc("/api/container/set/cpuset/{cid}", restfulContainerSetCPUSet)
	router.HandleFunc("/api/container/reset/cpu/{cid}", restfulContainerResetCPU)
	router.HandleFunc("/api/container/reset/cpuset/{cid}", restfulContainerResetCPUSet)
	log.Printf("APIs are ready.")
	go func() {
		err := http.ListenAndServe(":" + strconv.Itoa(config.ListeningPort), router)
		log.Fatal(err)
		finish(config.EXITPORT)
	}()
}

func restfulControl(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  controlMsg := vars["controlMsg"]
	log.Println("control requested:", controlMsg)
	switch strings.ToLower(controlMsg) {
	case "resume":
		StartMainLoop()
		fmt.Fprintln(w, "resumed")
	case "pause":
		StopMainLoop()
		fmt.Fprintln(w, "paused")
	case "exit":
		StopMainLoop()
		finish(config.EXITNORMAL)
		fmt.Fprintln(w, "exited")
	}
}

func restfulIndex(w http.ResponseWriter, r *http.Request) {
	log.Println("index")
}

func restfulProcessGetContainer(w http.ResponseWriter, r *http.Request) {
	var outBuffer bytes.Buffer
	var statusOK = false
	var container = Container{Id: "", Type: "", Path: ""}

	defer func() {
		text := outBuffer.String()
		prettyText := strings.Replace(text, "\n", "\n\t", -1)
		log.Println(prettyText)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if statusOK {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		json.NewEncoder(w).Encode(container)
	}()

	outBuffer.WriteString("Process API: getcontainer\n")
	vars := mux.Vars(r)
	_, err := strconv.ParseInt(vars["pid"], 10, 32)
	if err != nil {
		outBuffer.WriteString(fmt.Sprintf("The requested process ID is '%s'. Wrong ID", vars["pid"]))
		return
	}
	outBuffer.WriteString(fmt.Sprintf("The requested process ID is ''%s'\n", vars["pid"]))

	basePath := path.Join("/proc/", vars["pid"])
	b, err := ioutil.ReadFile(path.Join(basePath, "cmdline"))
	if err != nil {
		outBuffer.WriteString(fmt.Sprintf("The process does not exist"))
		return
	}

	_, filename := path.Split(string(b))
	outBuffer.WriteString(fmt.Sprintf("The process is '%s'\n", filename))

	file, err := os.Open(path.Join(basePath, "cgroup"))
	if err != nil {
		outBuffer.WriteString(fmt.Sprintln(err))
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		cpusetLine := scanner.Text()
		tokens := strings.Split(cpusetLine, "/")[1:]
		switch strings.ToLower(tokens[0]) {
		case "docker":
			outBuffer.WriteString("The process is of 'docker' container\n")
			outBuffer.WriteString(fmt.Sprintf("The container ID is '%s'\n", tokens[1]))
			container = Container{Id: tokens[1], Type: tokens[0], Path: path.Join("/", path.Join(tokens[0], tokens[1]))}
			statusOK = true
		case "":
			outBuffer.WriteString("The process is in a default container\n")
			container = Container{Id: tokens[1], Type: tokens[0], Path: path.Join("/", tokens[0])}
			statusOK = true
		}
	}
}

func restfulContainerRegister(w http.ResponseWriter, r *http.Request) {
	var outBuffer bytes.Buffer
	var result = SimpleResult{Result: false}
	var container = Container{Id: "", Type: "", Path: ""}

	defer func() {
		outBuffer.WriteString(result.Desc)
		outBuffer.WriteString("\n")
		text := outBuffer.String()
		prettyText := strings.Replace(text, "\n", "\n\t", -1)
		log.Println(prettyText)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}()

	outBuffer.WriteString("Process API: register\n")
	vars := mux.Vars(r)
	cid := vars["cid"]
	outBuffer.WriteString(fmt.Sprintf("The requested container ID is '%s'\n", cid))

	manager := GetContainerManager()
	if !cgroups.IsContainerExist(cid) {
		result.Desc = fmt.Sprintf("The container does not exist")
		return
	}
	if manager.IsContainerRegistered(cid) {
		result.Result = true
		result.Desc = fmt.Sprintf("The container is already registered")
		return
	}

	container.Id = cid
	container.Type = cgroups.GetContainerType(cid)
	container.Path = cgroups.GetContainerFullPath(config.CpuSetSubSystem, cid)[0]
	container.CAdvisorInfo, _ = GetContainerInfo(container)
	manager.AddContainer(&container)
	result.Result = true
	result.Desc = fmt.Sprintf("The container is registered")
}

func restfulContainerUnregister(w http.ResponseWriter, r *http.Request) {
	var outBuffer bytes.Buffer
	var result = SimpleResult{Result: false}

	defer func() {
		outBuffer.WriteString(result.Desc)
		outBuffer.WriteString("\n")
		text := outBuffer.String()
		prettyText := strings.Replace(text, "\n", "\n\t", -1)
		log.Println(prettyText)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}()

	manager := GetContainerManager()
	outBuffer.WriteString("Process API: unregister\n")
	vars := mux.Vars(r)
	cid := vars["cid"]
	outBuffer.WriteString(fmt.Sprintf("The requested container ID is %s'\n", cid))
	if manager.RemoveContainer(cid) {
		result.Desc = fmt.Sprintf("The container is unregistered")
	} else {
		result.Desc = fmt.Sprintf("Have not registered")
	}
	result.Result = true
}

func restfulContainerIsRegistered(w http.ResponseWriter, r *http.Request) {
	var outBuffer bytes.Buffer
	var result = SimpleResult{Result: false}

	defer func() {
		outBuffer.WriteString(result.Desc)
		outBuffer.WriteString("\n")
		text := outBuffer.String()
		prettyText := strings.Replace(text, "\n", "\n\t", -1)
		log.Println(prettyText)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}()

	outBuffer.WriteString("Process API: isregistered\n")
	vars := mux.Vars(r)
	cid := vars["cid"]
	outBuffer.WriteString(fmt.Sprintf("The requested container ID is '%s'\n", cid))

	manager := GetContainerManager()
	if manager.IsContainerRegistered(cid) {
		if !cgroups.IsContainerExist(cid) {
			result.Desc = fmt.Sprintf("The container is registered, but disappeared... clean up")
			manager.RemoveContainer(cid)
			return
		}
		result.Result = true
		result.Desc = fmt.Sprintf("The container is registered")
	} else {
		result.Desc = fmt.Sprintf("The container is not registered")
	}
}

func restfulContainerStatus(w http.ResponseWriter, r *http.Request) {
	var outBuffer bytes.Buffer
	var container = &Container{Id: "", Type: "", Path: ""}

	defer func() {
		if len(container.Id) > 0 {
			outBuffer.WriteString(JSONStructureToString(container))
			outBuffer.WriteString("\n")
		}
		text := outBuffer.String()
		prettyText := strings.Replace(text, "\n", "\n\t", -1)
		log.Println(prettyText)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(container)
	}()

	outBuffer.WriteString("Process API: status\n")
	vars := mux.Vars(r)
	cid := vars["cid"]
	outBuffer.WriteString(fmt.Sprintf("The requested container ID is '%s'\n", cid))

	manager := GetContainerManager()
	if manager.IsContainerRegistered(cid) {
		if !cgroups.IsContainerExist(cid) {
			outBuffer.WriteString(fmt.Sprintf("The container is registered, but disappeared... clean up\n"))
			manager.RemoveContainer(cid)
			return
		}
		container, _ = manager.GetContainers(cid)
	} else {
		outBuffer.WriteString(fmt.Sprintf("The container is not registered\n"))
	}
}

func restfulContainerSetCPU(w http.ResponseWriter, r *http.Request) {
}

func restfulContainerSetCPUSet(w http.ResponseWriter, r *http.Request) {
}

func restfulContainerResetCPU(w http.ResponseWriter, r *http.Request) {
	var outBuffer bytes.Buffer
	var registered = false

	defer func() {
		text := outBuffer.String()
		prettyText := strings.Replace(text, "\n", "\n\t", -1)
		log.Println(prettyText)

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(registered)
	}()

	outBuffer.WriteString("Process API: isregistered\n")
	vars := mux.Vars(r)
	cid := vars["cid"]
	outBuffer.WriteString(fmt.Sprintf("The requested container ID is '%s'\n", cid))

	manager := GetContainerManager()
	if manager.IsContainerRegistered(cid) {
		outBuffer.WriteString(fmt.Sprintf("The requested container is in the list\n"))
		if !cgroups.IsContainerExist(cid) {
			outBuffer.WriteString(fmt.Sprintf("Oops, Not exist. remove in the list\n"))
			manager.RemoveContainer(cid)
			return
		}
	} else {
		outBuffer.WriteString(fmt.Sprintf("The container is not registered\n"))
		return
	}
	registered = true
	cgroups.ResetCgroupInfo(cid)
}

func restfulContainerResetCPUSet(w http.ResponseWriter, r *http.Request) {
}
