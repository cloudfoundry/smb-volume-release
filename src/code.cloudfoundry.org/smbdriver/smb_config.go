package smbdriver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Config represents the configurations for SMB mount
type Config struct {
	Allowed   []string
	Mandatory []string

	Forced  map[string]string
	Options map[string]string
}

func inArray(list []string, key string) bool {
	for _, k := range list {
		if k == key {
			return true
		}
	}

	return false
}

// NewSmbConfig creates Config for SMB
func NewSmbConfig() *Config {
	myConf := new(Config)

	myConf.Allowed = make([]string, 0)
	myConf.Options = make(map[string]string, 0)
	myConf.Forced = make(map[string]string, 0)
	myConf.Mandatory = make([]string, 0)

	return myConf
}

// Copy copy a config
func (config *Config) Copy() *Config {
	myConf := new(Config)

	myConf.Allowed = config.Allowed
	myConf.Mandatory = config.Mandatory

	myConf.Forced = make(map[string]string, 0)
	myConf.Options = make(map[string]string, 0)
	for k, v := range config.Forced {
		myConf.Forced[k] = v
	}
	for k, v := range config.Options {
		myConf.Options[k] = v
	}
	return myConf
}

// SetEntries set options to a config. Those options in ignoreList will be ignored.
func (config *Config) SetEntries(opts map[string]interface{}, ignoreList []string) error {
	errorList := config.parseMap(opts, ignoreList)

	if len(errorList) > 0 {
		err := errors.New("Not allowed options : " + strings.Join(errorList, ", "))
		return err
	}

	if mdtErr := config.CheckMandatory(); len(mdtErr) > 0 {
		err := errors.New("Missing mandatory options : " + strings.Join(mdtErr, ", "))
		return err
	}

	return nil
}

// MakeParams generate parameters as an array from config.
func (config Config) MakeParams() []string {
	params := []string{}

	for k, v := range config.MakeConfig() {
		if val, err := strconv.ParseInt(v.(string), 10, 16); err == nil {
			params = append(params, fmt.Sprintf("%s=%d", k, val))
			continue
		}

		if k == "readonly" || k == "ro" {
			params = append(params, "ro")
			continue
		}

		params = append(params, fmt.Sprintf("%s=%s", k, v.(string)))
	}

	return params
}

// MakeConfig generate parameters as a map from config.
func (config Config) MakeConfig() map[string]interface{} {
	params := map[string]interface{}{}

	for k, v := range config.Options {
		params[k] = v
	}

	for k, v := range config.Forced {
		params[k] = v
	}

	return params
}

// ReadConf read config.
func (config *Config) ReadConf(allowedFlag string, defaultFlag string, mandatoryFields []string) error {
	if len(allowedFlag) > 0 {
		config.Allowed = strings.Split(allowedFlag, ",")
	}

	config.readConfDefault(defaultFlag)

	if len(mandatoryFields) > 0 {
		config.Mandatory = mandatoryFields
	}

	return nil
}

// CheckMandatory get missing mandatory options.
func (config Config) CheckMandatory() []string {
	var result []string

	for _, k := range config.Mandatory {
		_, oko := config.Options[k]
		_, okf := config.Forced[k]

		if !okf && !oko {
			result = append(result, k)
		}
	}

	return result
}

func (config *Config) readConfDefault(flagString string) {
	if len(flagString) < 1 {
		return
	}

	config.Options = config.parseConfig(strings.Split(flagString, ","))
	config.Forced = make(map[string]string)

	for k, v := range config.Options {
		if !inArray(config.Allowed, k) {
			config.Forced[k] = v
			delete(config.Options, k)
		}
	}
}

func (config Config) parseConfig(listEntry []string) map[string]string {
	result := map[string]string{}

	for _, opt := range listEntry {
		key := strings.SplitN(opt, ":", 2)

		if len(key[0]) < 1 {
			continue
		}

		if len(key[1]) < 1 {
			result[key[0]] = ""
		} else {
			result[key[0]] = key[1]
		}
	}

	return result
}

func (config Config) uniformData(data interface{}, boolAsInt bool) string {
	switch data.(type) {
	case int:
		return strconv.FormatInt(int64(data.(int)), 10)

	case string:
		return data.(string)

	case bool:
		if boolAsInt {
			if data.(bool) {
				return "1"
			} else {
				return "0"
			}
		} else {
			return strconv.FormatBool(data.(bool))
		}
	}

	return ""
}

func (config *Config) parseMap(entryList map[string]interface{}, ignoreList []string) []string {
	errorList := []string{}

	for k, v := range entryList {
		value := config.uniformData(v, false)

		if value == "" || len(k) < 1 || inArray(ignoreList, k) {
			continue
		}

		if inArray(config.Allowed, k) {
			config.Options[k] = value
		} else {
			errorList = append(errorList, k)
		}
	}

	return errorList
}
