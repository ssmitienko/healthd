package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-systemd/v22/dbus"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type healthCheckReq struct {
	serviceIsRunning arrayFlags
	fileExists       arrayFlags
	fileDontExists   arrayFlags
	httpGet          arrayFlags
	erigonSyncing    arrayFlags
}

func checkFileExists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	return false
}

func checkServiceIsRunning(name string) bool {
	conn, err := dbus.New()
	defer conn.Close()
	if err != nil {
		return false
	}
	units, err := conn.ListUnits()
	if err != nil {
		return false
	}
	for _, unit := range units {
		if unit.Name == name+".service" {
			return unit.ActiveState == "active"
		}
	}
	return false
}

func checkHttp(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	//fmt.Println(url, resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	return true
}

func checkErigon(url string) bool {

	resp, err := http.Post(url, "application/json",
		bytes.NewBuffer([]byte("{\"method\":\"eth_syncing\",\"params\":[],\"id\":1,\"jsonrpc\":\"2.0\"}")))
	if err != nil {
		return false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	//fmt.Println("res:", res)

	if val, ok := res["result"]; ok {
		switch val.(type) {
		case bool:
			return val.(bool) == false
		case *bool:
			if val == nil {
				return false == false
			}
			return *val.(*bool)
		case string:
			return val.(string) == "false"
		case *string:
			return *val.(*string) == "false"
		}
	}
	return false
}

func doHealthCheckFiles(h *healthCheckReq) bool {
	for i := range h.fileExists {
		if checkFileExists(h.fileExists[i]) == false {
			return false
		}
	}
	for i := range h.fileDontExists {
		if checkFileExists(h.fileDontExists[i]) == true {
			return false
		}
	}
	return true
}

func doHealthCheckServices(h *healthCheckReq) bool {
	for i := range h.serviceIsRunning {
		if checkServiceIsRunning(h.serviceIsRunning[i]) == false {
			return false
		}
	}
	return true
}

func doHealthCheckHttp(h *healthCheckReq) bool {
	for i := range h.httpGet {
		if checkHttp(h.httpGet[i]) == false {
			return false
		}
	}
	return true
}

func doHealthCheckErigon(h *healthCheckReq) bool {
	for i := range h.erigonSyncing {
		if checkErigon(h.erigonSyncing[i]) == false {
			return false
		}
	}
	return true
}

func doHealthCheck(h *healthCheckReq) bool {
	if doHealthCheckFiles(h) == false {
		return false
	}
	if doHealthCheckServices(h) == false {
		return false
	}
	if doHealthCheckHttp(h) == false {
		return false
	}
	if doHealthCheckErigon(h) == false {
		return false
	}
	return true
}

var h healthCheckReq

func httpHandler(w http.ResponseWriter, r *http.Request) {
	res := doHealthCheck(&h)
	if res {
		fmt.Fprintf(w, "Health check OK")
		return
	} else {
		http.Error(w, "Health check failed", http.StatusInternalServerError)
	}
}

func main() {
	var runOnce bool
	flag.Var(&h.serviceIsRunning, "service", "List of systemd services required to run.")
	flag.Var(&h.fileExists, "fileexists", "List of files required to exist.")
	flag.Var(&h.fileDontExists, "filedontexists", "List of files indicating an error.")
	flag.Var(&h.httpGet, "httpget", "HTTP GET request url.")
	flag.Var(&h.erigonSyncing, "erigon", "check Erigon syncing state")
	flag.BoolVar(&runOnce, "runonce", false, "Run checks and output results.")
	listen := flag.String("listen", "0.0.0.0:8202", "Socket address to listen")
	flag.Parse()
	if runOnce {
		res := doHealthCheck(&h)
		fmt.Println("Check result:", res)
		if res == false {
			os.Exit(1)
		}
		return
	}
	http.HandleFunc("/", httpHandler)
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}
