package main

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"net"
)

type ClassTable map[string]map[string]interface{}

// roles/*.yaml
type Role struct {
	Name string
	Imports []string
	Classes ClassTable
}

// bundles/*.yaml
type Bundle struct {
	Name string
	Classes ClassTable
}

// groups/*.yaml
type Group struct {
	Name string
	Parents []string
	Classes ClassTable
	Environment string
}

// nodes/*.yaml
type Node struct {
	Name string
	Groups []string
	Roles []string
	Imports []string
	Classes ClassTable
	Environment string
}

// implications/*.yaml
type Implication struct {
	If string
	Then ClassTable
}

type Classification struct {
	Classes ClassTable
	Environment string
}

func classify(node string) (Classification, error) {
	classification := Classification{Classes: map[string]map[string]interface{} {"cubyte::welcome::hostname": {"hostname": node}}, Environment: "production"}

	ips, _ := net.LookupIP(node)
	stringIps := make([]string, len(ips))
	for i, ip := range ips {
		stringIps[i] = ip.String()
	}

	classification.Classes["cubyte::network"] = map[string]interface{}{"ips": stringIps}

	return classification, nil
}

func main() {

	if len(os.Args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <node>\n", os.Args[0])
		os.Exit(1)
		return
	}

	class, _ := classify(os.Args[1])
	response, _ := yaml.Marshal(class)

	fmt.Printf("---\n%s\n", string(response))
}
