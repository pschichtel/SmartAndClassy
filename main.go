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

// components/*.yml
type Component struct {
	Classes ClassTable
	Data HieraData
	Implies []string
}

type ResolutionResult struct {
	Classes ClassTable
	Data    HieraData
}

// nodes.yml
type NodeSpec struct {
	Fallback Node
	Nodes    map[string]Node
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

func loadComponent(dst *Component, name string, configPrefix string) error {
	data, err := ioutil.ReadFile(configPrefix + "/components/" + name + ".yml")
	if err != nil {
		return err
	}
	yaml.Unmarshal(data, dst)
	return nil
}

func resolveClasses(dst *ResolutionResult, implications[]string, confPrefix string, seen map[string]interface{}) {
	for i := range implications {
		implication := implications[i]
		_, seenBefore := seen[implication]
		if !seenBefore {
			seen[implication] = true
			component := Component{}
			err := loadComponent(&component, implication, confPrefix)
			if err == nil {
				mergo.Merge(&dst.Classes, component.Classes)
				mergo.Merge(&dst.Data, component.Data)
				resolveClasses(dst, component.Implies, confPrefix, seen)
			}
		}
	}
}

func classify(nodeName string, confPrefix string) (*Classification, error) {

	nodesData, err := ioutil.ReadFile(confPrefix + "/nodes.yml")
	if err != nil {
		return nil, err
	}

	nodes := NodeSpec{}
	yaml.Unmarshal(nodesData, &nodes)


	fallback := nodes.Fallback
	node, found := nodes.Nodes[nodeName]
	if !found {
		node = fallback
	}

	result := ResolutionResult{}
	resolveClasses(&result, node.Implies, confPrefix, map[string]interface{}{})
	classification := Classification{Classes: result.Classes, Data: result.Data, Environment: node.Environment}
	if classification.Environment == "" {
		classification.Environment = fallback.Environment
	}

	ips, _ := net.LookupIP(nodeName)
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
