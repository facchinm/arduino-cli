/*
 * This file is part of Arduino Builder.
 *
 * Arduino Builder is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
 *
 * As a special exception, you may use this file as part of a free software
 * library without restriction.  Specifically, if other files instantiate
 * templates or use macros or inline functions from this file, or you compile
 * this file and link it with other files to produce an executable, this
 * file does not by itself cause the resulting executable to be covered by
 * the GNU General Public License.  This exception does not however
 * invalidate any other reasons why the executable file might be covered by
 * the GNU General Public License.
 *
 * Copyright 2015 Arduino LLC (http://www.arduino.cc/)
 */

package builder

import (
	"errors"
	"math"
	"os"
	"strconv"

	"github.com/arduino/go-paths-helper"
	"github.com/marcinbor85/gohex"

	"github.com/arduino/arduino-cli/legacy/builder/constants"
	"github.com/arduino/arduino-cli/legacy/builder/types"
)

type MergeSketchWithBootloader struct{}

func (s *MergeSketchWithBootloader) Run(ctx *types.Context) error {
	buildProperties := ctx.BuildProperties
	if !buildProperties.ContainsKey(constants.BUILD_PROPERTIES_BOOTLOADER_NOBLINK) && !buildProperties.ContainsKey(constants.BUILD_PROPERTIES_BOOTLOADER_FILE) {
		return nil
	}

	buildPath := ctx.BuildPath
	sketch := ctx.Sketch
	sketchFileName := sketch.MainFile.Name.Base()
	logger := ctx.GetLogger()

	sketchInBuildPath := buildPath.Join(sketchFileName + ".hex")
	sketchInSubfolder := buildPath.Join(constants.FOLDER_SKETCH, sketchFileName+".hex")

	var builtSketchPath *paths.Path
	if sketchInBuildPath.Exist() {
		builtSketchPath = sketchInBuildPath
	} else if sketchInSubfolder.Exist() {
		builtSketchPath = sketchInSubfolder
	} else {
		return nil
	}

	bootloader := constants.EMPTY_STRING
	if bootloaderNoBlink, ok := buildProperties.GetOk(constants.BUILD_PROPERTIES_BOOTLOADER_NOBLINK); ok {
		bootloader = bootloaderNoBlink
	} else {
		bootloader = buildProperties.Get(constants.BUILD_PROPERTIES_BOOTLOADER_FILE)
	}
	bootloader = buildProperties.ExpandPropsInString(bootloader)

	bootloaderPath := buildProperties.GetPath(constants.BUILD_PROPERTIES_RUNTIME_PLATFORM_PATH).Join(constants.FOLDER_BOOTLOADERS, bootloader)
	if bootloaderPath.NotExist() {
		logger.Fprintln(os.Stdout, constants.LOG_LEVEL_WARN, constants.MSG_BOOTLOADER_FILE_MISSING, bootloaderPath)
		return nil
	}

	mergedSketchPath := builtSketchPath.Parent().Join(sketchFileName + ".with_bootloader.hex")

	// Ignore merger errors for the first iteration
	maximumBinSize := 16000000
	if uploadMaxSize, ok := buildProperties.GetOk(constants.PROPERTY_UPLOAD_MAX_SIZE); ok {
		maximumBinSize, _ = strconv.Atoi(uploadMaxSize)
		maximumBinSize *= 2
	}
	err := merge(builtSketchPath, bootloaderPath, mergedSketchPath, maximumBinSize)
	if err != nil {
		logger.Fprintln(os.Stdout, constants.LOG_LEVEL_WARN, err.Error())
	}

	return nil
}

func merge(builtSketchPath, bootloaderPath, mergedSketchPath *paths.Path, maximumBinSize int) error {
	if bootloaderPath.Ext() == ".bin" {
		bootloaderPath = bootloaderPath.Join(bootloaderPath.Base(), ".hex")
	}

	bootFile, err := os.Open(bootloaderPath.String())
	if err != nil {
		return err
	}
	defer bootFile.Close()

	mem_boot := gohex.NewMemory()
	err = mem_boot.ParseIntelHex(bootFile)
	if err != nil {
		return errors.New(bootFile.Name() + " " + err.Error())
	}

	buildFile, err := os.Open(builtSketchPath.String())
	if err != nil {
		return err
	}
	defer buildFile.Close()

	mem_sketch := gohex.NewMemory()
	err = mem_sketch.ParseIntelHex(buildFile)
	if err != nil {
		return errors.New(buildFile.Name() + " " + err.Error())
	}

	mem_merge := gohex.NewMemory()
	initial_address := uint32(math.MaxUint32)
	last_address := uint32(0)

	for _, segment := range mem_boot.GetDataSegments() {
		err = mem_merge.AddBinary(segment.Address, segment.Data)
		if err != nil {
			continue
		} else {
			if segment.Address < initial_address {
				initial_address = segment.Address
			}
			if segment.Address+uint32(len(segment.Data)) > last_address {
				last_address = segment.Address + uint32(len(segment.Data))
			}
		}
	}
	for _, segment := range mem_sketch.GetDataSegments() {
		err = mem_merge.AddBinary(segment.Address, segment.Data)
		if err != nil {
			continue
		}
		if segment.Address < initial_address {
			initial_address = segment.Address
		}
		if segment.Address+uint32(len(segment.Data)) > last_address {
			last_address = segment.Address + uint32(len(segment.Data))
		}
	}

	mergeFile, err := os.Create(mergedSketchPath.String())
	if err != nil {
		return err
	}
	defer mergeFile.Close()

	mem_merge.DumpIntelHex(mergeFile, 16)

	mergedSketchPathBin := mergedSketchPath.Join(mergedSketchPath.Base(), ".bin")

	size := last_address - initial_address
	if size > uint32(maximumBinSize) {
		return nil
	}

	bytes := mem_merge.ToBinary(initial_address, last_address-initial_address, 0xFF)
	return mergedSketchPathBin.WriteFile(bytes)
}
