# SMB volume release

This is a bosh release that packages:
- an [smbdriver](https://github.com/cloudfoundry/smbdriver) 
- an [smbbroker](https://github.com/cloudfoundry/smbbroker) for preexisting SMB shares
- a test SMB server that provides a preexisting share to test against

The broker and driver pair allows you:
- to provision preexisting shares and bind the shares to your applications for share file access.

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
    $ bosh -e my-env -d cf deploy cf.yml -v deployment-vars.yml -o operations/experimental/enable-smb-volume-service.yml
    ```
    
**Note:** the above command is an example, but your deployment command should match the one you used to deploy Cloud Foundry initially, with the addition of a `-o operations/experimental/enable-smb-volume-service.yml` option.

Your CF deployment will now have a running service broker and volume drivers, ready to mount or create SMB volumes.  Unless you have explicitly defined a variable for your broker password, BOSH will generate one for you.

# Testing or Using this Release

## Deploying the Test SMB Server (Optional)

If you do not have an existing SMB Server then you can optionally deploy the test SMB server bundled in this release.

The easiest way to deploy the test server is to include the `enable-smb-test-server.yml` operations file when you deploy Cloud Foundry, also specifying `smb-username` and `smb-password` variables:
```bash
$ bosh -e my-env -d cf deploy cf.yml -v deployment-vars.yml \
  -v smb-username=smbuser \
  -v smb-password=something-secret \
  -o operations/experimental/enable-smb-volume-service.yml \
  -o operations/test/enable-smb-test-server.yml
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

## Follow the cf docs to deploy and test a sample app

Test instructions are [here](https://docs.cloudfoundry.org/devguide/services/using-vol-services.html#smb-sample)
The smbbroker uses credhub as a backing store, and as a result, does not require separate scripts for backup and restore, since credhub itself will get backed up by BBR.

# Troubleshooting
If you have trouble getting this release to operate properly, try consulting the [Volume Services Troubleshooting Page](https://github.com/cloudfoundry-incubator/volman/blob/master/TROUBLESHOOTING.md)
