package azurefilebroker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	resty "gopkg.in/resty.v0"

	"code.cloudfoundry.org/lager"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	file "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

const (
	creator                     = "creator"
	resourceNotFound            = "StatusCode=404"
	fileRequestTimeoutInSeconds = 60
)

const (
	userAgent              = "azurefilebroker"
	restAPIProviderStorage = "Microsoft.Storage"
	restAPIStorageAccounts = "storageAccounts"
	restAPIStorageKind     = "Storage"
	contentTypeJSON        = "application/json"
	contentTypeWWW         = "application/x-www-form-urlencoded"
)

var (
	restRetryCodes = []int{408, 429, 500, 502, 503, 504}
)

const (
	AzureCloud        = "AzureCloud"
	AzureChinaCloud   = "AzureChinaCloud"
	AzureGermanCloud  = "AzureGermanCloud"
	AzureUSGovernment = "AzureUSGovernment"
	AzureStack        = "AzureStack"
)

type APIVersions struct {
	StorageForREST  string
	StorageForSDK   string
	ActiveDirectory string
}

type Environment struct {
	ResourceManagerEndpointURL string
	ActiveDirectoryEndpointURL string
	APIVersions                APIVersions
}

var Environments = map[string]Environment{
	AzureCloud: Environment{
		ResourceManagerEndpointURL: "https://management.azure.com/",
		ActiveDirectoryEndpointURL: "https://login.microsoftonline.com",
		APIVersions: APIVersions{
			StorageForREST:  "2016-12-01",
			StorageForSDK:   "2016-05-31",
			ActiveDirectory: "2015-06-15",
		},
	},
	AzureChinaCloud: Environment{
		ResourceManagerEndpointURL: "https://management.chinacloudapi.cn/",
		ActiveDirectoryEndpointURL: "https://login.chinacloudapi.cn",
		APIVersions: APIVersions{
			StorageForREST:  "2016-12-01",
			StorageForSDK:   "2016-05-31",
			ActiveDirectory: "2015-06-15",
		},
	},
	AzureUSGovernment: Environment{
		ResourceManagerEndpointURL: "https://management.usgovcloudapi.net/",
		ActiveDirectoryEndpointURL: "https://login.microsoftonline.com",
		APIVersions: APIVersions{
			StorageForREST:  "2016-12-01",
			StorageForSDK:   "2016-05-31",
			ActiveDirectory: "2015-06-15",
		},
	},
	AzureGermanCloud: Environment{
		ResourceManagerEndpointURL: "https://management.microsoftazure.de/",
		ActiveDirectoryEndpointURL: "https://login.microsoftonline.de",
		APIVersions: APIVersions{
			StorageForREST:  "2016-12-01",
			StorageForSDK:   "2016-05-31",
			ActiveDirectory: "2015-06-15",
		},
	},
	AzureStack: Environment{
		APIVersions: APIVersions{
			StorageForREST:  "2016-12-01",
			StorageForSDK:   "2016-05-31",
			ActiveDirectory: "2015-06-15",
		},
	},
}

//go:generate counterfeiter -o ../azurefilebrokerfakes/fake_azure_storage_account_sdk_client.go . AzureStorageAccountSDKClient
type AzureStorageAccountSDKClient interface {
	Exists() (bool, error)
	GetAccessKey() (string, error)
	DeleteStorageAccount() error
	HasFileShare(fileShareName string) (bool, error)
	CreateFileShare(fileShareName string) error
	DeleteFileShare(fileShareName string) error
	GetShareURL(fileShareName string) (string, error)
}

//go:generate counterfeiter -o ../azurefilebrokerfakes/fake_azure_storage_account_rest_client.go . AzureStorageAccountRESTClient
type AzureStorageAccountRESTClient interface {
	CreateStorageAccount() (string, error)
	CheckCompletion(asyncURL string) (bool, error)
}

type StorageAccount struct {
	SubscriptionID          string
	ResourceGroupName       string
	StorageAccountName      string
	UseHTTPS                bool
	EnableEncryption        bool
	SkuName                 storage.SkuName
	Location                string
	IsCreatedStorageAccount bool
	AccessKey               string
	BaseURL                 string
	OperationURL            string
	SDKClient               AzureStorageAccountSDKClient
}

func NewStorageAccount(logger lager.Logger, configuration Configuration) (*StorageAccount, error) {
	logger = logger.Session("storage-account").WithData(lager.Data{"StorageAccountName": configuration.StorageAccountName})
	storageAccount := StorageAccount{
		SubscriptionID:          configuration.SubscriptionID,
		ResourceGroupName:       configuration.ResourceGroupName,
		StorageAccountName:      configuration.StorageAccountName,
		SkuName:                 storage.StandardRAGRS,
		Location:                configuration.Location,
		UseHTTPS:                false,
		EnableEncryption:        true,
		IsCreatedStorageAccount: false,
		SDKClient:               nil,
	}

	if configuration.UseHTTPS != "" {
		if ret, err := strconv.ParseBool(configuration.UseHTTPS); err == nil {
			storageAccount.UseHTTPS = ret
		} else {
			return nil, fmt.Errorf("Failed in parsing UseHTTPS. It must be true or false. Error: %v", err)
		}
	}

	if configuration.EnableEncryption != "" {
		if ret, err := strconv.ParseBool(configuration.EnableEncryption); err == nil {
			storageAccount.EnableEncryption = ret
		} else {
			return nil, fmt.Errorf("Failed in parsing EnableEncryption. It must be true or false. Error: %v", err)
		}
	}
	if configuration.SkuName != "" {
		storageAccount.SkuName = storage.SkuName(configuration.SkuName)
		if storageAccount.SkuName != storage.StandardGRS && storageAccount.SkuName != storage.StandardLRS && storageAccount.SkuName != storage.StandardRAGRS {
			err := fmt.Errorf("The SkuName %q to create the storage account is invalid. It must be Standard_GRS, Standard_LRS or Standard_RAGRS", configuration.SkuName)
			logger.Error("check-sku-name", err)
			return nil, err
		}
	}
	if configuration.Location != "" {
		storageAccount.Location = configuration.Location
	}

	return &storageAccount, nil
}

type AzureStorageSDKClient struct {
	logger                   lager.Logger
	cloudConfig              *CloudConfig
	StorageAccount           *StorageAccount
	storageManagementClient  *storage.AccountsClient
	storageFileServiceClient *file.Client
}

func NewAzureStorageAccountSDKClient(logger lager.Logger, cloudConfig *CloudConfig, storageAccount *StorageAccount) (AzureStorageAccountSDKClient, error) {
	logger = logger.Session("storage-sdk-client").WithData(lager.Data{"ResourceGroupName": storageAccount.ResourceGroupName, "StorageAccountName": storageAccount.StorageAccountName})
	connection := AzureStorageSDKClient{
		logger:                   logger,
		cloudConfig:              cloudConfig,
		StorageAccount:           storageAccount,
		storageManagementClient:  nil,
		storageFileServiceClient: nil,
	}
	if err := connection.initManagementClient(); err != nil {
		return nil, err
	}
	return &connection, nil
}

func (c *AzureStorageSDKClient) initManagementClient() error {
	logger := c.logger.Session("init-management-client")
	logger.Info("start")
	defer logger.Info("end")

	environment := c.cloudConfig.Azure.Environment
	tenantID := c.cloudConfig.Azure.TenanID
	clientID := c.cloudConfig.Azure.ClientID
	clientSecret := c.cloudConfig.Azure.ClientSecret
	oauthConfig, err := adal.NewOAuthConfig(Environments[environment].ActiveDirectoryEndpointURL, tenantID)
	if err != nil {
		logger.Error("newO-auth-config", err, lager.Data{
			"Environment":                environment,
			"ActiveDirectoryEndpointURL": Environments[environment].ActiveDirectoryEndpointURL,
			"TenanID":                    tenantID,
		})
		return fmt.Errorf("Error in initManagementClient: %v", err)
	}

	resourceManagerEndpointURL := Environments[environment].ResourceManagerEndpointURL
	spt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, resourceManagerEndpointURL)
	if err != nil {
		logger.Error("newO-service-principal-token", err, lager.Data{
			"Environment":                environment,
			"resourceManagerEndpointURL": resourceManagerEndpointURL,
			"TenanID":                    tenantID,
			"ClientID":                   clientID,
		})
		return fmt.Errorf("Error in initManagementClient: %v", err)
	}

	client := storage.NewAccountsClientWithBaseURI(resourceManagerEndpointURL, c.StorageAccount.SubscriptionID)
	c.storageManagementClient = &client
	c.storageManagementClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	return nil
}

func (c *AzureStorageSDKClient) Exists() (bool, error) {
	logger := c.logger.Session("exists")
	logger.Info("start")
	defer logger.Info("end")

	if _, err := c.getStorageAccountProperties(); err != nil {
		if strings.Contains(err.Error(), resourceNotFound) {
			err = nil
		}
		return false, err
	}
	return true, nil
}

func (c *AzureStorageSDKClient) getBaseURL() error {
	logger := c.logger.Session("get-base-url")
	logger.Info("start")
	defer logger.Info("end")

	result, err := c.getStorageAccountProperties()
	if err != nil {
		logger.Error("get-storage-account-properties", err)
		return err
	}
	properties := *result.AccountProperties
	if properties.ProvisioningState != storage.Succeeded {
		err := fmt.Errorf("The storage account %q is still in creating", c.StorageAccount.StorageAccountName)
		logger.Error("get-storage-account-properties", err)
		return err
	}
	c.StorageAccount.BaseURL, err = parseBaseURL(*(properties.PrimaryEndpoints).File)
	if err != nil {
		logger.Error("parse-base-url", err)
		return err
	}

	return nil
}

func (c *AzureStorageSDKClient) getStorageAccountProperties() (storage.Account, error) {
	logger := c.logger.Session("get-storage-account-properties")
	logger.Info("start")
	defer logger.Info("end")

	result, err := c.storageManagementClient.GetProperties(c.StorageAccount.ResourceGroupName, c.StorageAccount.StorageAccountName)
	return result, err
}

func parseBaseURL(fileEndpoint string) (string, error) {
	re := regexp.MustCompile(`http[s]?://([^\.]*)\.([^\.]*)\.([^/]*).*`)
	result := re.FindStringSubmatch(fileEndpoint)
	if len(result) != 4 {
		return "", fmt.Errorf("Error in parsing baseURL from fileEndpoint: %q", fileEndpoint)
	}
	return result[3], nil
}

func (c *AzureStorageSDKClient) GetAccessKey() (string, error) {
	logger := c.logger.Session("get-access-key")
	logger.Info("start")
	defer logger.Info("end")

	if c.StorageAccount.AccessKey == "" {
		result, err := c.storageManagementClient.ListKeys(c.StorageAccount.ResourceGroupName, c.StorageAccount.StorageAccountName)
		if err != nil {
			logger.Error("list-keys", err)
			return "", fmt.Errorf("Failed to list keys: %v", err)
		}
		c.StorageAccount.AccessKey = *(*result.Keys)[0].Value
	}
	return c.StorageAccount.AccessKey, nil
}

func (c *AzureStorageSDKClient) DeleteStorageAccount() error {
	logger := c.logger.Session("delete-storage-account")
	logger.Info("start")
	defer logger.Info("end")

	_, err := c.storageManagementClient.Delete(c.StorageAccount.ResourceGroupName, c.StorageAccount.StorageAccountName)
	if err != nil {
		logger.Error("delete", err)
		return fmt.Errorf("Failed to list keys: %v", err)
	}
	return nil
}

func (c *AzureStorageSDKClient) initFileServiceClient() error {
	logger := c.logger.Session("init-file-service-client")
	logger.Info("start")
	defer logger.Info("end")

	if c.storageFileServiceClient != nil {
		return nil
	}

	if c.StorageAccount.AccessKey == "" {
		if _, err := c.GetAccessKey(); err != nil {
			return err
		}
	}

	if c.StorageAccount.BaseURL == "" {
		if err := c.getBaseURL(); err != nil {
			return err
		}
	}

	environment := c.cloudConfig.Azure.Environment
	apiVersion := Environments[environment].APIVersions.StorageForSDK
	client, err := file.NewClient(c.StorageAccount.StorageAccountName, c.StorageAccount.AccessKey, c.StorageAccount.BaseURL, apiVersion, c.StorageAccount.UseHTTPS)
	if err != nil {
		logger.Error("new-file-client", err, lager.Data{
			"baseURL":    c.StorageAccount.BaseURL,
			"apiVersion": apiVersion,
			"UseHTTPS":   c.StorageAccount.UseHTTPS,
		})
		return err
	}
	c.storageFileServiceClient = &client
	c.storageFileServiceClient.AddToUserAgent(userAgent)
	return nil
}

func (c *AzureStorageSDKClient) HasFileShare(fileShareName string) (bool, error) {
	logger := c.logger.Session("has-file-share").WithData(lager.Data{"FileShareName": fileShareName})
	logger.Info("start")
	defer logger.Info("end")

	if err := c.initFileServiceClient(); err != nil {
		return false, err
	}
	fileService := c.storageFileServiceClient.GetFileService()
	share := fileService.GetShareReference(fileShareName)
	exists, err := share.Exists()
	if err != nil {
		logger.Error("check-file-share-exists", err)
	}
	return exists, err
}

func (c *AzureStorageSDKClient) CreateFileShare(fileShareName string) error {
	logger := c.logger.Session("create-file-share").WithData(lager.Data{"FileShareName": fileShareName})
	logger.Info("start")
	defer logger.Info("end")

	if err := c.initFileServiceClient(); err != nil {
		return err
	}
	fileService := c.storageFileServiceClient.GetFileService()
	share := fileService.GetShareReference(fileShareName)
	options := file.FileRequestOptions{Timeout: fileRequestTimeoutInSeconds}
	err := share.Create(&options)
	if err != nil {
		logger.Error("create-file-share", err)
	}
	return err
}

func (c *AzureStorageSDKClient) DeleteFileShare(fileShareName string) error {
	logger := c.logger.Session("delete-file-share").WithData(lager.Data{"FileShareName": fileShareName})
	logger.Info("start")
	defer logger.Info("end")

	if err := c.initFileServiceClient(); err != nil {
		return err
	}
	fileService := c.storageFileServiceClient.GetFileService()
	share := fileService.GetShareReference(fileShareName)
	options := file.FileRequestOptions{Timeout: fileRequestTimeoutInSeconds}
	err := share.Delete(&options)
	if err != nil {
		// TBD: return nil when the share does not exist
		logger.Error("delete-file-share", err)
	}
	return err
}

func (c *AzureStorageSDKClient) GetShareURL(fileShareName string) (string, error) {
	logger := c.logger.Session("get-share-url").WithData(lager.Data{"FileShareName": fileShareName})
	logger.Info("start")
	defer logger.Info("end")

	if c.StorageAccount.BaseURL == "" {
		if err := c.getBaseURL(); err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("//%s.file.%s/%s", c.StorageAccount.StorageAccountName, c.StorageAccount.BaseURL, fileShareName), nil
}

type AzureToken struct {
	ExpiresOn   time.Time
	AccessToken string
}

type AzureRESTClient struct {
	logger         lager.Logger
	cloudConfig    *CloudConfig
	storageAccount *StorageAccount
	token          AzureToken
}

func NewAzureStorageAccountRESTClient(logger lager.Logger, cloudConfig *CloudConfig, storageAccount *StorageAccount) (AzureStorageAccountRESTClient, error) {
	logger = logger.Session("storage-account-rest-client").WithData(lager.Data{"StorageAccountName": storageAccount.StorageAccountName})
	client := AzureRESTClient{
		logger:         logger,
		cloudConfig:    cloudConfig,
		storageAccount: storageAccount,
	}
	return &client, nil
}

func (c *AzureRESTClient) refreshToken(force bool) error {
	if c.token.AccessToken == "" || time.Until(c.token.ExpiresOn) <= 0 || force {
		headers := map[string]string{
			"Content-Type": contentTypeWWW,
			"User-Agent":   userAgent,
		}

		hostURL := fmt.Sprintf("%s/%s/oauth2/token", Environments[c.cloudConfig.Azure.Environment].ActiveDirectoryEndpointURL, c.cloudConfig.Azure.TenanID)
		body := url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {c.cloudConfig.Azure.ClientID},
			"client_secret": {c.cloudConfig.Azure.ClientSecret},
			"resource":      {Environments[c.cloudConfig.Azure.Environment].ResourceManagerEndpointURL},
			"scope":         {"user_impersonation"},
		}

		resty.DefaultClient.SetRetryCount(3).SetRetryWaitTime(10)
		resp, err := resty.R().
			SetHeaders(headers).
			SetQueryParam("api-version", Environments[c.cloudConfig.Azure.Environment].APIVersions.ActiveDirectory).
			SetBody(body.Encode()).
			Post(hostURL)
		if err != nil {
			return err
		}
		if resp.StatusCode() == http.StatusOK {
			type ResponseBody struct {
				ExpiresOn   string `json:"expires_on"`
				AccessToken string `json:"access_token"`
			}
			responseBody := ResponseBody{}
			err := json.Unmarshal(resp.Body(), &responseBody)
			if err != nil {
				return err
			}
			expiresOn, err := strconv.ParseInt(responseBody.ExpiresOn, 10, 64)
			if err != nil {
				return err
			}
			c.token.ExpiresOn = time.Unix(expiresOn, 0)
			c.token.AccessToken = responseBody.AccessToken
		} else {
			return fmt.Errorf("HTTP CODE: %#v", resp.StatusCode())
		}
	}
	return nil
}

func (c *AzureRESTClient) initialize() (map[string]string, map[string]string, error) {
	resty.DefaultClient.SetRetryCount(3).SetRetryWaitTime(10)
	check := resty.RetryConditionFunc(func(r *resty.Response) (bool, error) {
		for _, v := range restRetryCodes {
			if r.StatusCode() == v {
				return true, nil
			}
		}
		return false, nil
	})
	resty.DefaultClient.AddRetryCondition(check)
	headers := map[string]string{
		"Content-Type": contentTypeJSON,
		"User-Agent":   userAgent,
	}
	queries := map[string]string{
		"api-version": Environments[c.cloudConfig.Azure.Environment].APIVersions.StorageForREST,
	}
	err := c.refreshToken(false)
	if err != nil {
		return nil, nil, err
	}

	return headers, queries, nil
}

// CreateStorageAccount Create a storage account. You need to call CheckCompletion to check whether the creation is finished.
// Return "", nil when the storage account has been created.
// Return "operation-url", nil when the storage account is still in creating.
// Reference: https://docs.microsoft.com/en-us/rest/api/storagerp/storageaccounts#StorageAccounts_Create
func (c *AzureRESTClient) CreateStorageAccount() (string, error) {
	headers, queries, err := c.initialize()
	if err != nil {
		return "", err
	}
	hostURL := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s",
		Environments[c.cloudConfig.Azure.Environment].ResourceManagerEndpointURL,
		c.storageAccount.SubscriptionID,
		c.storageAccount.ResourceGroupName,
		restAPIProviderStorage,
		restAPIStorageAccounts,
		c.storageAccount.StorageAccountName)

	tags := map[string]string{}
	tags["User-Agent"] = userAgent

	storageAccount := map[string]interface{}{
		"location": c.storageAccount.Location,
		"tags":     tags,
		"name":     c.storageAccount.StorageAccountName,
		"properties": map[string]interface{}{
			"supportsHttpsTrafficOnly": c.storageAccount.UseHTTPS,
			"encryption": map[string]interface{}{
				"services": map[string]interface{}{
					"blob": map[string]interface{}{
						"enabled": c.storageAccount.EnableEncryption,
					},
					"file": map[string]interface{}{
						"enabled": c.storageAccount.EnableEncryption,
					},
				},
				"keySource": restAPIProviderStorage,
			},
		},
		"sku": map[string]interface{}{
			"name": string(c.storageAccount.SkuName),
		},
		"kind": restAPIStorageKind,
	}
	body, err := json.Marshal(storageAccount)
	if err != nil {
		return "", err
	}

	resp, err := resty.R().
		SetHeaders(headers).
		SetQueryParams(queries).
		SetAuthToken(c.token.AccessToken).
		SetBody(body).
		Put(hostURL)
	if err != nil {
		return "", err
	}
	statusCode := resp.StatusCode()
	if statusCode == http.StatusOK {
		return "", nil
	} else if statusCode == http.StatusAccepted {
		return resp.Header().Get("Location"), nil
	}
	return "", fmt.Errorf("Error Code: %d, %v", statusCode, resp)
}

// CheckCompletion Check whether an asynchronous operation finishes or not
func (c *AzureRESTClient) CheckCompletion(asyncURL string) (bool, error) {
	headers, queries, err := c.initialize()
	if err != nil {
		return false, err
	}
	headers["x-ms-version"] = queries["api-version"]

	resp, err := resty.R().
		SetHeaders(headers).
		SetQueryParams(queries).
		SetAuthToken(c.token.AccessToken).
		Get(asyncURL)
	statusCode := resp.StatusCode()
	if statusCode == http.StatusAccepted {
		return false, nil
	} else if statusCode == http.StatusOK {
		return true, nil
	}
	apiResponse := map[string]interface{}{}
	if err := json.Unmarshal(resp.Body(), &apiResponse); err != nil {
		return false, fmt.Errorf("StatusCode: %d - %v\n\t%s", statusCode, resp, err)
	}
	return false, fmt.Errorf("StatusCode: %d - %s", statusCode, apiResponse)
}
