package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// StatusServer common info
type StatusServer struct {
	CPU  float64
	Disk float64
	Mem  float64
	Swap float64
}

var statusHealth int = 200

func hostInfo() StatusServer {
	v, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()
	cc, _ := cpu.Percent(time.Second, false)
	d, _ := disk.Usage("/")

	var ss StatusServer

	ss.Mem = v.UsedPercent
	ss.CPU = cc[0]
	ss.Swap = s.UsedPercent
	ss.Disk = d.UsedPercent

	return ss
}

// HardwareStatus -
func HardwareStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/hardware")
	w.Header().Set("Content-Type", "application/json")
	var data = hostInfo()
	json.NewEncoder(w).Encode(data)
}

// Health -
func Health(w http.ResponseWriter, r *http.Request) {
	switch statusHealth {
	case 200:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func upNode(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/up")
	statusHealth = 200
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Ok")
}

func downNode(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/down")
	statusHealth = 400
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Ok")
}

func main() {
	fmt.Printf("=== Start service on port 8888 ===\n")

	// =========== API ============

	r := mux.NewRouter()

	r.HandleFunc("/down", downNode).Methods("POST")
	r.HandleFunc("/up", upNode).Methods("POST")

	r.HandleFunc("/hardware", HardwareStatus).Methods("GET")
	r.HandleFunc("/", Health).Methods("GET")

	log.Fatal(http.ListenAndServe(":8888", r))
}
