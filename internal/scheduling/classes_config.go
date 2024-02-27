package scheduling

import (
	"encoding/json"
	"github.com/grussorusso/serverledge/internal/function"
	"io"
	"log"
	"os"
)

var Classes = make(map[string]function.QoSClass)

var DefaultClass = function.QoSClass{
	Name:                "default",
	Utility:             1,
	MaximumResponseTime: -1,
	CompletedPercentage: 0,
}

// AddDefaultClass Add default class to the classes list
func addDefaultClass() {
	Classes["default"] = DefaultClass
}

var DefaultClassesConfigFileName = "serverledge-classes.json"

// ReadClassesConfiguration reads a configuration file stored in one of the predefined paths which contains classes configuration.
func ReadClassesConfiguration(fileName string) {
	// paths where the config file can be placed
	var paths = []string{"/etc/serverledge/", "$HOME/", "./"}
	var jsonFile *os.File
	var err error
	var name string

	if fileName == "" {
		name = DefaultClassesConfigFileName
	} else {
		name = fileName
	}

	for i := range paths {
		x := paths[i] + name
		jsonFile, err = os.Open(x)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Println(err)
		log.Println("Classes configuration file not found, using default.")
		addDefaultClass()
		return
	}

	byteValue, _ := io.ReadAll(jsonFile)
	defer jsonFile.Close()

	var list []function.QoSClass
	err = json.Unmarshal(byteValue, &list)
	if err != nil {
		log.Println("Failed to parse classes, using default")
		addDefaultClass()
		return
	}

	for c := range list {
		Classes[list[c].Name] = list[c]
	}

	if _, prs := Classes["default"]; !prs {
		log.Println("Adding default class")
		addDefaultClass()
	}

	log.Printf("Found %d classes available\n", len(Classes))
}
