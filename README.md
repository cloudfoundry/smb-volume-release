# SMB volume release

This is a bosh release that packages:
- an [smbdriver](https://github.com/cloudfoundry/smbdriver) 
- an [smbbroker](https://github.com/cloudfoundry/smbbroker) for preexisting SMB shares
- a test SMB server that provides a preexisting share to test against

The broker and driver pair allows you:
- to provision preexisting shares and bind the shares to your applications for share file access.

The test server provides an easy test target with which you can try out volume mounts of preexisting shares.

# Troubleshooting
If you have trouble getting this release to operate properly, try consulting the 
[Volume Services Troubleshooting Page](https://github.com/cloudfoundry-incubator/volman/blob/master/TROUBLESHOOTING.md)
