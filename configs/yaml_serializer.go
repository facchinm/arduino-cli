/*
 * This file is part of arduino-cli.
 *
 * Copyright 2018 ARDUINO SA (http://www.arduino.cc/)
 *
 * This software is released under the GNU General Public License version 3,
 * which covers the main part of arduino-cli.
 * The terms of this license can be found at:
 * https://www.gnu.org/licenses/gpl-3.0.en.html
 *
 * You can be released from the requirements of the above licenses by purchasing
 * a commercial license. Buying such a license is mandatory if you want to modify or
 * otherwise use the software for commercial activities involving the Arduino
 * software without disclosing the source code of your own applications. To purchase
 * a commercial license, send an email to license@arduino.cc.
 */

package configs

import (
	"fmt"
	"io/ioutil"
	"net/url"

	paths "github.com/arduino/go-paths-helper"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type yamlConfig struct {
	ProxyType         string                   `yaml:"proxy_type"`
	ProxyManualConfig *yamlProxyConfig         `yaml:"manual_configs,omitempty"`
	SketchbookPath    string                   `yaml:"sketchbook_path,omitempty"`
	ArduinoDataDir    string                   `yaml:"arduino_data,omitempty"`
	BoardsManager     *yamlBoardsManagerConfig `yaml:"board_manager"`
}

type yamlBoardsManagerConfig struct {
	AdditionalURLS []string `yaml:"additional_urls,omitempty"`
}

type yamlProxyConfig struct {
	Hostname string `yaml:"hostname"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"` // can be encrypted, see issue #71
}

// LoadFromYAML loads the configs from a yaml file.
func (config *Configuration) LoadFromYAML(path *paths.Path) error {
	logrus.Info("Unserializing configurations from ", path)
	content, err := path.ReadFile()
	if err != nil {
		logrus.WithError(err).Warn("Error reading config, using default configuration")
		return err
	}
	var ret yamlConfig
	err = yaml.Unmarshal(content, &ret)
	if err != nil {
		logrus.WithError(err).Warn("Error parsing config, using default configuration")
		return err
	}

	if ret.ArduinoDataDir != "" {
		config.DataDir = paths.New(ret.ArduinoDataDir)
	}
	if ret.SketchbookPath != "" {
		config.SketchbookDir = paths.New(ret.SketchbookPath)
	}
	if ret.ProxyType != "" {
		config.ProxyType = ret.ProxyType
		if ret.ProxyManualConfig != nil {
			config.ProxyHostname = ret.ProxyManualConfig.Hostname
			config.ProxyUsername = ret.ProxyManualConfig.Username
			config.ProxyPassword = ret.ProxyManualConfig.Password
		}
	}
	if ret.BoardsManager != nil {
		for _, rawurl := range ret.BoardsManager.AdditionalURLS {
			url, err := url.Parse(rawurl)
			if err != nil {
				logrus.WithError(err).Warn("Error parsing config")
				continue
			}
			config.BoardManagerAdditionalUrls = append(config.BoardManagerAdditionalUrls, url)
		}
	}
	return nil
}

// SerializeToYAML encodes the current configuration as YAML
func (config *Configuration) SerializeToYAML() ([]byte, error) {
	c := &yamlConfig{}
	if config.SketchbookDir != nil {
		c.SketchbookPath = config.SketchbookDir.String()
	}
	if config.DataDir != nil {
		c.ArduinoDataDir = config.DataDir.String()
	}
	c.ProxyType = config.ProxyType
	if config.ProxyType == "manual" {
		c.ProxyManualConfig = &yamlProxyConfig{
			Hostname: config.ProxyHostname,
			Username: config.ProxyUsername,
			Password: config.ProxyPassword,
		}
	}
	if len(config.BoardManagerAdditionalUrls) > 1 {
		c.BoardsManager = &yamlBoardsManagerConfig{AdditionalURLS: []string{}}
		for _, URL := range config.BoardManagerAdditionalUrls[1:] {
			c.BoardsManager.AdditionalURLS = append(c.BoardsManager.AdditionalURLS, URL.String())
		}
	}
	return yaml.Marshal(c)
}

// SaveToYAML the current configuration to a YAML file
func (config *Configuration) SaveToYAML(path string) error {
	content, err := config.SerializeToYAML()
	if err != nil {
		return fmt.Errorf("econding configuration to YAML: %s", err)
	}

	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		return fmt.Errorf("writing configuration to %s: %s", path, err)
	}
	return nil
}
