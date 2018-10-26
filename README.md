# SMB volume release

This is a bosh release that packages:
- an [smbdriver](https://github.com/cloudfoundry/smbdriver) 
- an [smbbroker](https://github.com/cloudfoundry/smbbroker) for preexisting SMB shares
- an [azurefilebroker](https://github.com/cloudfoundry/azurefilebroker) for Azure storage accounts
- BOSH backup & restore jobs for the azurefilebroker's database
- a test SMB server that provides a preexisting share to test against

The broker and driver pair allows you:
- to provision preexisting shares and bind the shares to your applications for share file access.
- to provision Azure storage accounts and bind Azure file shares to your applications for shared file access.

The test server provides an easy test target with which you can try out volume mounts of preexisting shares.

# Deploying to Cloud Foundry

## Pre-requisites

1. Install Cloud Foundry, or start from an existing CF deployment.  If you are starting from scratch, the article [Overview of Deploying Cloud Foundry](https://docs.cloudfoundry.org/deploying/index.html) provides detailed instructions.

## Redeploy Cloud Foundry with smb enabled

1. You should have it already after deploying Cloud Foundry, but if not clone the cf-deployment repository from git:

    ```bash
    $ cd ~/workspace
    $ git clone https://github.com/cloudfoundry/cf-deployment.git
    $ cd ~/workspace/cf-deployment
    ```

2. Now redeploy your cf-deployment while including the smb ops file:
    ```bash
    $ bosh -e my-env -d cf deploy cf.yml -v deployment-vars.yml -o /operations/experimental/enable-smb-volume-service.yml
    ```
    
**Note:** the above command is an example, but your deployment command should match the one you used to deploy Cloud Foundry initially, with the addition of a `-o /operations/experimental/enable-smb-volume-service.yml` option.

Your CF deployment will now have a running service broker and volume drivers, ready to mount or create SMB volumes.  Unless you have explicitly defined a variable for your broker password, BOSH will generate one for you.

# Testing or Using this Release

## Deploying the Test SMB Server (Optional)

If you do not have an existing SMB Server then you can optionally deploy the test SMB server bundled in this release.

The easiest way to deploy the test server is to include the `enable-smb-test-server.yml` operations file when you deploy Cloud Foundry, also specifying `smb-username` and `smb-password` variables:
```bash
$ bosh -e my-env -d cf deploy cf.yml -v deployment-vars.yml \
  -v smb-username=smbuser \
  -v smb-password=something-secret \
  -o /operations/experimental/enable-smb-volume-service.yml \
  -o ../smb-volume-release/operations/enable-smb-test-server.yml
```

**NOTE**: *This test SMB server only works with Ubuntu stemcells.*

## Register smbbroker

* Deploy and register the broker and grant access to its service with the following command:

    ```bash
    $ bosh -e my-env -d cf run-errand smb-broker-registrar
    $ cf enable-service-access smb
    ```

## Testing and General Usage with smbbroker

You can refer to the [Cloud Foundry docs](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html#smb) for testing and general usage information.

## Testing with azurefilebroker

We don't currently provide ops files in this release that include the azurefilebrokerpush errand, but you can make one fairly easily, or refer to an older version of this release.  Once you have the azure file broker installed, you can use the instructions below to create an azure file service.

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
    - The Azure file share only can be bound to your application in Linux when they are in the same location.

## Follow the cf docs to deploy and test a sample app

Test instructions are [here](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html#smb-sample)
# BBR Support for azurefilebroker
The smbbroker uses credhub as a backing store, and as a result, does not require separate scripts for backup and restore, since credhub itself will get backed up by BBR.
For azurefilebroker, if you are using [Bosh Backup and Restore](https://docs.cloudfoundry.org/bbr/) (BBR) to keep backups of your Cloud Foundry deployment, consider including the [enable-azurefile-broker-backup.yml](https://github.com/cloudfoundry/smb-volume-release/blob/master/operations/enable-azurefile-broker-backup.yml) operations file from this repository when you redeploy Cloud Foundry.  This file will install the requiste backup and restore scripts for service broker metadata on the backup/restore VM.

# Troubleshooting
If you have trouble getting this release to operate properly, try consulting the [Volume Services Troubleshooting Page](https://github.com/cloudfoundry-incubator/volman/blob/master/TROUBLESHOOTING.md)
