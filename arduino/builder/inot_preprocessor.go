// This file is part of arduino-cli.
//
// Copyright 2021 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package builder

import (
	"strings"

	"github.com/arduino/go-paths-helper"
)

// GetSourceStr reads the item file contents and returns it as a string
func getSourceStrWithoutLinesStartingWith(file *paths.Path, start string) (string, string, int, error) {
	source, err := file.ReadFile()
	if err != nil {
		return "", "", 0, err
	}
	lines := strings.Split(string(source), "\n")
	out_ok := ""
	out_ko := ""
	howMany := 0
	for _, line := range lines {
		if strings.HasPrefix(line, start) {
			out_ko += line + "\n"
			howMany++
		} else {
			out_ok += line + "\n"
		}
	}
	return out_ok, out_ko, howMany, err
}

func getSourceLines(file *paths.Path) int {
	n := 0
	data, err := file.ReadFile()
	s := string(data)
	if err != nil {
		return 0
	}
	for _, r := range s {
		if r == '\n' {
			n++
		}
	}
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

func preprocessInot(mergedSource string, lineOffset int, ThreadSketchFiles paths.PathList) (string, int, error) {
	mergedSource += "#include <Arduino_Threads.h>\n"
	mergedSource += "#include \"SharedVariables.h\"\n"
	mergedSource += "__INCLUDES_PLACEHOLDER__\n"
	lineOffset += 2
	includes := ""
	howManyIncludes := 1
	for _, el := range ThreadSketchFiles {
		src, include, howManyInclude, err := getSourceStrWithoutLinesStartingWith(el, "#include")
		if err != nil {
			return "", 0, err
		}
		includes += include
		howManyIncludes += howManyInclude
		filename := strings.TrimSuffix(el.Base(), el.Ext())
		mergedSource += "THD_ENTER(" + filename + ")\n"
		mergedSource += "#line 1 " + QuoteCppString(el.String()) + "\n"
		mergedSource += src
		mergedSource += "THD_DONE(" + filename + ")\n"
		lineOffset += getSourceLines(el) + 2
	}
	mergedSource += "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"
	lineOffset += 16
	mergedSource = strings.Replace(mergedSource, "__INCLUDES_PLACEHOLDER__", includes, 1)
	lineOffset += howManyIncludes - 1
	return mergedSource, lineOffset, nil
}
