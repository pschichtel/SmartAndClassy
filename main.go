package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"dario.cat/mergo"
	"net"
	"io/ioutil"
	"path/filepath"
	"github.com/akamensky/argparse"
	"strings"
)

type ClassTableEntry map[string]interface{}
type ClassTable map[string]ClassTableEntry
type DataTable map[string]interface{}

// production/*.yml
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

func loadComponent(name string, componentsBase string) (*Component, error) {
	nameComponents := strings.Split(strings.ToLower(name)+".yml", "/")
	data, err := ioutil.ReadFile(filepath.Join(componentsBase, filepath.Join(nameComponents...)))
	if err != nil {
		return nil, err
	}
	dst := &Component{}
	err = yaml.Unmarshal(data, dst)
	if err != nil {
		return nil, err
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

	return dst, nil
}

func resolveClasses(dst *ResolutionResult, implications []string, componentsBase string, strictMode bool, seen map[string]interface{}) {
	for i := range implications {
		implication := strings.ToLower(implications[i])
		_, seenBefore := seen[implication]
		if !seenBefore {
			seen[implication] = true
			fmt.Printf("# component: %s\n", implication)
			component, err := loadComponent(implication, componentsBase)
			if err == nil {
				// first merge the implications
				resolveClasses(dst, component.Implies, componentsBase, strictMode, seen)

				// then merge the current component (post-order) to prioritize explicitly configured values higher up the tree
				err = mergo.Merge(&dst.Classes, component.Classes, mergo.WithOverride)
				if err != nil {
					fmt.Printf("# Failed to merge classes!\n")
					if strictMode {
						panic("Failed to merge classes in strict mode!")
					}
				}
				err = mergo.Merge(&dst.Data, component.Data, mergo.WithOverride)
				if err != nil {
					fmt.Printf("# Failed to merge data!\n")
					if strictMode {
						panic("Failed to merge data in strict mode!")
					}
				}
				err = mergo.Merge(&dst.Parameters, component.Parameters, mergo.WithOverride)
				if err != nil {
					fmt.Printf("# Failed to merge parameters!\n")
					if strictMode {
						panic("Failed to merge parameters in strict mode!")
					}
				}
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

func classify(dst *Classification, nodeName string, nodesFile string, componentsBase string, strictMode bool) error {
	nodesData, err := ioutil.ReadFile(nodesFile)
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

	if node.Environment == "" {
		node.Environment = fallback.Environment
	}

	if strings.Contains(componentsBase, "%s") {
		componentsBase = fmt.Sprintf(componentsBase, node.Environment)
	}

	result := ResolutionResult{Data: DataTable{}, Classes: ClassTable{}}
	resolveClasses(&result, node.Implies, componentsBase, strictMode, map[string]interface{}{})
	dst.Classes = result.Classes
	dst.Data = result.Data
	dst.Parameters = result.Parameters
	dst.Environment = node.Environment

	return nil
}

func main() {

	parser := argparse.NewParser("classyfy", "A Puppet external node classifier (ENC)")

	componentsBase := parser.String("c", "production-base", &argparse.Options{Default: "production", Required: false, Help: "The base path for configuration production."})
	nodesFile := parser.String("N", "nodes-file", &argparse.Options{Default: "nodes.yml", Required: false, Help: "The path to the node specification file."})
	nodeName := parser.String("n", "node", &argparse.Options{Required: true, Help: "The hostname of the node to classify."})
	dataOnly := parser.Flag("d", "data", &argparse.Options{Help: "Only output the data."})
	strictMode := parser.Flag("s", "strict", &argparse.Options{Help: "Fail on a inconsistent model."})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	classification := Classification{}
	err = classify(&classification, *nodeName, *nodesFile, *componentsBase, *strictMode)
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
