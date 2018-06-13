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

type ClassTableEntry map[string]interface{}
type ClassTable map[string]ClassTableEntry
type DataTable map[string]interface{}

// components/*.yml
type Component struct {
	Classes    ClassTable
	Data       DataTable
	Parameters DataTable
	Implies    []string
}

type ResolutionResult struct {
	Classes    ClassTable
	Data       DataTable
	Parameters DataTable
}

// nodes.yml
type NodeSpec struct {
	Fallback Node
	Nodes    map[string]Node
}

type Node struct {
	Environment string
	Implies     []string
}

type Classification struct {
	Classes     ClassTable
	Data        DataTable
	Parameters  DataTable
	Environment string
}

func loadComponent(dst *Component, name string, confPrefix string) error {
	nameComponents := strings.Split(strings.ToLower(name)+".yml", "/")
	data, err := ioutil.ReadFile(filepath.Join(confPrefix, "components", filepath.Join(nameComponents...)))
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, dst)
	if err != nil {
		return err
	}
	if dst.Classes == nil {
		dst.Classes = ClassTable{}
	}
	// for ease of configuration allow to define the classes with without a value, we'll fix them to empty objects.
	for i := range dst.Classes {
		if dst.Classes[i] == nil {
			dst.Classes[i] = ClassTableEntry{}
		}
	}

	if dst.Data == nil {
		dst.Data = DataTable{}
	}

	if dst.Parameters == nil {
		dst.Parameters = DataTable{}
	}

	return nil
}

func resolveClasses(dst *ResolutionResult, implications []string, confPrefix string, seen map[string]interface{}, strictMode bool) {
	for i := range implications {
		implication := strings.ToLower(implications[i])
		_, seenBefore := seen[implication]
		if !seenBefore {
			seen[implication] = true
			component := Component{}
			err := loadComponent(&component, implication, confPrefix)
			if err == nil {
				err = mergo.Merge(&dst.Classes, component.Classes, mergo.WithOverride)
				if err != nil {
					if strictMode {
						panic("Failed to merge classes in strict mode!")
					}
					fmt.Printf("# Failed to merge classes!\n")
				}
				err = mergo.Merge(&dst.Data, component.Data, mergo.WithOverride)
				if err != nil {
					if strictMode {
						panic("Failed to merge data in strict mode!")
					}
					fmt.Printf("# Failed to merge data!\n")
				}
				err = mergo.Merge(&dst.Parameters, component.Parameters, mergo.WithOverride)
				if err != nil {
					if strictMode {
						panic("Failed to merge parameters in strict mode!")
					}
					fmt.Printf("# Failed to merge parameters!\n")
				}
				resolveClasses(dst, component.Implies, confPrefix, seen, strictMode)
			} else {
				if strictMode {
					panic("Failed to load a component in strict mode!")
				}
				fmt.Printf("# Error loading component %s: %s\n", implication, err)
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

func classify(dst *Classification, nodeName string, confPrefix string, strictMode bool) error {
	nodesData, err := ioutil.ReadFile(filepath.Join(confPrefix, "nodes.yml"))
	nodes := NodeSpec{}

	if err == nil {
		err = yaml.Unmarshal(nodesData, &nodes)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	fallback := nodes.Fallback
	node, found := nodes.Nodes[nodeName]
	if !found {
		if strictMode {
			panic("Node not found in strict mode!")
		}
		fmt.Println("# Node is not known, falling back!")
		node = fallback
	}

	result := ResolutionResult{Data: DataTable{}, Classes: ClassTable{}}
	resolveClasses(&result, node.Implies, confPrefix, map[string]interface{}{}, strictMode)
	dst.Classes = result.Classes
	dst.Data = result.Data
	dst.Parameters = result.Parameters
	dst.Environment = node.Environment
	if dst.Environment == "" {
		dst.Environment = fallback.Environment
	}

	return nil
}

func main() {

	parser := argparse.NewParser("classyfy", "A Puppet external node classifier (ENC)")

	confPrefix := parser.String("c", "conf-prefix", &argparse.Options{Default: ".", Required: false, Help: "The base path for configuration."})
	nodeName := parser.String("n", "node", &argparse.Options{Required: true, Help: "The hostname of the node to classify."})
	dataOnly := parser.Flag("d", "data", &argparse.Options{Help: "Only output the data."})
	strictMode := parser.Flag("s", "strict", &argparse.Options{Help: "Fail on a inconsistent model."})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	classification := Classification{}
	err = classify(&classification, *nodeName, *confPrefix, *strictMode)
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
