package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	goModFile = "go.mod"
)

func red(str interface{}) string {
	return fmt.Sprintf("\033[91m%v\033[00m", str)
}

var report struct {
	rootNode *node
	// pkgname+ver -> node
	nodeMap map[string]*node

	// pkgname -> node array
	pkgnameMap map[string][]*node

	conflictPaths map[string][][]string
}

func init() {
	report.nodeMap = make(map[string]*node)
	report.pkgnameMap = make(map[string][]*node)

	report.conflictPaths = make(map[string][][]string)
}

// node represent a unique lib appeared in go mod graph
type node struct {
	pkgNameWithVersion string // package name with version
	parent             []*node
	children           []*node
	path               []string
}

func recordPath(n *node, parentPath []string) {
	if n == nil {
		return
	}

	curPath := make([]string, len(parentPath))
	copy(curPath, parentPath)

	curPath = append(curPath, n.pkgNameWithVersion)
	n.path = curPath

	for _, subNode := range n.children {
		recordPath(subNode, curPath)
	}
}

func recordPkg2Nodes(n *node) {
	if n == nil {
		return
	}
	nameAndVer := strings.Split(n.pkgNameWithVersion, "@")
	var name = nameAndVer[0]
	report.pkgnameMap[name] = append(report.pkgnameMap[name], n)

	for _, subNode := range n.children {
		recordPkg2Nodes(subNode)
	}
}

func whyPackage() {
	for pkg, _ := range report.conflictPaths {
		fmt.Println("why pkg", pkg, "? >>>")
		cmd := exec.Command("go", "mod", "why", pkg)
		out, err := cmd.Output()
		if err != nil {
			_ = fmt.Errorf("cannot execute go mod graph: %v", err)
		}
		var lines = strings.Split(string(out), "\n")
		for _, line := range lines {
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}
			pkgDep := strings.Split(line, " ")
			fmt.Printf("\t%s\n", pkgDep[0])
		}
	}
}

func findConflict() {
	conflictInSomePkg := false
	for pkgname, nodes := range report.pkgnameMap {
		if len(nodes) <= 1 {
			continue
		}

		// may have conflict
		var hasConflict = false
		for i := 1; i < len(nodes); i++ {
			if nodes[i].pkgNameWithVersion != nodes[i-1].pkgNameWithVersion {
				hasConflict = true
				conflictInSomePkg = true
				break
			}
		}

		if hasConflict {
			fmt.Println(red("Conflict"), "in pkg", pkgname, "paths are: ")
			for _, n := range nodes {
				fmt.Println("\r", strings.Join(n.path, " -> "))
				report.conflictPaths[pkgname] = append(report.conflictPaths[pkgname], n.path)
			}
		}
	}

	if !conflictInSomePkg {
		fmt.Println("there is no conflict in your project dependencies")
	}
}

func findConflicts() string {
	cmd := exec.Command("go", "mod", "graph")
	resultBytes, err := cmd.Output()
	if err != nil {
		_ = fmt.Errorf("cannot execute go mod graph: %v", err)
	}
	return string(resultBytes)
}

func listRequired() {
	data, err := os.ReadFile(goModFile)
	if err != nil {
		fmt.Println("Error reading go.mod:", err)
		return
	}
	modFile, err := modfile.Parse(goModFile, data, nil)
	if err != nil {
		fmt.Printf("Error parsing '%s': %v\n", goModFile, err)
		return
	}
	fmt.Println("Module Path:", modFile.Module.Mod.Path)
	for _, req := range modFile.Require {
		fmt.Println("Require:", req.Mod.Path, req.Mod.Version)
	}
}

func main() {
	goModLocationPtr := flag.String("go-mod-location", ".", "Path to go.mod file")
	flag.Parse()

	goModLocation := *goModLocationPtr
	fmt.Println("location for go.mod is", goModLocation)

	if goModLocation != "." {
		currentLocation, _ := os.Getwd()
		// jump back to current location
		defer func(dir string) {
			err := os.Chdir(dir)
			if err != nil {
				_ = fmt.Errorf("cannot change to project path: %v", err)
			}
		}(currentLocation)
		// check if directory exists
		err := os.Chdir(goModLocation)
		if err != nil {
			_ = fmt.Errorf("cannot change to project path: %v", err)
		}
	}
	// check if file exists
	goModLocation = fmt.Sprintf("%s/go.mod", goModLocation)
	if _, err := os.Stat(goModLocation); os.IsNotExist(err) {
		_ = fmt.Errorf("cannot find go.mod, \nPlease execute this tool under your %s \n\nand make sure there is a %s file", red("project path"), red("go.mod"))
	}

	listRequired()

	var lines = strings.Split(findConflicts(), "\n")
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		pkgDep := strings.Split(line, " ")
		if len(pkgDep) != 2 {
			fmt.Println("wrong format", pkgDep)
			return
		}

		// github.com/cch123@v1.0.1    github.com/google@v2.3.4
		pkg, dep := pkgDep[0], pkgDep[1]
		pkgNode := &node{pkgNameWithVersion: pkg}
		depNode := &node{pkgNameWithVersion: dep}
		if _, ok := report.nodeMap[pkg]; !ok {
			report.nodeMap[pkg] = pkgNode
		}

		report.nodeMap[pkg].children = append(report.nodeMap[pkg].children, depNode)

		if _, ok := report.nodeMap[dep]; !ok {
			report.nodeMap[dep] = depNode
		}
		report.nodeMap[dep].parent = append(report.nodeMap[dep].parent, pkgNode)

		if report.rootNode == nil {
			report.rootNode = pkgNode
		}
	}

	recordPath(report.rootNode, []string{})
	recordPkg2Nodes(report.rootNode)
	findConflict()
	whyPackage()
}
