## Development

# Run Unit Test

```bash
./scripts/run-driver-unit-tests
```

# Parameters of smbdriver
All parameters must start with `--`.

- listenPort: Port to serve volume management functions. Listen address is always `127.0.0.1`. Default value is `8589`.
- adminPort: Port to serve process admin functions. Default value is `8590`.
- debugAddr: (optional) - Address smbdriver will serve debug info. For example, `127.0.0.1:8689`.
- driversPath: [REQUIRED] - Path to directory where drivers are installed. For example, `/var/vcap/data/voldrivers`.
- transport: Transport protocol to transmit HTTP over. Default value is `tcp`.
- mountDir: Path to directory where fake volumes are created. Default value is `/tmp/volumes`.
- requireSSL: Whether the fake driver should require ssl-secured communication. Default value is `false`.
- mountFlagAllowed: [REQUIRED] - This is a comma separted list of parameters allowed to be send in extra config. Each of this parameters can be specify by brokers. For example, `username,password,uid,gid,file_mode,dir_mode,readonly,domain,vers,sec`.
- mountFlagDefault: (optional) - This is a comma separted list of like params:value. This list specify default value of parameters. If parameters has default value and is not in allowed list, this default value become a forced value who's cannot be override. For example, `uid:2000,gid:2000`.
- caFile: (optional) - The certificate authority public key file to use with ssl authentication.
- certFile: (optional) - The public key file to use with ssl authentication.
- keyFile: (optional) - The private key file to use with ssl authentication.
- clientCertFile: (optional) - The public key file to use with client ssl authentication.
- clientKeyFile: (optional) - The private key file to use with client ssl authentication.
- insecureSkipVerify: Whether SSL communication should skip verification of server IP addresses in the certificate. Default value is `false`.

  **NOTE:**
  About how to use the debug server, please see more details [here](https://github.com/cloudfoundry/debugserver).