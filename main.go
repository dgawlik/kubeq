package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/itchyny/gojq"
)

func translateQuery(query string, dict map[string]string) string {
	openBraceIdx := strings.Index(query, "(")
	closeBraceIdx := strings.LastIndex(query, ")")

	if openBraceIdx == -1 || closeBraceIdx == -1 {
		fmt.Printf("Invalid query: %s\n", query)
		os.Exit(1)
	}

	funcName := query[:openBraceIdx]
	argsString := query[openBraceIdx+1 : closeBraceIdx]

	query, ok := dict[funcName]
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

	return string(query)
}

func main() {
	builtinQueries := map[string]string{
		"podsForImage":          "select(.items.[].spec.containers.[].image | test(\"$1\", \"g\"))",
		"podsForName":           "select(.items[].metadata.name | test(\"$1\", \"g\"))",
		"podsForLabel":          ".items | map(select(.metadata.labels.$1 == $2))",
		"podsForMountPath":      ".items | map(select(.spec.containers.[].volumeMounts.[].mountPath == $1))",
		"podsForReadinessProbe": ".items | map(select(.spec.containers.[].readinessProbe.httpGet.path == $1 and .spec.containers.[].readinessProbe.httpGet.port == $2))",
		"servicesForTargetPort": ".items | map(select(.spec.ports[].targetPort == $1))",
		"servicesForPort":       ".items | map(select(.spec.ports[].port == $1))",
		"servicesForSelector":   ".items | map(select(.spec.selector.$1 == $2))",
		"roleByResourceName":    ".items | map(select(.rules.resourceNames. == $1))",
	}

	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("No arguments provided.")
		os.Exit(1)
	}

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

	fmt.Println(configFile)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		file, err := os.Create(configFile)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", configFile, err)
			os.Exit(1)
		}
		defer file.Close()

		for k, v := range builtinQueries {
			content := fmt.Sprintf("%s=%s\n", k, v)
			_, err := file.WriteString(content)
			if err != nil {
				fmt.Printf("Error writing to file %s: %v\n", configFile, err)
				os.Exit(1)
			}
		}
	}

	file, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", configFile, err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	queries := make(map[string]string)
	for scanner.Scan() {

		line := scanner.Text()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error reading file %s: %v\n", configFile, err)
			os.Exit(1)
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			queries[strings.Trim(parts[0], "\n\t ")] = strings.Trim(parts[1], "\n\t ")
		}
	}

	parsedArgs := make([]string, 0)
	queryArgsIndex := -1
	overrideAllIndex := -1
	overrideAll := ""
	query := ""
	for i, arg := range args {
		if args[i] == "-q" {
			if i+1 == len(args) {
				fmt.Printf("Missing query for -q parameter")
				os.Exit(1)
			} else {
				queryArgsIndex = i + 1
			}
		} else if args[i] == "-x" {
			if i+1 == len(args) {
				fmt.Printf("Missing query for -q parameter")
				os.Exit(1)
			} else {
				overrideAllIndex = i + 1
			}
		} else if i == queryArgsIndex {
			query = translateQuery(args[i], queries)
		} else if i == overrideAllIndex {
			overrideAll = args[i]
		} else {
			parsedArgs = append(parsedArgs, arg)
		}
	}

	parsedArgs = append(parsedArgs, "-o", "json")

	cmd := exec.Command("kubectl")
	cmd.Env = os.Environ()
	err = cmd.Run()
	if err != nil {
		fmt.Println("kubectl command not found.")
		os.Exit(1)
	}

	tempFile, err := os.Create(".tmp")
	if err != nil {
		fmt.Printf("Error creating pipe file: %v\n", err)
		os.Exit(1)
	}
	defer tempFile.Close()
	defer os.Remove(".tmp")

	res := ""
	if overrideAll != "" {
		res = overrideAll
	} else {
		res = "all"
	}

	parsedArgs = append([]string{"get", res}, parsedArgs...)

	cmd = exec.Command("kubectl", parsedArgs...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr

	cmd.Stdout = tempFile

	err = cmd.Run()

	if err != nil {
		fmt.Printf("Error running kubectl command: %v\n", err)
		os.Exit(1)
	}

	content, err := os.ReadFile(".tmp")
	if err != nil {
		fmt.Printf("Error reading pipe file: %v\n", err)
		os.Exit(1)
	}

	jqQuery, err := gojq.Parse(query)
	if err != nil {
		log.Fatalln(err)
	}

	var data interface{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		fmt.Printf("Error unmarshalling JSON: %v\n", err)
		os.Exit(1)
	}

	result := make([]interface{}, 0)
	iter := jqQuery.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		result = append(result, v)
	}

	resultB, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(resultB))

}
