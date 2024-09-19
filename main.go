package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

//go:embed presets.json
var presetsBytes []byte

type Presets struct {
	Shorts map[string]string `json:"shorts"`

	Selects map[string]string `json:"selects"`

	Filters map[string]string `json:"filters"`
}

var presets Presets

func assert(err error, message string) {
	if err != nil {
		fmt.Printf("%s: %v\n", message, err)
		os.Exit(1)
	}
}

func assertCondition(condition bool, message string) {
	if !condition {
		fmt.Printf("%s\n", message)
		os.Exit(1)
	}
}

func translateWhere(where string, presets Presets) string {
	openBraceIdx := strings.Index(where, "(")
	closeBraceIdx := strings.LastIndex(where, ")")

	if openBraceIdx == -1 || closeBraceIdx == -1 {
		fmt.Printf("Invalid query: %s\n", where)
		os.Exit(1)
	}

	funcName := where[:openBraceIdx]
	argsString := where[openBraceIdx+1 : closeBraceIdx]

	query, ok := presets.Filters[funcName]
	if !ok {
		fmt.Printf("Unknown function: %s\n", funcName)
		os.Exit(1)
	}

	args := strings.Split(argsString, ",")
	for i, arg := range args {
		args[i] = strings.Trim(arg, " \t\n")
	}

	for i, arg := range args {
		query = strings.Replace(query, fmt.Sprintf("$%d", i+1), arg, 1)
	}

	for k, v := range presets.Shorts {
		query = strings.Replace(query, fmt.Sprintf("$%s", k), v, 1)
	}

	return query
}

func translateSelect(selects string, presets Presets) string {
	s := presets.Selects[selects]

	for k, v := range presets.Shorts {
		s = strings.Replace(s, fmt.Sprintf("$%s", k), v, 1)
	}

	return s
}

func translateQuery(selects string, where string, presets Presets) string {
	template := ".items[] | %s | [%s]"

	return fmt.Sprintf(template, translateWhere(where, presets), translateSelect(selects, presets))
}

func main() {

	args := os.Args[1:]

	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("APPDATA")
	}

	configFile := ""
	if home == "" {
		configFile = ".kubequery"
	} else {
		configFile = fmt.Sprintf("%s%c.kubequery", home, os.PathSeparator)
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		file, err := os.Create(configFile)
		assert(err, fmt.Sprintf("Error creating file %s", configFile))

		defer file.Close()

		_, err = file.Write(presetsBytes)

		assert(err, fmt.Sprintf("Error writing to file %s", configFile))
	}

	file, err := os.Open(configFile)
	assert(err, fmt.Sprintf("Error opening file %s", configFile))
	defer file.Close()

	loadedPresets, err := io.ReadAll(file)
	assert(err, fmt.Sprintf("Error reading file %s", configFile))

	presets = Presets{}
	err = json.Unmarshal(loadedPresets, &presets)
	assert(err, fmt.Sprintf("Error parsing file %s", configFile))

	parsedArgs := make([]string, 0)
	resource := "all"
	where := "id()"
	selects := "all"

	for i := 0; i < len(args); i++ {
		if args[i] == "-w" {
			assertCondition(i+1 < len(args), "Missing query for -q parameter")
			where = args[i+1]
			i++

		} else if args[i] == "-x" {
			assertCondition(i+1 < len(args), "Missing query for -x parameter")
			resource = args[i+1]
			i++
		} else if args[i] == "-s" {
			assertCondition(i+1 < len(args), "Missing query for -s parameter")
			selects = args[i+1]
			i++
		} else {
			parsedArgs = append(parsedArgs, args[i])
		}
	}

	parsedArgs = append(parsedArgs, "-o", "json")

	parsedArgs = append([]string{"get", resource}, parsedArgs...)

	cmd := exec.Command("kubectl", parsedArgs...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr

	outBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf

	err = cmd.Run()

	if err != nil {
		fmt.Printf("Error running kubectl command: %v\n", err)
		os.Exit(1)
	}

	query := translateQuery(selects, where, presets)

	jqCmd := exec.Command("jq", query)
	jqCmd.Env = os.Environ()
	jqCmd.Stderr = os.Stderr
	jqCmd.Stdin = outBuf
	jqCmd.Stdout = os.Stdout

	jqCmd.Run()

}
