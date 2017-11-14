package azurefilebroker

import (
	"errors"
	"strings"
)

const preexisting = "Preexisting"

type MountConfig struct {
	Allowed []string

	Forced  map[string]string
	Options map[string]string
}

type AzureConfig struct {
	Environment              string
	TenanID                  string
	ClientID                 string
	ClientSecret             string
	DefaultSubscriptionID    string
	DefaultResourceGroupName string
	DefaultLocation          string
}

func NewAzureConfig(environment, tenanID, clientID, clientSecret, defaultSubscriptionID, defaultResourceGroupName, defaultLocation string) *AzureConfig {
	myConf := new(AzureConfig)

	myConf.Environment = environment
	myConf.TenanID = tenanID
	myConf.ClientID = clientID
	myConf.ClientSecret = clientSecret
	myConf.DefaultSubscriptionID = defaultSubscriptionID
	myConf.DefaultResourceGroupName = defaultResourceGroupName
	myConf.DefaultLocation = defaultLocation

	return myConf
}

func (config *AzureConfig) IsSupportAzureFileShare() bool {
	return config.Environment != preexisting
}

func (config *AzureConfig) Validate() error {
	if !config.IsSupportAzureFileShare() {
		return nil
	}

	missingKeys := []string{}
	if config.Environment == "" {
		missingKeys = append(missingKeys, "environment")
	}
	if config.TenanID == "" {
		missingKeys = append(missingKeys, "tenanID")
	}
	if config.ClientID == "" {
		missingKeys = append(missingKeys, "clientID")
	}
	if config.ClientSecret == "" {
		missingKeys = append(missingKeys, "clientSecret")
	}

	if len(missingKeys) > 0 {
		return errors.New("Missing required parameters: " + strings.Join(missingKeys, ", "))
	}
	return nil
}

type ControlConfig struct {
	AllowCreateStorageAccount bool
	AllowCreateFileShare      bool
	AllowDeleteStorageAccount bool
	AllowDeleteFileShare      bool
}

func NewControlConfig(allowCreateStorageAccount, allowCreateFileShare, allowDeleteStorageAccount, allowDeleteFileShare bool) *ControlConfig {
	myConf := new(ControlConfig)

	myConf.AllowCreateStorageAccount = allowCreateStorageAccount
	myConf.AllowCreateFileShare = allowCreateFileShare
	myConf.AllowDeleteStorageAccount = allowDeleteStorageAccount
	myConf.AllowDeleteFileShare = allowDeleteFileShare

	return myConf
}

type AzureStackConfig struct {
	AzureStackDomain         string
	AzureStackAuthentication string
	AzureStackResource       string
	AzureStackEndpointPrefix string
}

func NewAzureStackConfig(azureStackDomain, azureStackAuthentication, azureStackResource, azureStackEndpointPrefix string) *AzureStackConfig {
	myConf := new(AzureStackConfig)

	myConf.AzureStackDomain = azureStackDomain
	myConf.AzureStackAuthentication = azureStackAuthentication
	myConf.AzureStackResource = azureStackResource
	myConf.AzureStackEndpointPrefix = azureStackEndpointPrefix

	return myConf
}

func (config *AzureStackConfig) Validate() error {
	missingKeys := []string{}
	if config.AzureStackDomain == "" {
		missingKeys = append(missingKeys, "azureStackDomain")
	}
	if config.AzureStackAuthentication == "" {
		missingKeys = append(missingKeys, "azureStackAuthentication")
	}
	if config.AzureStackResource == "" {
		missingKeys = append(missingKeys, "azureStackResource")
	}
	if config.AzureStackEndpointPrefix == "" {
		missingKeys = append(missingKeys, "azureStackEndpointPrefix")
	}

	if len(missingKeys) > 0 {
		return errors.New("Missing required parameters when 'environment' is 'AzureStack': " + strings.Join(missingKeys, ", "))
	}
	return nil
}

type CloudConfig struct {
	Azure      AzureConfig
	Control    ControlConfig
	AzureStack AzureStackConfig
}

type Config struct {
	mount MountConfig
	cloud CloudConfig
}

func inArray(list []string, key string) bool {
	for _, k := range list {
		if k == key {
			return true
		}
	}

	return false
}

func NewAzurefilebrokerConfig(mountConfig *MountConfig, cloudConfig *CloudConfig) *Config {
	myConf := new(Config)

	myConf.mount = *mountConfig
	myConf.cloud = *cloudConfig

	return myConf
}

func NewAzurefilebrokerMountConfig() *MountConfig {
	myConf := new(MountConfig)

	myConf.Allowed = make([]string, 0)
	myConf.Options = make(map[string]string, 0)
	myConf.Forced = make(map[string]string, 0)

	return myConf
}

func NewAzurefilebrokerCloudConfig(azure *AzureConfig, control *ControlConfig, azureStack *AzureStackConfig) *CloudConfig {
	myConf := new(CloudConfig)

	myConf.Azure = *azure
	myConf.Control = *control
	myConf.AzureStack = *azureStack

	return myConf
}

func (config *CloudConfig) Validate() error {
	if err := config.Azure.Validate(); err != nil {
		return err
	}

	if config.Azure.Environment == AzureStack {
		if err := config.AzureStack.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (config *MountConfig) Copy() *MountConfig {
	myConf := new(MountConfig)

	myConf.Allowed = config.Allowed

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

func (config *MountConfig) SetEntries(opts map[string]string) error {
	errorList := []string{}

	for k, v := range opts {
		if inArray(config.Allowed, k) {
			config.Options[k] = v
		} else {
			errorList = append(errorList, k)
		}
	}

	if len(errorList) > 0 {
		err := errors.New("Not allowed options : " + strings.Join(errorList, ", "))
		return err
	}

	return nil
}

func (config MountConfig) MakeConfig() map[string]interface{} {
	params := map[string]interface{}{}

	for k, v := range config.Options {
		params[k] = v
	}

	for k, v := range config.Forced {
		params[k] = v
	}

	return params
}

func (config *MountConfig) ReadConf(allowedFlag string, defaultFlag string) error {
	if len(allowedFlag) > 0 {
		config.Allowed = strings.Split(allowedFlag, ",")
	}

	config.readConfDefault(defaultFlag)

	return nil
}

func (config *MountConfig) readConfDefault(flagString string) {
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

func (config MountConfig) parseConfig(listEntry []string) map[string]string {
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
