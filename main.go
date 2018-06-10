package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"github.com/imdario/mergo"
	"net"
	"io/ioutil"
	"path/filepath"
	"github.com/akamensky/argparse"
	"strings"
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
	nameComponents := strings.Split(strings.ToLower(name) + ".yml", "/")
	data, err := ioutil.ReadFile(filepath.Join(confPrefix, "components", filepath.Join(nameComponents...)))
	if err != nil {
		return err
	}
	yaml.Unmarshal(data, dst)
	return nil
}

func resolveClasses(dst *ResolutionResult, implications []string, confPrefix string, seen map[string]interface{}) {
	for i := range implications {
		implication := strings.ToLower(implications[i])
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
	nodes := NodeSpec{}

	if err == nil {
		yaml.Unmarshal(nodesData, &nodes)
	} else {
		return err
	}

	fallback := nodes.Fallback
	node, found := nodes.Nodes[nodeName]
	if !found {
		node = fallback
	}

	result := ResolutionResult{Data:HieraData{}, Classes:ClassTable{}}
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

	parser := argparse.NewParser("classyfy", "A Puppet external node classifier (ENC)")

	confPrefix := parser.String("c", "conf-prefix", &argparse.Options{Default:".", Required:false, Help:"The base path for configuration"})
	nodeName := parser.String("n", "node", &argparse.Options{Required:true, Help:"The hostname of the node to classify"})
	dataOnly := parser.Flag("d", "data", &argparse.Options{Help:"Only output the data"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	classification := Classification{}
	err = classify(&classification, *nodeName, *confPrefix)
	if err != nil {
		fmt.Println("Failed to classify the given node!")
		fmt.Println(err)
		os.Exit(1)
	}

	var data interface{}
	if *dataOnly {
		data = classification.Data
	} else {
		data = classification
	}

	response, _ := yaml.Marshal(data)
	fmt.Printf("---\n%s\n", string(response))
}
