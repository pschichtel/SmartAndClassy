package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"github.com/imdario/mergo"
	"net"
	"io/ioutil"
)

type ClassTable map[string]map[string]interface{}
type HieraData map[string]interface{}

// roles/*.yaml
type Component struct {
	Classes ClassTable
	Data HieraData
	Implies []string
}

type ResolutionResult struct {
	Classes ClassTable
	Data    HieraData
}

// nodes/*.yaml
type NodeSpec struct {
	Defaults Node
	Nodes map[string]Node
}

type Node struct {
	Environment string
	Implies []string
}

type Classification struct {
	Classes ClassTable
	Data HieraData
	Environment string
}

func loadComponent(name string, configPrefix string) (*Component, error) {
	data, err := ioutil.ReadFile(configPrefix + "/components/" + name + ".yml")
	if err != nil {
		return nil, err
	}
	component := Component{}
	yaml.Unmarshal(data, &component)
	return &component, nil
}

func resolveClasses(dst *ResolutionResult, implications[]string, confPrefix string, seen map[string]interface{}) {
	for i := range implications {
		implication := implications[i]
		_, seenBefore := seen[implication]
		if !seenBefore {
			seen[implication] = true
			component, err := loadComponent(implication, confPrefix)
			if err == nil {
				mergo.Merge(&dst.Classes, component.Classes)
				mergo.Merge(&dst.Data, component.Data)
				resolveClasses(dst, component.Implies, confPrefix, seen)
			}
		}
	}
}

func classify(node string, confPrefix string) (*Classification, error) {

	nodesData, err := ioutil.ReadFile(confPrefix + "/nodes.yml")
	if err != nil {
		return nil, err
	}

	nodes := NodeSpec{}
	yaml.Unmarshal(nodesData, &nodes)


	defaultNode := nodes.Defaults
	nodeSpec, found := nodes.Nodes[node]
	if !found {
		nodeSpec = defaultNode
	}

	result := ResolutionResult{}
	resolveClasses(&result, nodeSpec.Implies, confPrefix, map[string]interface{}{})
	classification := Classification{Classes: result.Classes, Data: result.Data, Environment: nodeSpec.Environment}
	if classification.Environment == "" {
		classification.Environment = defaultNode.Environment
	}

	ips, _ := net.LookupIP(node)
	stringIps := make([]string, len(ips))
	for i, ip := range ips {
		stringIps[i] = ip.String()
	}

	classification.Classes["cubyte::network"] = map[string]interface{}{"ips": stringIps}

	return &classification, nil
}

func main() {

	if len(os.Args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <node>\n", os.Args[0])
		os.Exit(1)
		return
	}

	class, _ := classify(os.Args[1], ".")
	response, _ := yaml.Marshal(class)

	fmt.Printf("---\n%s\n", string(response))
}
