---
title: Overview and Configuring
expires_at: never
tags: [smb-volume-release]
---

## Overview
On Cloud Foundry, applications connect to services via a service marketplace. Each service has a Service Broker, with encapsulates the logic for creating, managing, and binding services to applications. Until recently, the only data services that have been allowed were ones with a network-based connection, such as a SQL database. With Volume Services, we've added the ability to attach data services that have a filesystem-based interface.

Currently, we have platform support for Shared Volumes. Shared Volumes are distributed filesystems, such as NFS-based or SMB-Based systems, which allow all instances of an application to share the same mounted volume simultaneously and access it concurrently.

This feature adds two new concepts to CF: Volume Mounts on Service Brokers (SMBBroker) and Volume Drivers on Diego Cells (SMBDriver).

For more information on CF Volume Services, [please refer to this introductory document](https://docs.google.com/document/d/1YtPMY9EjxlgJPa4SVVwIinfid_fshCF48xRhzyoZhrQ/edit?usp=sharing).


## Parameters for smbdriver
All parameters must start with `--`.

- listenPort: Port to serve volume management functions. Listen address is always `127.0.0.1`. Default value is `8589`.
- adminPort: Port to serve process admin functions. Default value is `8590`.
- debugAddr: (optional) - Address smbdriver will serve debug info. For example, `127.0.0.1:8689`.
- driversPath: [REQUIRED] - Path to directory where drivers are installed. For example, `/var/vcap/data/voldrivers`.
- transport: Transport protocol to transmit HTTP over. Default value is `tcp`.
- mountDir: Path to directory where fake volumes are created. Default value is `/tmp/volumes`.
- requireSSL: Whether the fake driver should require ssl-secured communication. Default value is `false`.
- caFile: (optional) - The certificate authority public key file to use with ssl authentication.
- certFile: (optional) - The public key file to use with ssl authentication.
- keyFile: (optional) - The private key file to use with ssl authentication.
- clientCertFile: (optional) - The public key file to use with client ssl authentication.
- clientKeyFile: (optional) - The private key file to use with client ssl authentication.
- insecureSkipVerify: Whether SSL communication should skip verification of server IP addresses in the certificate. Default value is `false`.

> \[!NOTE\]
>
> About how to use the debug server, please see more details [here](https://github.com/cloudfoundry/debugserver).
