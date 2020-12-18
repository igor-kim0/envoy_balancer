package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config -
type Config struct {
	EnvoyConfig string `yaml:"config_envoy"`
	Port string `yaml:"config_port"`
	PathHardware string `yaml:"config_path_hardware"`
	PathHealth string `yaml:"config_path_health"`
}

type hardwareNode struct {
	CPU  float64
	Disk float64
	Mem  float64
	Swap float64
}

type stateNode struct {
	Weight    int
	Health    int
	Hardware  int
	newWeight int
}

var config Config

// ============================

func iterateNode(node *yaml.Node, identifier string) *yaml.Node {
	returnNode := false
	for _, n := range node.Content {
		if n.Value == identifier {
			returnNode = true
			continue
		}
		if returnNode {
			return n
		}
		if len(n.Content) > 0 {
			acNode := iterateNode(n, identifier)
			if acNode != nil {
				return acNode
			}
		}
	}
	return nil
}

func readConfigEnvoy() {
	filename, _ := filepath.Abs("./config.yaml")
	yamlFile, err := ioutil.ReadFile(filename)

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}
}

// CheckHealth -
func CheckHealth(address string) int {

	fmt.Print("Check health http://" + address + ":" + config.Port + config.PathHealth)

	resp, err := http.Get("http://" + address + ":" + config.Port + config.PathHealth)
	if err != nil {
		fmt.Println(" -  error")
		return 0
	}

	fmt.Println(" - ", resp.StatusCode)
	if resp.StatusCode == 200 {
		return 1
	}

	return 0
}

// CheckHardware -
func CheckHardware(address string) int {

	var Node hardwareNode

	resp, err := http.Get("http://" + address + ":" + config.Port + config.PathHardware)
	if err != nil {
		return 0
	}
	if resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		json.Unmarshal(body, &Node)

		// cpu load ?
		return int(Node.CPU)
	}

	return 0
}

// CalcClusterWeight -
func CalcClusterWeight(filename string) {

	fmt.Println(" calc cluster " + filename)

	stateNodes := make(map[string]stateNode)
	yamlFile, err := ioutil.ReadFile(filename)
	Cluster := yaml.Node{}
	err = yaml.Unmarshal(yamlFile, &Cluster)
	if err != nil {
		panic(err)
	}

	totalClusterLoad := 0
	totalClusterWeight := 0

	lbEndpointsNode := iterateNode(&Cluster, "lb_endpoints")
	for _, endpoint := range lbEndpointsNode.Content {

		Endpoint := iterateNode(endpoint, "endpoint")
		Address := iterateNode(Endpoint, "address")
		SocketAddress := iterateNode(Address, "address")
		Weight := iterateNode(endpoint, "load_balancing_weight")

		health := CheckHealth(SocketAddress.Value)
		weight, _ := strconv.ParseInt(Weight.Value, 10, 32)
		hardware := 0

		if health == 1 {
			hardware = CheckHardware(SocketAddress.Value)
			totalClusterLoad += hardware
			totalClusterWeight += int(weight)
			stateNodes[SocketAddress.Value] = stateNode{int(weight), health, hardware, int(weight)}
		}
	}

	if len(stateNodes) > 1 {
		for adr := range stateNodes {

			delta := float64(totalClusterLoad/len(stateNodes) - stateNodes[adr].Hardware) 
			delta = delta * 1.3
			weight := int(float64(stateNodes[adr].Weight) + delta)

			if weight < 1 { weight = 1 }
			if weight > 100 { weight = 100 }

			stateNodes[adr] = stateNode{stateNodes[adr].Weight, stateNodes[adr].Health, stateNodes[adr].Hardware, weight}
			fmt.Println(adr, stateNodes[adr].Hardware, " [", stateNodes[adr].Weight, "=>",  stateNodes[adr].newWeight, "]", delta)
		}

		lbEndpointsNode := iterateNode(&Cluster, "lb_endpoints")
		for _, endpoint := range lbEndpointsNode.Content {

			Endpoint := iterateNode(endpoint, "endpoint")
			Address := iterateNode(Endpoint, "address")
			SocketAddress := iterateNode(Address, "address")
			Weight := iterateNode(endpoint, "load_balancing_weight")

			if stateNodes[SocketAddress.Value].Health != 0 {
				Weight.Value = strconv.Itoa(stateNodes[SocketAddress.Value].newWeight)
			}
		}

		b, err := yaml.Marshal(&Cluster)
		err = ioutil.WriteFile(filename, b, 0644)
		if err != nil {
		}
	}
}

// ============================

func main() {
	readConfigEnvoy()

	filename, _ := filepath.Abs(config.EnvoyConfig)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	Envoy := yaml.Node{}

	err = yaml.Unmarshal(yamlFile, &Envoy)
	if err != nil {
		panic(err)
	}

	resourcesNode := iterateNode(&Envoy, "resources")
	for _, cluster := range resourcesNode.Content {
		edsClusterConfig := iterateNode(cluster, "eds_cluster_config")
		edsConfig := iterateNode(edsClusterConfig, "eds_config")
		Path := iterateNode(edsConfig, "path")

		CalcClusterWeight(Path.Value)
	}
}
