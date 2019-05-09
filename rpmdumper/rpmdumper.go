// Copyright 2019 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
)

func LinesInFile(fileName string) ([]string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	// Create new Scanner.
	scanner := bufio.NewScanner(f)
	result := []string{}
	// Use Scan.
	for scanner.Scan() {
		line := scanner.Text()
		// Append line to result.
		result = append(result, line)
	}
	return result, nil
}

func extractFiles(lines []string) error {
	fileInfo := regexp.MustCompile(`^(\w+) scriptlet \(using /bin/sh\):$`)

	var file *os.File
	var err error
	for _, line := range lines {
		if matchs := fileInfo.FindStringSubmatch(line); len(matchs) == 2 {
			if file != nil {
				file.Close()
			}

			if file, err = os.Create(matchs[1]); err != nil {
				return err
			}
			continue
		}
		if file == nil {
			return errors.New("file handle is nil")
		}
		file.Write([]byte(line + "\n"))
	}
	if file != nil {
		file.Close()
	}

	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <rpm-scripts-file>\n", os.Args[0])
		os.Exit(1)
	}

	lines, err := LinesInFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	if err := extractFiles(lines); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
