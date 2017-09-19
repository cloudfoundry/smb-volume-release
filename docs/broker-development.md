## Development

# Build azurefilebroker

```bash
./script/build-broker
```

# Run Unit Test

```bash
./scripts/run-broker-unit-tests
```

# Run Lifecycle Test

You may need to install `curl` and `jq` to run test. Before running test, you need to fill parameters in `./scripts/run-broker-lifecycle-tests` with yours.

```bash
./scripts/run-broker-lifecycle-tests
```

# Configurations of azurefilebroker

To start azurefilebroker, all configurations must start with `--`.

- Environment variables for Broker
    - USERNAME: [REQUIRED] - Username for your broker.
    - PASSWORD: [REQUIRED] - Password for your broker.
    - DB_USERNAME: Required when `cfServiceName` is not used. Username for the database which stores the state of your broker.
    - DB_PASSWORD: Required when `cfServiceName` is not used. Password for the database which stores the state of your broker.

- Configurations for Broker
    - listenAddr: host:port to serve service broker API. Default value is `0.0.0.0:9000`. You must use the environment variable `PORT` if you deploy broker as a Cloud Foundry application. Please reference [here](https://docs.run.pivotal.io/devguide/deploy-apps/environment-variable.html#PORT).
    - serviceName: Name of the service to register with cloud controller. Default value is `azuresmbvolume`.
    - serviceID: ID of the service to register with cloud controller. Default value is `06948cb0-cad7-4buh-leba-9ed8b5c345a3`.

- Configurations for database used by Broker
    - dbDriver: [REQUIRED] - Database driver name to use SQL to store broker state. Allowed values: `mssql` or `mysql`.
    - dbCACert: (optional) - Content of CA Cert to verify SSL connection.
    - hostNameInCertificate: (optional) - For Azure SQL service or Azure MySQL service, you need to specify one of below values to enable TLS encryption. For your certificate, you need to specify the Comman Name (CN) in the server certificate.
      - For AzureCloud: `*.database.windows.net`
      - For AzureUSGovernment: `*.database.usgovcloudapi.net`
      - For AzureChinaCloud: `*.database.chinacloudapi.cn`
      - For AzureGermanCloud: `*.database.cloudapi.de`
    - cfServiceName: (optional) - For CF pushed apps, the service name in VCAP_SERVICES where we should find database credentials. If this option is set, all db parameters will be extracted from the service binding except `dbCACert` and `hostNameInCertificate`. It must be set to the service name for the database service as seen in `cf marketplace` which you want to bind to this broker. In the `manifest.yml`, alias is `DBSERVICENAME` to keep same format as nfsbroker.
    - dbHostname: (optional) - Database hostname when using SQL to store broker state.
    - dbPort: (optional) - Database port when using SQL to store broker state.
    - dbName: (optional) - Database name when using SQL to store broker state.

- Configurations for bind
    - allowedOptions: A comma separated list of parameters allowed to be set in during bind operations. Default value is `share,uid,gid,file_mode,dir_mode,readonly,vers,mount`.
    - defaultOptions: A comma separated list of defaults specified as param:value. If a parameter has a default value and is not in the allowed list, this default value becomes a fixed value that cannot be overridden. Default value is `vers:3.0`.

- Configurations for Azure
    - environment: The environment for Azure Management Service. Allowed values: `AzureCloud`, `AzureChinaCloud`, `AzureUSGovernment` or `AzureGermanCloud`. Default value is `AzureCloud`.
    - tenantID: [REQUIRED] - The tenant id for your service principal.
    - clientID: [REQUIRED] - The client id for your service principal.
    - clientSecret: [REQUIRED] - The client secret for your service principal.
    - defaultSubscriptionID: (optional) - The default Azure Subscription id to use for storage accounts.
    - defaultResourceGroupName: (optional) - The default resource group name to use for storage accounts.
    - defaultLocation: (optional) - The default location to use for creating storage accounts.

    **NOTE:**

    - Please see more details about how to create a service principal [here](https://github.com/cloudfoundry-incubator/bosh-azure-cpi-release/blob/master/docs/get-started/create-service-principal.md).
    - `PORT` in Procfile will be allocated dynamically by Cloud Foundry runtime.

- Configurations for permission
    - allowCreateStorageAccount: Allow Broker to create storage accounts. Default value is `true`.
    - allowCreateFileShare: Allow Broker to create file shares. Default value is `true`.
    - allowDeleteStorageAccount: Allow Broker to delete storage accounts which are created by Broker. Default value is `false`.
    - allowDeleteFileShare: Allow Broker to delete file shares which are created by Broker. Default value is `false`.

    **NOTE:**

    - AzureStack does not support file service now.

# Parameters for provision

- storage\_account_name: [REQUIRED] - The name of the storage account. If the storage account does not exist, Broker will help you to create a new standard storage account with the name when `allowCreateStorageAccount` is set to `true`. The storage account names must be between 3 and 24 characters in length and use numbers and lower-case letters only.
- subscription_id: (optional) - The Azure Subscription id to use for storage accounts. If it is not set, `defaultSubscriptionID` will be used. It will fails if neither is set.
- resource\_group_name: (optional) - The resource group name to use for storage accounts. If it is not set, `defaultResourceGroupName` will be used. It will fails if neither is set.
- location: Available when creating a new storage account. The location to use for creating storage accounts. If it is not set, `defaultLocation` will be used. It will fails to create a new storage account if neither is set.
- use_https: Available when creating a new storage account. Allows https traffic only to storage service if sets to `true`. It *MUST* be set to `false` if you want to use smbdriver inside Linux VMs. Otherwise, the mount in Linux will fail. Please see more details [here](https://docs.microsoft.com/en-us/azure/storage/storage-security-guide). Default value is `false`.
- sku_name: Available when creating a new storage account. The sku name for the storage account. Only standard storage account supports Azure file service. Allowed values: `Standard_GRS`, `Standard_LRS` or `Standard_RAGRS`. Default value is `Standard_RAGRS`.
- enable_encryption: Available when creating a new storage account. Indicating whether or not the service encrypts the data as it is stored. Only blob service and file service support encryption. Default value is `true`.

# Parameters for bind

- share: [REQUIRED] - The file share name in the storage account. If the file share does not exist, Broker will help you to create a new file share with the name when `allowCreateFileShare` is set to `true`. Please see share name restrictions [here](https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-shares--directories--files--and-metadata#share-names).
- uid: Sets the uid that will own all files or directories on the mounted filesystem.
- gid: Sets the gid that will own all files or directories on the mounted filesystem.
- file_mode: Sets the default file mode. For example, `0777`.
- dir_mode: Sets the default mode for directories. For example, `0666`.
- readonly: Mounts the share as read-only. Default value is `false`.
- vers: The SMB version used to mount Azure file shares. Allowed values: `3.0` and `2.1`. Please see more information [here](https://azure.microsoft.com/en-us/blog/azure-file-storage-now-generally-available/).
- mount: (optional) - The local directory mount-point. If it is not set, Broker will use `/var/vcap/data/#{ServiceInstanceID}` as the mount-point.
