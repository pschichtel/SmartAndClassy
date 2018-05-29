package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"github.com/imdario/mergo"
	"net"
	"io/ioutil"
	"path/filepath"
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

func loadComponent(dst *Component, name string, confPrefix string) error {
	data, err := ioutil.ReadFile(filepath.Join(confPrefix, "components", name + ".yml"))
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

func resolveNodeName(nodeName string) {
	ips, _ := net.LookupIP(nodeName)
	stringIps := make([]string, len(ips))
	for i, ip := range ips {
		stringIps[i] = ip.String()
	}
}

func classify(dst *Classification, nodeName string, confPrefix string) error {
	nodesData, err := ioutil.ReadFile(filepath.Join(confPrefix, "nodes.yml"))
	if err != nil {
		return err
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
	dst.Classes = result.Classes
	dst.Data = result.Data
	dst.Environment = node.Environment
	if dst.Environment == "" {
		dst.Environment = fallback.Environment
	}

	return nil
}

func main() {

	if len(os.Args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <node>\n", os.Args[0])
		os.Exit(1)
		return
	}

	classification := Classification{}
	_ = classify(&classification, os.Args[1], ".")
	response, _ := yaml.Marshal(classification)

	fmt.Printf("---\n%s\n", string(response))
}
