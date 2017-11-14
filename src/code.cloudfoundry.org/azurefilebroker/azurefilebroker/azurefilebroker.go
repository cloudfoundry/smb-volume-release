package azurefilebroker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	"crypto/md5"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
)

const (
	permissionVolumeMount = brokerapi.RequiredPermission("volume_mount")
	defaultContainerPath  = "/var/vcap/data"
)

const (
	driverName       string = "smbdriver"
	deviceTypeShared string = "shared"
	databaseVersion  string = "1.0"
)

const (
	lockTimeoutInSeconds int = 30
)

/*
This broker supports both AzureFileShare and preexisting shares.
	AzureFileShare:
		Provision with parameters: subscription_id, resource_group_name, storage_account_name, location, use_https, sku_name, enable_encryption, custom_domain_name, use_sub_domain
			Create or use a storage account
		Bind with parameters which are defined in BindOptions: uid, gid, file_mode, dir_mode, readonly, mount, vers, share
			Create or use a file share; Return credentials
		Unbind
			Delete a file share or do nothing
		Deprovision
			Delete a storage account or do nothing
	Preexisting shares:
		Provision with parameters: share
			Use a preexsting share
		Bind with parameters: uid, gid, file_mode, dir_mode, readonly, mount, domain, username, password, sec
			Return credentials
		Unbind
			Do nothing
		Deprovision
			Do nothing
*/

// TBD: custom_domain_name and use_sub_domain are not supported now.
type Configuration struct {
	SubscriptionID     string `json:"subscription_id"`
	ResourceGroupName  string `json:"resource_group_name"`
	StorageAccountName string `json:"storage_account_name"` // Required for AzureFileShare
	Location           string `json:"location"`
	UseHTTPS           string `json:"use_https"` // bool
	SkuName            string `json:"sku_name"`
	CustomDomainName   string `json:"custom_domain_name"`
	UseSubDomain       string `json:"use_sub_domain"`    // bool
	EnableEncryption   string `json:"enable_encryption"` // bool
	Share              string `json:"share"`             // Required for preexisting shares
}

func (config *Configuration) ValidateForAzureFileShare() error {
	missingKeys := []string{}
	if config.SubscriptionID == "" {
		missingKeys = append(missingKeys, "subscription_id")
	}
	if config.ResourceGroupName == "" {
		missingKeys = append(missingKeys, "resource_group_name")
	}
	if config.StorageAccountName == "" {
		missingKeys = append(missingKeys, "storage_account_name")
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("Missing required parameters: %s", strings.Join(missingKeys, ", "))
	}
	return nil
}

type BindOptions struct {
	UID           string `json:"uid"`
	GID           string `json:"gid"`
	FileMode      string `json:"file_mode"`
	DirMode       string `json:"dir_mode"`
	Readonly      bool   `json:"readonly"`
	Mount         string `json:"mount"`
	Vers          string `json:"vers"`     // Required for AzureFileShare
	FileShareName string `json:"share"`    // Required for AzureFileShare
	Domain        string `json:"domain"`   // Optional for preexisting shares
	Username      string `json:"username"` // Required for preexisting shares
	Password      string `json:"password"` // Optional for preexisting shares
	Sec           string `json:"sec"`      // Optional for preexisting shares
}

// ToMap Omit Mount, FileShareName, Domain, Username and Password
func (options BindOptions) ToMap() map[string]string {
	ret := make(map[string]string)
	if options.UID != "" {
		ret["uid"] = options.UID
	}
	if options.GID != "" {
		ret["gid"] = options.GID
	}
	if options.FileMode != "" {
		ret["file_mode"] = options.FileMode
	}
	if options.DirMode != "" {
		ret["dir_mode"] = options.DirMode
	}
	if options.Readonly {
		ret["readonly"] = strconv.FormatBool(options.Readonly)
	}
	if options.Vers != "" {
		ret["vers"] = options.Vers
	}
	if options.Sec != "" {
		ret["sec"] = options.Sec
	}
	return ret
}

func (options BindOptions) Validate(isPreexisting bool) error {
	missingKeys := []string{}
	if !isPreexisting {
		if options.FileShareName == "" {
			missingKeys = append(missingKeys, "share")
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("Missing required parameters: %s", strings.Join(missingKeys, ", "))
	}
	return nil
}

type staticState struct {
	ServiceName string `json:"service_name"`
	ServiceID   string `json:"service_id"`
}

type FileShare struct {
	InstanceID      string `json:"instance_id"`
	FileShareName   string `json:"file_share_name"`
	IsCreated       bool   `json:"is_created"` // true if it is created by the broker.
	Count           int    `json:"count"`
	URL             string `json:"url"`
	DatabaseVersion string `json:"database_version"`
}

func getFileShareID(instanceID, fileShareName string) string {
	return fmt.Sprintf("%s-%s", instanceID, fileShareName)
}

type ServiceInstance struct {
	ServiceID               string `json:"service_id"`
	PlanID                  string `json:"plan_id"`
	OrganizationGUID        string `json:"organization_guid"`
	SpaceGUID               string `json:"space_guid"`
	TargetName              string `json:"target_name"`    // AzureFileShare: StorageAccountName; Preexisting shares: Share URL
	IsPreexisting           bool   `json:"is_preexisting"` // True when preexisting shares are used; False when AzureFileShare is used.
	SubscriptionID          string `json:"subscription_id"`
	ResourceGroupName       string `json:"resource_group_name"`
	UseHTTPS                string `json:"use_https"`
	IsCreatedStorageAccount bool   `json:"is_created_storage_account"`
	OperationURL            string `json:"operation_url"`
	DatabaseVersion         string `json:"database_version"`
}

type lock interface {
	Lock()
	Unlock()
}

type Broker struct {
	logger lager.Logger
	mutex  lock
	clock  clock.Clock
	static staticState
	store  Store
	config Config
}

func New(
	logger lager.Logger,
	serviceName, serviceID string,
	clock clock.Clock,
	store Store,
	config *Config,
) *Broker {
	theBroker := Broker{
		logger: logger,
		mutex:  &sync.Mutex{},
		clock:  clock,
		static: staticState{
			ServiceName: serviceName,
			ServiceID:   serviceID,
		},
		store:  store,
		config: *config,
	}

	return &theBroker
}

func (b *Broker) isSupportAzureFileShare() bool {
	return b.config.cloud.Azure.IsSupportAzureFileShare()
}

func (b *Broker) Services(_ context.Context) []brokerapi.Service {
	logger := b.logger.Session("services")
	logger.Info("start")
	defer logger.Info("end")

	var plans []brokerapi.ServicePlan
	if b.isSupportAzureFileShare() {
		plans = []brokerapi.ServicePlan{
			{
				Name:        "Existing",
				ID:          "06948cb0-cad7-4buh-leba-9ed8b5c345a1",
				Description: "A preexisting filesystem",
			},
			{
				Name:        "AzureFileShare",
				ID:          "06948cb0-cad7-4buh-leba-9ed8b5c345a2",
				Description: "An Azure File Share filesystem",
			},
		}
	} else {
		plans = []brokerapi.ServicePlan{
			{
				Name:        "Existing",
				ID:          "06948cb0-cad7-4buh-leba-9ed8b5c345a1",
				Description: "A preexisting filesystem",
			},
		}
	}

	return []brokerapi.Service{{
		ID:            b.static.ServiceID,
		Name:          b.static.ServiceName,
		Description:   "SMB volumes (see: https://github.com/cloudfoundry/smb-volume-release/)",
		Bindable:      true,
		PlanUpdatable: false,
		Tags:          []string{"azurefile", "smb"},
		Requires:      []brokerapi.RequiredPermission{permissionVolumeMount},
		Plans:         plans,
	}}
}

// Provision Create a service instance which is mapped to a storage account or preexisting shares
// For AzureFileShare: UseHTTPS must be set to false. Otherwise, the mount in Linux will fail. https://docs.microsoft.com/en-us/azure/storage/storage-security-guide
func (b *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (_ brokerapi.ProvisionedServiceSpec, e error) {
	logger := b.logger.Session("provision").WithData(lager.Data{"instanceID": instanceID, "details": details, "asyncAllowed": asyncAllowed})
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	var configuration Configuration
	var decoder = json.NewDecoder(bytes.NewBuffer(details.RawParameters))
	if err := decoder.Decode(&configuration); err != nil {
		logger.Error("decode-configuration", err)
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrRawParamsInvalid
	}

	if !b.isSupportAzureFileShare() && configuration.Share == "" {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("Missing required parameters: share")
	}

	if configuration.Share != "" {
		// Provisiong preexisting shares
		serviceInstance := ServiceInstance{
			ServiceID:        details.ServiceID,
			PlanID:           details.PlanID,
			OrganizationGUID: details.OrganizationGUID,
			SpaceGUID:        details.SpaceGUID,
			TargetName:       configuration.Share,
			IsPreexisting:    true,
		}

		if err := b.store.CreateServiceInstance(instanceID, serviceInstance); err != nil {
			logger.Error("create-service-instance", err, lager.Data{"serviceInstance": serviceInstance})
			return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("Failed to store instance details %q: %s", instanceID, err)
		}

		logger.Debug("service-instance-created", lager.Data{"serviceInstance": serviceInstance})

		return brokerapi.ProvisionedServiceSpec{IsAsync: false}, nil
	}

	// Provisioning an Azure file share
	if configuration.SubscriptionID == "" {
		configuration.SubscriptionID = b.config.cloud.Azure.DefaultSubscriptionID
	}
	if configuration.ResourceGroupName == "" {
		configuration.ResourceGroupName = b.config.cloud.Azure.DefaultResourceGroupName
	}
	// Do not check location in this function because location is only used when the storage account does not exist
	if configuration.Location == "" {
		configuration.Location = b.config.cloud.Azure.DefaultLocation
	}

	if err := configuration.ValidateForAzureFileShare(); err != nil {
		logger.Error("validate-configuration", err)
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	storageAccount, err := b.getStorageAccount(logger, configuration)
	if err != nil {
		logger.Error("get-storage-account", err)
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	serviceInstance := ServiceInstance{
		ServiceID:               details.ServiceID,
		PlanID:                  details.PlanID,
		OrganizationGUID:        details.OrganizationGUID,
		SpaceGUID:               details.SpaceGUID,
		TargetName:              storageAccount.StorageAccountName,
		IsPreexisting:           false,
		SubscriptionID:          storageAccount.SubscriptionID,
		ResourceGroupName:       storageAccount.ResourceGroupName,
		UseHTTPS:                strconv.FormatBool(storageAccount.UseHTTPS),
		IsCreatedStorageAccount: storageAccount.IsCreatedStorageAccount,
		OperationURL:            storageAccount.OperationURL,
		DatabaseVersion:         databaseVersion,
	}

	err = b.store.CreateServiceInstance(instanceID, serviceInstance)
	if err != nil {
		logger.Error("create-service-instance", err, lager.Data{"serviceInstance": serviceInstance})
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("Failed to store instance details %q: %s", instanceID, err)
	}

	logger.Debug("service-instance-created", lager.Data{"serviceInstance": serviceInstance})

	isAsync := storageAccount.IsCreatedStorageAccount && storageAccount.OperationURL != ""
	return brokerapi.ProvisionedServiceSpec{IsAsync: isAsync, OperationData: storageAccount.OperationURL}, nil
}

func (b *Broker) getStorageAccount(logger lager.Logger, configuration Configuration) (*StorageAccount, error) {
	logger = logger.Session("get-storage-account")
	logger.Info("start")
	defer logger.Info("end")

	storageAccount, err := NewStorageAccount(logger, configuration)
	if err != nil {
		return nil, err
	}
	storageAccount.SDKClient, err = NewAzureStorageAccountSDKClient(
		logger,
		&b.config.cloud,
		storageAccount,
	)
	if err != nil {
		return nil, err
	}

	// Consider multiple users may send provision requests with a same storage account name
	// Multiple broker instances may check whether the storage account exists or not at the same time
	// They will send same creation requests to Azure if all of above checks return false
	// All of them will consider they are the owner of the new created storage account
	// We use a global lock as a solution for above race
	err = b.store.GetLockForUpdate(storageAccount.StorageAccountName, lockTimeoutInSeconds)
	if err != nil {
		logger.Error("get-lock-for-check-storage-account", err)
		return nil, err
	}
	defer b.store.ReleaseLockForUpdate(storageAccount.StorageAccountName)

	if exist, err := storageAccount.SDKClient.Exists(); err != nil {
		return nil, fmt.Errorf("Failed to check whether storage account exists: %v", err)
	} else if exist {
		logger.Debug("check-storage-account-exist", lager.Data{
			"message": fmt.Sprintf("The storage account %q exists.", storageAccount.StorageAccountName),
		})
		return storageAccount, nil
	} else if !b.config.cloud.Control.AllowCreateStorageAccount {
		return nil, fmt.Errorf("The storage account %q does not exist under the resource group %q in the subscription %q and the administrator does not allow to create it automatically", storageAccount.StorageAccountName, storageAccount.ResourceGroupName, storageAccount.SubscriptionID)
	}

	restClient, err := NewAzureStorageAccountRESTClient(
		logger,
		&b.config.cloud,
		storageAccount,
	)
	if err != nil {
		return nil, err
	}

	storageAccount.OperationURL, err = restClient.CreateStorageAccount()
	if err != nil {
		return nil, fmt.Errorf("Failed to create the storage account %q under the resource group %q in the subscription %q: %v", storageAccount.StorageAccountName, storageAccount.ResourceGroupName, storageAccount.SubscriptionID, err)
	}

	storageAccount.IsCreatedStorageAccount = true

	return storageAccount, nil
}

func (b *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (_ brokerapi.DeprovisionServiceSpec, e error) {
	logger := b.logger.Session("deprovision").WithData(lager.Data{"instanceID": instanceID, "details": details, "asyncAllowed": asyncAllowed})
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	serviceInstance, err := b.store.RetrieveServiceInstance(instanceID)
	if err != nil {
		logger.Error("retrieve-service-instance", err)
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrInstanceDoesNotExist
	}

	if !serviceInstance.IsPreexisting {
		if serviceInstance.IsCreatedStorageAccount && b.config.cloud.Control.AllowDeleteStorageAccount {
			storageAccount, err := NewStorageAccount(
				logger,
				Configuration{
					SubscriptionID:     serviceInstance.SubscriptionID,
					ResourceGroupName:  serviceInstance.ResourceGroupName,
					StorageAccountName: serviceInstance.TargetName,
					UseHTTPS:           serviceInstance.UseHTTPS,
				})
			if err != nil {
				return brokerapi.DeprovisionServiceSpec{}, err
			}
			storageAccount.SDKClient, err = NewAzureStorageAccountSDKClient(
				logger,
				&b.config.cloud,
				storageAccount,
			)
			if err != nil {
				return brokerapi.DeprovisionServiceSpec{}, err
			}
			if ok, err := storageAccount.SDKClient.Exists(); err != nil {
				return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("Failed to delete the storage account %q under the resource group %q in the subscription %q: %v", serviceInstance.TargetName, serviceInstance.ResourceGroupName, serviceInstance.SubscriptionID, err)
			} else if ok {
				if err := storageAccount.SDKClient.DeleteStorageAccount(); err != nil {
					return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("Failed to delete the storage account %q under the resource group %q in the subscription %q: %v", serviceInstance.TargetName, serviceInstance.ResourceGroupName, serviceInstance.SubscriptionID, err)
				}
			}
		}
	}

	err = b.store.DeleteServiceInstance(instanceID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	logger.Debug("service-instance-deleted", lager.Data{"serviceInstance": serviceInstance})

	return brokerapi.DeprovisionServiceSpec{IsAsync: false, OperationData: "deprovision"}, nil
}

func (b *Broker) Bind(context context.Context, instanceID string, bindingID string, details brokerapi.BindDetails) (_ brokerapi.Binding, e error) {
	logger := b.logger.Session("bind").WithData(lager.Data{"instanceID": instanceID, "bindingID": bindingID, "details": details})
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	serviceInstance, err := b.store.RetrieveServiceInstance(instanceID)
	if err != nil {
		err := brokerapi.ErrInstanceDoesNotExist
		logger.Error("retrieve-service-instance", err)
		return brokerapi.Binding{}, err
	}

	if details.AppGUID == "" {
		err := brokerapi.ErrAppGuidNotProvided
		logger.Error("missing-app-guid-parameter", err)
		return brokerapi.Binding{}, err
	}

	var bindOptions BindOptions
	var decoder = json.NewDecoder(bytes.NewBuffer(details.RawParameters))
	if err := decoder.Decode(&bindOptions); err != nil {
		logger.Error("decode-bind-raw-parameters", err, lager.Data{
			"RawParameters:": details.RawParameters,
		})
		return brokerapi.Binding{}, brokerapi.ErrRawParamsInvalid
	}
	if err := bindOptions.Validate(serviceInstance.IsPreexisting); err != nil {
		logger.Error("validate-bind-parameters", err)
		return brokerapi.Binding{}, err
	}

	globalMountConfig := b.config.mount.Copy()
	if err := globalMountConfig.SetEntries(bindOptions.ToMap()); err != nil {
		logger.Error("set-mount-entries", err, lager.Data{
			"bindOptions": bindOptions,
			"mount":       globalMountConfig.MakeConfig(),
		})
		return brokerapi.Binding{}, err
	}

	mountConfig := globalMountConfig.MakeConfig()
	var source, username, password string

	if serviceInstance.IsPreexisting {
		// Bind for preexisting shares
		if bindOptions.Domain != "" {
			mountConfig["domain"] = bindOptions.Domain
		}
		source = serviceInstance.TargetName
		username = bindOptions.Username
		password = bindOptions.Password
	} else {
		// Bind for AzureFileShare
		fileShareName := bindOptions.FileShareName

		fileShareID := getFileShareID(instanceID, fileShareName)
		err = b.store.GetLockForUpdate(fileShareID, lockTimeoutInSeconds)
		if err != nil {
			logger.Error("get-lock-for-update", err)
			return brokerapi.Binding{}, err
		}
		defer b.store.ReleaseLockForUpdate(fileShareID)

		fileShare, err := b.store.RetrieveFileShare(fileShareID)
		if err != nil {
			if err != brokerapi.ErrInstanceDoesNotExist {
				logger.Error("retrieve-file-share", err)
				return brokerapi.Binding{}, err
			}

			logger.Info("retrieve-file-share", lager.Data{"message": fmt.Sprintf("%s does not exist", fileShareID)})
			fileShare = FileShare{
				InstanceID:      instanceID,
				FileShareName:   fileShareName,
				IsCreated:       false,
				Count:           0,
				URL:             "",
				DatabaseVersion: databaseVersion,
			}
			err = nil
		}
		storageAccount, err := b.handleBindShare(logger, &serviceInstance, &fileShare)
		if err != nil {
			return brokerapi.Binding{}, err
		}

		if fileShare.Count == 1 {
			logger.Info("inserting-file-share-into-store", lager.Data{"fileShare": fileShare})
			if err := b.store.CreateFileShare(fileShareID, fileShare); err != nil {
				err = fmt.Errorf("Faied to insert file share into the store for %q: %v", fileShareID, err)
				logger.Error("insert-file-share-into-store", err)
				return brokerapi.Binding{}, err
			}
			logger.Info("inserted-file-share-into-store", lager.Data{"fileShare": fileShare})
		} else {
			logger.Info("updating-file-share-in-store", lager.Data{"fileShare": fileShare})
			if err := b.store.UpdateFileShare(fileShareID, fileShare); err != nil {
				err = fmt.Errorf("Faied to update file share in the store for %q: %v", fileShareID, err)
				logger.Error("update-file-share-in-store", err)
				return brokerapi.Binding{}, err
			}
			logger.Info("updated-file-share-in-store", lager.Data{"fileShare": fileShare})
		}

		source = fileShare.URL
		username = serviceInstance.TargetName
		password, err = storageAccount.SDKClient.GetAccessKey()
		if err != nil {
			return brokerapi.Binding{}, err
		}
	}

	err = b.store.CreateBindingDetails(bindingID, details, serviceInstance.IsPreexisting)
	if err != nil {
		logger.Error("create-binding-details", err)
		return brokerapi.Binding{}, err
	}

	logger.Info("binding-details-created")

	mountConfig["source"] = source
	mountConfig["username"] = username
	logger.Debug("volume-service-binding", lager.Data{"driver": "smbdriver", "mountConfig": mountConfig, "source": source})

	s, err := b.hash(mountConfig)
	if err != nil {
		logger.Error("error-calculating-volume-id", err, lager.Data{"config": mountConfig})
		return brokerapi.Binding{}, err
	}

	if password != "" {
		mountConfig["password"] = password
	}
	volumeID := fmt.Sprintf("%s-%s", instanceID, s)

	ret := brokerapi.Binding{
		Credentials: struct{}{}, // if nil, cloud controller chokes on response
		VolumeMounts: []brokerapi.VolumeMount{{
			ContainerDir: evaluateContainerPath(bindOptions, instanceID),
			Mode:         readOnlyToMode(bindOptions.Readonly),
			Driver:       driverName,
			DeviceType:   deviceTypeShared,
			Device: brokerapi.SharedDevice{
				VolumeId:    volumeID,
				MountConfig: mountConfig,
			},
		}},
	}

	return ret, nil
}

func (b *Broker) handleBindShare(logger lager.Logger, serviceInstance *ServiceInstance, share *FileShare) (*StorageAccount, error) {
	logger = logger.Session("handle-bind-share").WithData(lager.Data{"FileShareName": share.FileShareName})
	logger.Info("start")
	defer logger.Info("end")

	storageAccount, err := NewStorageAccount(
		logger,
		Configuration{
			SubscriptionID:     serviceInstance.SubscriptionID,
			ResourceGroupName:  serviceInstance.ResourceGroupName,
			StorageAccountName: serviceInstance.TargetName,
			UseHTTPS:           serviceInstance.UseHTTPS,
		})
	if err != nil {
		return nil, err
	}
	storageAccount.SDKClient, err = NewAzureStorageAccountSDKClient(
		logger,
		&b.config.cloud,
		storageAccount,
	)
	if err != nil {
		return nil, err
	}

	exist, err := storageAccount.SDKClient.HasFileShare(share.FileShareName)
	if err != nil {
		return nil, fmt.Errorf("Failed to check whether the file share %q exists: %v", share.FileShareName, err)
	}

	if exist {
		share.Count++
		if share.URL == "" {
			shareURL, err := storageAccount.SDKClient.GetShareURL(share.FileShareName)
			if err != nil {
				return nil, err
			}
			share.URL = shareURL
		}
		logger.Debug("file-share-get", lager.Data{"share": share})
	} else {
		if !b.config.cloud.Control.AllowCreateFileShare {
			return nil, fmt.Errorf("The file share %q does not exist in the storage account %q and the administrator does not allow to create it automatically", share.FileShareName, storageAccount.StorageAccountName)
		}
		if err := storageAccount.SDKClient.CreateFileShare(share.FileShareName); err != nil {
			return nil, fmt.Errorf("Failed to create file share %q in the storage account %q: %v", share.FileShareName, storageAccount.StorageAccountName, err)
		}
		share.IsCreated = true
		share.Count = 1
		shareURL, err := storageAccount.SDKClient.GetShareURL(share.FileShareName)
		if err != nil {
			return nil, err
		}
		share.URL = shareURL
		logger.Debug("file-share-created", lager.Data{"share": share})
	}

	return storageAccount, nil
}

func (b *Broker) hash(mountConfig map[string]interface{}) (string, error) {
	var (
		bytes []byte
		err   error
	)
	if bytes, err = json.Marshal(mountConfig); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(bytes)), nil
}

func (b *Broker) Unbind(context context.Context, instanceID string, bindingID string, details brokerapi.UnbindDetails) (e error) {
	logger := b.logger.Session("unbind").WithData(lager.Data{"instanceID": instanceID, "bindingID": bindingID, "details": details})
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	serviceInstance, err := b.store.RetrieveServiceInstance(instanceID)
	if err != nil {
		logger.Error("retrieve-service-instance", err)
		return brokerapi.ErrInstanceDoesNotExist
	}
	bindDetails, err := b.store.RetrieveBindingDetails(bindingID)
	if err != nil {
		logger.Error("retrieve-binding-details", err)
		return brokerapi.ErrBindingDoesNotExist
	}

	if !serviceInstance.IsPreexisting {
		var bindOptions BindOptions
		var decoder = json.NewDecoder(bytes.NewBuffer(bindDetails.RawParameters))
		if err := decoder.Decode(&bindOptions); err != nil {
			logger.Error("decode-bind-raw-parameters", err)
			return brokerapi.ErrRawParamsInvalid
		}
		fileShareName := bindOptions.FileShareName

		fileShareID := getFileShareID(instanceID, fileShareName)
		err = b.store.GetLockForUpdate(fileShareID, lockTimeoutInSeconds)
		if err != nil {
			logger.Error("get-lock-for-update", err)
			return err
		}
		defer b.store.ReleaseLockForUpdate(fileShareID)

		fileShare, err := b.store.RetrieveFileShare(fileShareID)
		if err != nil {
			logger.Error("retrieve-file-share", err)
			return err
		}

		if err := b.handleUnbindShare(logger, &serviceInstance, &fileShare); err != nil {
			return err
		}

		if fileShare.Count > 0 {
			logger.Debug("updating-file-share-in-store", lager.Data{"fileShare": fileShare})
			if err := b.store.UpdateFileShare(fileShareID, fileShare); err != nil {
				err = fmt.Errorf("Faied to update file share in the store for %q: %v", fileShareID, err)
				logger.Error("update-file-share-in-store", err)
				return err
			}
			logger.Debug("updated-file-share-in-store", lager.Data{"fileShare": fileShare})
		} else {
			logger.Debug("deleting-file-share-from-store", lager.Data{"fileShare": fileShare})
			if err := b.store.DeleteFileShare(fileShareID); err != nil {
				err = fmt.Errorf("Faied to delete file share from the store for %q: %v", fileShareID, err)
				logger.Error("delete-file-share-from-store", err)
				return err
			}
			logger.Debug("deleted-file-share-from-store", lager.Data{"fileShare": fileShare})
		}
	}

	if err := b.store.DeleteBindingDetails(bindingID); err != nil {
		return err
	}

	return nil
}

func (b *Broker) handleUnbindShare(logger lager.Logger, serviceInstance *ServiceInstance, share *FileShare) error {
	logger = logger.Session("handle-unbind-share").WithData(lager.Data{"FileShareName": share.FileShareName})
	logger.Info("start")
	defer logger.Info("end")

	share.Count--
	if share.Count > 0 {
		return nil
	}

	if share.IsCreated && b.config.cloud.Control.AllowDeleteFileShare {
		storageAccount, err := NewStorageAccount(
			logger,
			Configuration{
				SubscriptionID:     serviceInstance.SubscriptionID,
				ResourceGroupName:  serviceInstance.ResourceGroupName,
				StorageAccountName: serviceInstance.TargetName,
				UseHTTPS:           serviceInstance.UseHTTPS,
			})
		if err != nil {
			return err
		}
		storageAccount.SDKClient, err = NewAzureStorageAccountSDKClient(
			logger,
			&b.config.cloud,
			storageAccount,
		)
		if err != nil {
			return err
		}

		if err := storageAccount.SDKClient.DeleteFileShare(share.FileShareName); err != nil {
			return fmt.Errorf("Faied to delete the file share %q in the storage account %q: %v", share.FileShareName, serviceInstance.TargetName, err)
		}
	}

	return nil
}

func (b *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	panic("not implemented")
}

func (b *Broker) LastOperation(_ context.Context, instanceID string, operationData string) (brokerapi.LastOperation, error) {
	logger := b.logger.Session("last-operation").WithData(lager.Data{"instanceID": instanceID})
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if operationData == "" {
		return brokerapi.LastOperation{}, errors.New("unrecognized operationData")
	}

	serviceInstance, err := b.store.RetrieveServiceInstance(instanceID)
	if err != nil {
		err := brokerapi.ErrInstanceDoesNotExist
		logger.Error("retrieve-service-instance", err)
		return brokerapi.LastOperation{}, err
	}

	if serviceInstance.IsPreexisting {
		return brokerapi.LastOperation{}, errors.New("LastOperation cannot be called for preexisting shares")
	}

	storageAccount, err := NewStorageAccount(
		logger,
		Configuration{
			SubscriptionID:     serviceInstance.SubscriptionID,
			ResourceGroupName:  serviceInstance.ResourceGroupName,
			StorageAccountName: serviceInstance.TargetName,
			UseHTTPS:           serviceInstance.UseHTTPS,
		})
	if err != nil {
		return brokerapi.LastOperation{}, err
	}
	restClient, err := NewAzureStorageAccountRESTClient(
		logger,
		&b.config.cloud,
		storageAccount,
	)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}
	ret, err := restClient.CheckCompletion(operationData)
	state := brokerapi.InProgress
	description := ""
	if err != nil {
		state = brokerapi.Failed
		description = err.Error()
	} else if ret {
		state = brokerapi.Succeeded
	}

	return brokerapi.LastOperation{State: state, Description: description}, nil
}

func readOnlyToMode(ro bool) string {
	if ro {
		return "r"
	}
	return "rw"
}

func evaluateContainerPath(options BindOptions, volID string) string {
	if options.Mount != "" {
		return options.Mount
	}

	return path.Join(defaultContainerPath, volID)
}
