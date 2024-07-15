package debug

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var triggerconfigRegex = regexp.MustCompile(`^\[\[trigger\.(\w+)\]\]$`)
var componentConfigRegex = regexp.MustCompile(`^\[component\..*\]$`)

func update(filename, component string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(file)
	sectionLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Println("line is > ", line)
		// fmt.Println(">>>>>>")
		// fmt.Println(sectionLines)
		// fmt.Println("<<<<<<")
		if triggerconfigRegex.MatchString(line) || componentConfigRegex.MatchString(line) {
			if len(sectionLines) > 0 {
				// parse already loaded section
				parsed := parseSection(sectionLines, component)

				// reset section lines
				sectionLines = []string{}

				// print updated lines
				fmt.Fprintln(&buf, parsed)
			}

			sectionLines = append(sectionLines, line)
		} else if len(sectionLines) > 0 {
			// fmt.Println("appending section lines ", len(sectionLines))
			sectionLines = append(sectionLines, line)
		} else {
			// fmt.Println("printing out")
			fmt.Fprintln(&buf, line)
		}
	}

	if len(sectionLines) > 0 {
		// parse already loaded section
		parsed := parseSection(sectionLines, component)

		// print updated lines
		fmt.Fprintln(&buf, parsed)
	}

	return buf.String(), nil
}

func parseSection(lines []string, component string) string {
	if triggerconfigRegex.MatchString(lines[0]) {
		return parseTriggerConfigSection(lines, component)
	}

	return componentConfigSection(lines, component)
}

func componentConfigSection(lines []string, component string) string {
	requestedComponentFound := false
	// if the component is the requested component, do something about it. else return as is
	for _, line := range lines {
		if line == fmt.Sprintf(`[component.%s]`, component) {
			requestedComponentFound = true
		}
	}

	if !requestedComponentFound {
		return strings.Join(lines, "\n")
	}

	// just change the source to be wasm_console.wasm
	// keep all other configs same
	updatedLines := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, "source = ") {
			updatedLines = append(updatedLines, `source = { url = "https://github.com/rajatjindal/wasm-console/raw/main/main.wasm", digest = "sha256:ca1ab307ae38892efdc81379c93b1e29e6b1565ea706e874576d72b8f2598439" }`)
			continue
		}

		updatedLines = append(updatedLines, line)
	}

	return strings.Join(updatedLines, "\n")
}

func parseTriggerConfigSection(lines []string, component string) string {
	requestedComponentFound := false
	// if the component is the requested component, do something about it. else return as is
	for _, line := range lines {
		if line == fmt.Sprintf(`component = "%s"`, component) {
			requestedComponentFound = true
		}
	}

	if !requestedComponentFound {
		return strings.Join(lines, "\n")
	}

	// remove all other trigger specific config.
	// e.g. remove `route` if it was an http trigger
	return strings.Join(
		[]string{
			`[[trigger.command]]`,
			fmt.Sprintf(`component = "%s"`, component),
			"",
		}, "\n")
}
