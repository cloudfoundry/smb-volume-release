package smbdriver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"path/filepath"

	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/nfsdriver"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	"code.cloudfoundry.org/voldriver/invoker"
)

// smbMounter represent nfsdriver.Mounter for SMB
type smbMounter struct {
	invoker invoker.Invoker
	osutil  osshim.Os
	ioutil  ioutilshim.Ioutil
	config  Config
}

// NewSmbMounter create SMB mounter
func NewSmbMounter(invoker invoker.Invoker, osutil osshim.Os, ioutil ioutilshim.Ioutil, config *Config) nfsdriver.Mounter {
	return &smbMounter{invoker: invoker, osutil: osutil, ioutil: ioutil, config: *config}
}

// Reference: https://www.samba.org/samba/docs/man/manpages-3/mount.cifs.8.html
// Mount mount SMB folder to a local path
// Azure File Service:
//   required: username, password, vers=3.0
//   optional: uid, gid, file_mode, dir_mode, readonly | ro
// Windows Share Folders:
//   required: username, password | sec
//   optional: uid, gid, file_mode, dir_mode, readonly | ro, domain
func (m *smbMounter) Mount(env voldriver.Env, source string, target string, opts map[string]interface{}) error {
	logger := env.Logger().Session("smb-mount")
	logger.Info("start")
	defer logger.Info("end")

	// TODO--refactor the config object so that we don't have to make a local copy just to keep
	// TODO--it from leaking information between mounts.
	tempConfig := m.config.Copy()

	if err := tempConfig.SetEntries(opts, []string{"source"}); err != nil {
		logger.Debug("error-parse-entries", lager.Data{
			"given_source":  source,
			"given_target":  target,
			"given_options": opts,
			"config_mounts": tempConfig,
		})
		return err
	}

	mountOptions := []string{
		"-t", "cifs",
		source,
		target,
		"-o", strings.Join(tempConfig.MakeParams(), ","),
		"--verbose",
	}

	logger.Debug("parse-mount", lager.Data{
		"given_source":  source,
		"given_target":  target,
		"given_options": opts,
		"config_mounts": tempConfig,
		"mountOptions":  mountOptions,
	})

	logger.Debug("mount", lager.Data{"params": strings.Join(mountOptions, ",")})
	_, err := m.invoker.Invoke(env, "mount", mountOptions)
	return err
}

// Unmount unmount a SMB folder from a local path
func (m *smbMounter) Unmount(env voldriver.Env, target string) error {
	logger := env.Logger().Session("smb-umount")
	logger.Info("start")
	defer logger.Info("end")
	_, err := m.invoker.Invoke(env, "umount", []string{target})
	return err
}

// Check check whether a local path is mounted or not
func (m *smbMounter) Check(env voldriver.Env, name, mountPoint string) bool {
	logger := env.Logger().Session("smb-check-mountpoint")
	logger.Info("start")
	defer logger.Info("end")

	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(time.Second*5))
	defer cancel()
	env = driverhttp.EnvWithContext(ctx, env)
	_, err := m.invoker.Invoke(env, "mountpoint", []string{"-q", mountPoint})

	if err != nil {
		// Note: Created volumes (with no mounts) will be removed
		//       since VolumeInfo.Mountpoint will be an empty string
		logger.Info(fmt.Sprintf("unable to verify volume %s (%s)", name, err.Error()))
		return false
	}
	return true
}

// Purge delete all files in a local path
func (m *smbMounter) Purge(env voldriver.Env, path string) {
	logger := env.Logger().Session("purge")
	logger.Info("start")
	defer logger.Info("end")

	fileInfos, err := m.ioutil.ReadDir(path)
	if err != nil {
		logger.Error("purge-readdir-failed", err, lager.Data{"path": path})
		return
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			if err := m.osutil.RemoveAll(filepath.Join(path, fileInfo.Name())); err != nil {
				logger.Error("purge-cannot-remove-directory", err, lager.Data{"name": fileInfo.Name(), "path": path})
			}
		}
	}
}
