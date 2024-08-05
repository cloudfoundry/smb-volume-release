package smbdriver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func ToKernelMountOptionFlagsAndEnvVars(mountOpts map[string]interface{}) (string, []string) {
	mountFlags, mountEnvVars := separateFlagsAndEnvVars(mountOpts)

	kernelMountOptions := convertToStringArr(sanitizeMountFlags(mountFlags))
	kernelMountEnvVars := convertToStringArr(sanitizeMountFlags(mountEnvVars))

	return strings.Join(kernelMountOptions, ","), kernelMountEnvVars
}

func convertToStringArr(mountOpts map[string]interface{}, valueless []string) []string {
	paramList := []string{}

	for k, v := range mountOpts {
		switch t := v.(type) {
		case string:
			if val, err := strconv.ParseInt(t, 10, 16); err == nil {
				paramList = append(paramList, fmt.Sprintf("%s=%d", k, val))
			} else {
				paramList = append(paramList, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	paramList = append(paramList, valueless...)
	sort.Strings(paramList)
	return paramList
}

func separateFlagsAndEnvVars(mountOpts map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	flagList := make(map[string]interface{})
	envVarList := make(map[string]interface{})

	for k, v := range mountOpts {
		if strings.ToLower(k) == "username" {
			envVarList[k] = v
		} else if strings.ToLower(k) == "password" {
			envVarList[k] = v
		} else {
			flagList[k] = v
		}
	}

	return flagList, envVarList
}

func sanitizeMountFlags(mountOpts map[string]interface{}) (map[string]interface{}, []string) {
	result := make(map[string]interface{})
	valueless := []string{}

	for k, v := range mountOpts {
		if strings.ToLower(k) == "username" {
			result["USER"] = v
		} else if strings.ToLower(k) == "password" {
			result["PASSWD"] = v
		} else if strings.ToLower(k) == "domain" {
			if v != "" {
				result["domain"] = v
			}
		} else if strings.ToLower(k) == "mfsymlinks" {
			if v == "true" || v == "" {
				valueless = append(valueless, "mfsymlinks")
			}
		} else if strings.ToLower(k) == "nodfs" {
			if v == "true" || v == "" {
				valueless = append(valueless, "nodfs")
			}
		} else {
			result[k] = v
		}
	}
	return result, valueless
}
