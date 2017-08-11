# SMB volume release

This is a bosh release that packages an [smbdriver](https://github.com/AbelHu/smbdriver) and [azurefilebroker](https://github.com/AbelHu/azurefilebroker) for consumption by a volume_services_enabled Cloud Foundry deployment.

This broker/driver pair allows you to provision existing Azure storage accounts and bind file shares to your applications for shared file access.

# Deploying to Azure

## Pre-requisites

1. Install Cloud Foundry with Diego, or start from an existing CF+Diego deployment on Azure.

1. If you are starting from scratch, you can follow this [guidance](https://github.com/cloudfoundry-incubator/bosh-azure-cpi-release/tree/master/docs) to deploy a Cloud Foundry with Diego on Azure via Azure template.

## Create and Upload this Release

1. Clone smb-volume-release (master branch) from git:

    ```bash
    $ cd ~/workspace
    $ git clone https://github.com/AbelHu/smb-volume-release.git
    $ cd ~/workspace/smb-volume-release
    $ direnv allow .
    $ git checkout master
    $ ./scripts/update
    ```

    **NOTE**:
    *You may need to install direnv by following this [guidance](https://github.com/direnv/direnv).*

1. Bosh Create and Upload the release

    ```bash
    $ bosh -n create release --force && bosh -n upload release
    ```

## Enable Volume Services in CF and Redeploy

In your CF manifest, check the setting for `properties: cc: volume_services_enabled`.  If it is not already `true`, set it to `true` and redeploy CF.  (This will be quick, as it only requires BOSH to restart the cloud controller job with the new property.)

## Colocate the smbdriver job on the Diego Cell

1. Add `smb-volume` to the `releases:` key

    ```yaml
    releases:
    - name: diego
      version: latest
      ...
    - name: smb-volume
      version: latest
    ```

1. Add `smbdriver` to the `jobs: name: cell_z1 templates:` key

    ```yaml
    jobs:
      ...
      - name: cell_z1
        ...
        templates:
        - name: consul_agent
          release: cf
          ...
        - name: smbdriver
          release: smb-volume
    ```

1. Redeploy Diego using your new manifest.

## Deploying azurefilebroker

The azurefilebroker can be deployed in two ways; as a cf app or as a BOSH deployment.  The choice is yours!

### Way `cf push` the broker

When the service broker is `cf push`ed, you can bind it to a MSSql or MySql database service instance.

**NOTE**
*For now, it is not supported to bind to Azure SQL with meta-azure-service-broker because the variable names do not match. A fix is in progress. You must specify variables in the manifest.*

If you are using Azure SQL service, you need to change the policy from "default" to "proxy" after creating your SQL server.

1. Install the Chocolatey by running the command below using CMD which is opening in administrator privilege.

    ```dos
    @"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -Command "iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))" && SET "PATH=%PATH%;%ALLUSERSPROFILE%\chocolatey\bin"
    ```

1. Run below command to install the armclient.

    ```dos
    choco install armclient
    ```

1. Run below command step by step to change the policy

- Login

    ```dos
    armclient login
    ```

- Check the current setting for the server. You need to fill the parameters in the URL with yours accordingly.

    ```dos
    armclient get https://management.azure.com/subscriptions/<YOUR-SUBSCRIPTION-ID>/resourceGroups/<YOUR-RESOURCE-GROUP-NAME>/providers/Microsoft.Sql/servers/<YOUR-SQL-SERVER-NAME>/connectionPolicies/Default?api-version=2014-04-01
    ```

- Save below content to a file and navigate to the folder containing this file in cmd window. Change the setting by command below. You need to fill the parameters in the URL with yours accordingly.

    ```
    {
            "properties": {
                    "connectionType": "Proxy"
            }
    }
    ```

    ```dos
    armclient put https://management.azure.com/subscriptions/<YOUR-SUBSCRIPTION-ID>/resourceGroups/<YOUR-RESOURCE-GROUP-NAME>/providers/Microsoft.Sql/servers/<YOUR-SQL-SERVER-NAME>/connectionPolicies/Default?api-version=2014-04-01 @policy.json
    ```

Once you have a database service instance available in the space where you will push your service broker application, follow the following steps:

- `cd src/github.com/AbelHu/azurefilebroker`
- `GOOS=linux GOARCH=amd64 go build -o bin/azurefilebroker`
- edit `manifest.yml` to set up broker username/password and sql db driver name and cf service name.  If you are using the [cf-mysql-release](http://bosh.io/releases/github.com/cloudfoundry/cf-mysql-release) from bosh.io, then the database parameters in manifest.yml will already be correct.
- `cf push <broker app name> --no-start`
- `cf bind-service <broker app name> <sql service instance name>`
- `cf start <broker app name>`

# Testing or Using this Release

## Register azurefilebroker
* Register the broker and grant access to it's service with the following command:

    ```bash
    $ cf create-service-broker azurefilebroker <BROKER_USERNAME> <BROKER_PASSWORD> http://azurefilebroker.YOUR.DOMAIN.com:9000
    $ cf enable-service-access azuresmbvolume
    ```

## Create an SMB volume service with an existing storage account
* type the following:

    ```bash
    $ cf create-service azuresmbvolume AzureFileShare myVolume -c '{"storage_account_name":"<YOUR-AZURE-STORAGE-ACCOUNT>"}'
    $ cf services
    ```

## Create an SMB volume service with a new storage account
* type the following:

    ```bash
    $ cf create-service azuresmbvolume AzureFileShare myVolume -c '{"storage_account_name":"<YOUR-AZURE-STORAGE-ACCOUNT>, "location":"<YOUR-LOCATION>"}'
    $ cf services
    ```

    **NOTE**:
    - You must not set `allowCreateStorageAccount` to `false` when deploying azurefilebroker.
    - The storage account created by Broker only can be deleted in deleteing the service when `allowDeleteStorageAccount` is set to `true`.
    - You must specify location when creating a new storage account if you do not specify default location when deploying azurefilebroker.
    - The Azure file share only can be binded to your application in Linux when they are in the same location.
    - Other supported parameters: `subscription_id`, `resource_group_name`, `use_https`, `sku_name`, `enable_encryption`
        - `subscription_id` must be specified if default subscription id is not specified when deploying azurefilebroker.
        - `resource_group_name` must be specified if default resource group is not specified when deploying azurefilebroker.
        - `use_https` must be set to `false` if your applicaiton is running in Linux. Reference more details [here](https://docs.microsoft.com/en-us/azure/storage/storage-security-guide). The default value is `false`.
        - `sku_name` must be one of `Standard_GRS`, `Standard_LRS` or `Standard_RAGRS`. The default value is `Standard_RAGRS`.
        - `enable_encryption` provides the encryption settings on the account. The default value is `true`.

## Deploy the pora test app, first by pushing the source code to CloudFoundry
* type the following:

    ```bash
    $ git clone https://github.com/cloudfoundry/persi-acceptance-tests.git
    $ cd persi-acceptance-tests/assets/pora
    $ cf push pora --no-start
    ```

* Bind the service to your app supplying the correct Azure file share. Broker will create it if it does not exist by default.

    ```bash
    $ cf bind-service pora myVolume -c '{"share": "one"}'
    ```

    **NOTE**:
        - You must not set `allowCreateFileShare` to `false` when deploying azurefilebroker.
        - The file share created by Broker only can be deleted in unbinding the service when `allowDeleteFileShare` is set to `true`.

> ####Bind Parameters####
> * **uid & gid:** When binding the Azure file share to the application, the uid and gid specified are supplied to the mount.cifs.  The fmount.cifs masks the running user id and group id as the true owner shown on the Azure file share.  Any operation on the mount will be executed as the owner, but locally the mount will be seen as being owned by the running user.
> * **mount:** By default, volumes are mounted into the application container in an arbitrarily named folder under /var/vcap/data.  If you prefer to mount your directory to some specific path where your application expects it, you can control the container mount path by specifying the `mount` option.  The resulting bind command would look something like
> ``` cf bind-service pora myVolume -c '{"share", "one", "uid":"0","gid":"0","mount":"/var/path"}'```
> > NOTE: As of this writing aufs used by Garden is not capable of creating new root level folders.  As a result, you must choose a path with a root level folder that already exists in the container.  (`/home`, `/usr` or `/var` are good choices.)  If you require a path that does not already exist in the container it is currently only possible if you upgrade your Diego deployment to use [GrootFS](https://github.com/cloudfoundry/grootfs-release) with Garden.  For details on how to generate a Diego manifest using GrootFS see [this note](https://github.com/cloudfoundry/diego-release/blob/develop/docs/manifest-generation.md#experimental--g-opt-into-using-grootfs-for-garden).  Eventually, GrootFS will become the standard file system for CF containers, and this limitation will go away.
> * **readonly:** Set true if you want the mounted volume to be read only.

* Start the application
    ```bash
    $ cf start pora
    ```

## Test the app to make sure that it can access your SMB volume

* to check if the app is running, `curl http://pora.YOUR.DOMAIN.com` should return the instance index for your app
* to check if the app can access the shared volume `curl http://pora.YOUR.DOMAIN.com/write` writes a file to the share and then reads it back out again.

# Reference

Please reference more information about Application specifies [here](https://github.com/cloudfoundry/nfs-volume-release).