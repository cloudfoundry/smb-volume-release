---
title: Overview
expires_at: never
tags: [smb-volume-release]
---

# Overview

Volume Services is a collection of bosh releases, service brokers, and drivers for management, installation and access to various file system types which can be mounted to a Cloud Foundry Foundation. These components exist to allow applications running on Cloud Foundry to have access to a persistent filesystem.

Volume services where implemented by extending the Open Service Broker API, enabling broker authors to create data services which have a file system based interface. Before this CF supported Shared Volumes, which is a distributed filesystems, such as NFS-based systems, which allow all instances of an application to share the same mounted volume simultaneously and access it concurrently.

Volume service added two new concepts to Cloud Foundry: Volume Mounts for Service Brokers and Volume Drivers for Diego Cells.

For more information checkout code.cloudfoundry.org/volman

