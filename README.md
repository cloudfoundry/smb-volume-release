# SMB volume release

This is a bosh release that packages an [smbdriver](https://github.com/cloudfoundry/smbdriver) and [azurefilebroker](https://github.com/cloudfoundry/azurefilebroker) for consumption by a volume_services_enabled Cloud Foundry deployment.

This broker/driver pair allows you:
1. to provision Azure storage accounts and bind Azure file shares to your applications for shared file access.
1. to provision preexisting shares and bind the shares to your applications for share file access.

# Deploying to Azure

## Pre-requisites

1. Install Cloud Foundry with Diego, or start from an existing CF+Diego deployment on Azure.

1. If you are starting from scratch, you can follow this [guidance](https://github.com/cloudfoundry-incubator/bosh-azure-cpi-release/tree/master/docs) to deploy a Cloud Foundry with Diego on Azure via Azure template.

1. Install [GO](https://golang.org/dl/):

    ```bash
    mkdir ~/workspace ~/go
    cd ~/workspace
    wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.9.linux-amd64.tar.gz
    echo 'export GOPATH=$HOME/go' >> ~/.bashrc
    echo 'export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin' >> ~/.bashrc
    exec $SHELL
    ```

1. Install [direnv](https://github.com/direnv/direnv#from-source):

    ```bash
    mkdir -p $GOPATH/src/github.com/direnv
    git clone https://github.com/direnv/direnv.git $GOPATH/src/github.com/direnv/direnv
    pushd $GOPATH/src/github.com/direnv/direnv
        make
        sudo make install
    popd
    echo 'eval "$(direnv hook bash)"' >> ~/.bashrc
    exec $SHELL
    ```

## Create and Upload this Release

1. Clone smb-volume-release (master branch) from git:

    ```bash
    $ cd ~/workspace
    $ git clone https://github.com/cloudfoundry/smb-volume-release.git
    $ cd ~/workspace/smb-volume-release
    $ direnv allow .
    $ git checkout master
    $ ./scripts/update
    ```

1. Bosh create the release

    ```bash
    $ bosh -n create release --force
    ```

1. Bosh upload the release

    ```bash
    # BOSH CLI v1
    $ bosh -n upload release
    ```

    ```bash
    # BOSH CLI v2
    $ bosh -n -e <YOUR BOSH DEPLOYMENT NAME> upload-release
    ```

## Enable Volume Services in CF and Redeploy

In your CF manifest, check the setting for `properties: cc: volume_services_enabled` (If multiple cc properties exist, please update cc properties under the job `api`).  If it is not already `true`, set it to `true` and redeploy CF.  (This will be quick, as it only requires BOSH to restart the cloud controller job with the new property.)

## Colocate the smbdriver job on the Diego Cell

If you have a bosh director version < `259` you will need to use one of the OLD WAYS below. (check `bosh status` to determine your version).  Otherwise we recommend the NEW WAY :thumbsup::thumbsup::thumbsup:

### OLD WAY Manual Editing

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

### NEW WAY Use bosh add-ons with filtering

This technique allows you to co-locate bosh jobs on cells without editing the Diego bosh manifest.

1. Create a new `runtime-config.yml` with the following content:

    ```yaml
    ---
    releases:
    - name: smb-volume
      version: <YOUR VERSION HERE>
    addons:
    - name: voldrivers
      include:
        deployments:
        - <YOUR DIEGO DEPLOYMENT NAME>
        jobs:
        - name: rep
          release: diego
      jobs:
      - name: smbdriver
        release: smb-volume
        properties: {}
    ```

1. Set the runtime config:

    ```bash
    # BOSH CLI v1
    $ bosh update runtime-config runtime-config.yml
    ```

    ```bash
    # BOSH CLI v2
    $ bosh -e <YOUR BOSH DEPLOYMENT NAME> update-runtime-config runtime-config.yml
    ```

1. Redeploy Diego using your new manifest.

## Deploying azurefilebroker

The azurefilebroker can be deployed in two ways; as a cf app or as a BOSH deployment.  The choice is yours!

### Way #1 `cf push` the broker

When the service broker is `cf push`ed, you can bind it to a MSSql or MySql database service instance.

**NOTE**
*It is not supported to bind to Azure SQL with an old meta-azure-service-broker before `v1.5.1` because the variable names do not match. You must specify variables in the manifest.*

If you want to use an Azure SQL service to store data for the service broker,
  - If you are using `meta-azure-service-broker` to [provision a new databse with a new server](https://github.com/Azure/meta-azure-service-broker/blob/master/docs/azure-sql-db.md#create-a-datbase-on-a-new-server), you can set `connectionPolicy` to `proxy` in configuration parameters.
  - Otherwise you need to change the policy from "default" to "proxy" after creating your SQL server by following below steps:

    1. Open PowerShell in administrator privilege and install AzureRM module.

        ```powershell
        Install-Module AzureRM
        Import-Module AzureRM
        ```

    1. Download [reconfig-mssql-policy.ps1](./scripts/reconfig-mssql-policy.ps1) and run it to change the policy of your Azure SQL server.

        ```powershell
        Set-ExecutionPolicy Unrestricted -Scope CurrentUser
        ./reconfig-mssql-policy.ps1
        ```

Once you have a database service instance available in the space where you will push your service broker application, follow the following steps:

- `./script/build-broker`
- Edit `manifest.yml` to set up all required parameters. `manifest.yml` and `Procfile` will work together to start the broker.
    - With a database directly: `DBCACERT`, `HOSTNAMEINCERTIFICATE`, `DBUSERNAME`, `DBPASSWORD`, `DBHOST`, `DBPORT` and `DBNAME` in `manifest.yml` need to be set up.

    ```bash
    cf push azurefilebroker
    ```

    - With a Cloud Foundry database service instance: `DBCACERT`, `HOSTNAMEINCERTIFICATE` and `DBSERVICENAME` needs to be set up.

    ```bash
    cf push azurefilebroker --no-start
    cf bind-service azurefilebroker <sql service instance name>
    cf start azurefilebroker
    ```

    **NOTE**:

    - When `environment` is `Preexisting`, only one plan `Existing` is supported; For others, both two plans `Existing` and `AzureFileShare` are supported.
    - Please see more details about broker's configurations [here](./docs/broker-development.md#configurations-of-azurefilebroker).

### Way #2 - `bosh deploy` the broker

You can reference [bosh deploy nfsbroker](https://github.com/cloudfoundry/nfs-volume-release/blob/master/README.md#way-2---bosh-deploy-the-broker).

# Testing or Using this Release

## Deploying the Test SMB Server (Optional)

If you do not have an existing SMB Server then you can optionally deploy the test SMB server bundled in this release.

**NOTE**: *This test SMB server only works with Ubuntu stemcells.*

### Generate the Deployment Manifest

#### Create Stub Files

##### director.yml
* determine your bosh director uuid by invoking

    ```bash
    # BOSH CLI v1
    $ bosh status --uuid
    ```

    ```bash
    # BOSH CLI v2
    $ bosh -e <YOUR BOSH DEPLOYMENT NAME> env
    ```

* create a new director.yml file and place the following contents into it:

    ```yaml
    ---
    director_uuid: <your uuid>
    ```

#### iaas.yml

* Create a stub for your iaas settings from the following template:

    ```yaml
    ---
    networks:
    - name: smbvolume-subnet
      subnets:
      - cloud_properties:
        virtual_network_name: <--- SUBNET YOU WANT YOUR AZUREFILEBROKER TO BE IN --->
        subnet_name: <--- SUBNET YOU WANT YOUR AZUREFILEBROKER TO BE IN --->
        security_group: <--- SECURITY GROUP YOU WANT YOUR AZUREFILEBROKER TO BE IN --->
        dns:
        - 10.10.0.2
        gateway: 10.10.200.1
        range: 10.10.200.0/24
        reserved:
        - 10.10.200.2 - 10.10.200.9
        # ceph range 10.10.200.106-110
        # local range 10.10.200.111-115
        # efs range 10.10.200.116-120
        # smb range 10.10.200.121-125
        - 10.10.200.106 - 10.10.200.125
        static:
        - 10.10.200.10 - 10.10.200.105

    resource_pools:
    - name: medium
      stemcell:
        name: bosh-azure-hyperv-ubuntu-trusty-go_agent
        version: latest
      cloud_properties:
        instance_type: Standard_A2
    - name: large
      stemcell:
        name: bosh-azure-hyperv-ubuntu-trusty-go_agent
        version: latest
      cloud_properties:
        instance_type: Standard_A3

    smb-test-server:
      ips: [<--- PRIVATE IP ADDRESS --->]
      username: <--- Username for SMB shares --->
      password: <--- Password for SMB shares --->
    ```

> NB: manually edit to fix hard-coded ip ranges, security group and subnets to match your deployment.

* run the following script:

    ```bash
    $ ./scripts/generate_server_manifest.sh director-uuid.yml iaas.yml
    ```

to generate `smb-test-server-azure-manifest.yml` into the current directory.

> NB: by default, the smb test server expects that your CF deployment is deployed to a 10.x.x.x subnet.  If you are deploying to a subnet that is not 10.x.x.x (e.g. 192.168.x.x) then you will need to override the `export_cidr` property.
> Edit the generated manifest, and add something like this:

  ```
    smbtestserver:
      username: xxx
      password: xxx
      export_cidr: 192.168.0.0/16
  ```

### Deploy the SMB Server
* Deploy the SMB server using the generated manifest:

    ```bash
    # BOSH CLI v1
    $ bosh -d smb-test-server-azure-manifest.yml deploy
    ```

    ```bash
    # BOSH CLI v2
    $ bosh -e <YOUR BOSH DEPLOYMENT NAME> -d smb-server deploy smb-test-server-azure-manifest.yml
    ```

## Register azurefilebroker

* Register the broker and grant access to it's service with the following command:

    ```bash
    $ cf create-service-broker azurefilebroker <BROKER_USERNAME> <BROKER_PASSWORD> http://azurefilebroker.YOUR.DOMAIN.com
    $ cf enable-service-access smbvolume
    ```

## Create an SMB volume service with an existing storage account on Azure

1. type the following:

    ```bash
    $ cf create-service smbvolume AzureFileShare myVolume -c '{"storage_account_name":"<YOUR-AZURE-STORAGE-ACCOUNT>"}'
    $ cf services
    ```

## Create an SMB volume service with a new storage account on Azure

1. type the following:

    ```bash
    $ cf create-service smbvolume AzureFileShare myVolume -c '{"storage_account_name":"<YOUR-AZURE-STORAGE-ACCOUNT>, "location":"<YOUR-LOCATION>"}'
    $ cf services
    ```

    **NOTE**:

    - Please see more details about parameters [here](./docs/broker-development.md#parameters-for-provision).
    - The Azure file share only can be binded to your application in Linux when they are in the same location.

## Create an SMB volume service with a preexisting share

1. type the following:

    ```bash
    $ cf create-service smbvolume Existing myVolume -c '{"share":"<PRIVATE_IP>/export/vol1"}'
    $ cf services
    ```

    **NOTE**:
    *The format of the share is `//server/folder` or `\\\\server\\folder`.*

## Deploy the pora test app, first by pushing the source code to CloudFoundry

1. type the following:

    ```bash
    $ git clone https://github.com/cloudfoundry/persi-acceptance-tests.git
    $ cd persi-acceptance-tests/assets/pora
    $ cf push pora --no-start
    ```

1. Bind the service to your app supplying the correct Azure file share. Broker will create it if it does not exist by default.

    ```bash
    $ cf bind-service pora myVolume -c '{"share": "one"}'
    ```

1. Bind the service to your app supplying the correct credentials for preexisting shares.

    ```bash
    $ cf bind-service pora myVolume -c '{"username": "a", "password": "b"}'
    ```

    **NOTE**:

    - Please see more details about parameters [here](./docs/broker-development.md#parameters-for-bind).
    - mount: By default, volumes are mounted into the application container in an arbitrarily named folder under `/var/vcap/data`.  If you prefer to mount your directory to some specific path where your application expects it, you can control the container mount path by specifying the `mount` option.  The resulting bind command would look something like
        ``` cf bind-service pora myVolume -c '{"share", "one", "mount":"/var/path"}' ```
    NOTE: As of this writing aufs used by Garden is not capable of creating new root level folders.  As a result, you must choose a path with a root level folder that already exists in the container.  (`/home`, `/usr` or `/var` are good choices.)  If you require a path that does not already exist in the container it is currently only possible if you upgrade your Diego deployment to use [GrootFS](https://github.com/cloudfoundry/grootfs-release) with Garden.  For details on how to generate a Diego manifest using GrootFS see [this note](https://github.com/cloudfoundry/diego-release/blob/develop/docs/manifest-generation.md#experimental--g-opt-into-using-grootfs-for-garden). Eventually, GrootFS will become the standard file system for CF containers, and this limitation will go away.
    - If you are using an Azure file share as the preexisting share, you need to specify `"vers": "3.0"` in the parameters.

1. Start the application

    ```bash
    $ cf start pora
    ```

## Test the app to make sure that it can access your SMB volume

1. to check if the app is running, `curl http://pora.YOUR.DOMAIN.com` should return the instance index for your app
1. to check if the app can access the shared volume `curl http://pora.YOUR.DOMAIN.com/write` writes a file to the share and then reads it back out again.

# Reference

Please reference more information about Application specifies [here](https://github.com/cloudfoundry/nfs-volume-release).
