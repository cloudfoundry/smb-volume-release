//go:build linux || darwin
// +build linux darwin

package smbdriver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/dockerdriver"
	"code.cloudfoundry.org/dockerdriver/driverhttp"
	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager/v3"
	vmo "code.cloudfoundry.org/volume-mount-options"
	"code.cloudfoundry.org/volumedriver"
	"code.cloudfoundry.org/volumedriver/invoker"
)

type smbMounter struct {
	invoker          invoker.Invoker
	osutil           osshim.Os
	ioutil           ioutilshim.Ioutil
	configMask       vmo.MountOptsMask
	forceNoserverino bool
	forceNoDfs       bool
}

func NewSmbMounter(invoker invoker.Invoker, osutil osshim.Os, ioutil ioutilshim.Ioutil, configMask vmo.MountOptsMask, forceNoserverino, forceNoDfs bool) volumedriver.Mounter {
	return &smbMounter{invoker: invoker, osutil: osutil, ioutil: ioutil, configMask: configMask, forceNoserverino: forceNoserverino, forceNoDfs: forceNoDfs}
}

func (m *smbMounter) Mount(env dockerdriver.Env, source string, target string, opts map[string]interface{}) error {
	logger := env.Logger().Session("smb-mount")
	logger.Info("start")
	defer logger.Info("end")

	mountOpts, err := vmo.NewMountOpts(opts, m.configMask)
	if err != nil {
		logger.Debug("error-parse-entries", lager.Data{
			"given_source":  source,
			"given_target":  target,
			"given_options": opts,
		})
		return safeError(err)
	}

	mountFlags, mountEnvVars := ToKernelMountOptionFlagsAndEnvVars(mountOpts)

	mountFlags = fmt.Sprintf("%s,uid=2000,gid=2000", mountFlags)

	if m.forceNoserverino {
		mountFlags = fmt.Sprintf("%s,noserverino", mountFlags)
	}

	if m.forceNoDfs {
		mountFlags = fmt.Sprintf("%s,nodfs", mountFlags)
	}

	mountArgs := []string{
		"-t", "cifs",
		source,
		target,
		"-o", mountFlags,
		"--verbose",
	}

	logger.Debug("parse-mount", lager.Data{
		"given_source":  source,
		"given_target":  target,
		"given_options": opts,
		"mountArgs":     mountArgs,
	})

	logger.Debug("mount", lager.Data{"params": strings.Join(mountArgs, ",")})
	invokeResult := m.invoker.Invoke(env, "mount", mountArgs, mountEnvVars...)
	return safeError(invokeResult.Wait())
}

func (m *smbMounter) Unmount(env dockerdriver.Env, target string) error {
	logger := env.Logger().Session("smb-umount")
	logger.Info("start")
	defer logger.Info("end")

	invokeResult := m.invoker.Invoke(env, "umount", []string{"-l", target})
	err := invokeResult.Wait()
	if err != nil {
		return safeError(err)
	}
	return nil
}

func (m *smbMounter) Check(env dockerdriver.Env, name, mountPoint string) bool {
	logger := env.Logger().Session("smb-check-mountpoint")
	logger.Info("start")
	defer logger.Info("end")

	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(time.Second*5))
	defer cancel()
	env = driverhttp.EnvWithContext(ctx, env)
	invokeResult := m.invoker.Invoke(env, "mountpoint", []string{"-q", mountPoint})
	err := invokeResult.Wait()
	if err != nil {
		// Note: Created volumes (with no mounts) will be removed
		//       since VolumeInfo.Mountpoint will be an empty string
		logger.Info(fmt.Sprintf("unable to verify volume %s (%s)", name, err.Error()))
		return false
	}
	return true
}

func (m *smbMounter) Purge(env dockerdriver.Env, path string) {
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
			mountDir := filepath.Join(path, fileInfo.Name())

			err = m.invoker.Invoke(env, "umount", []string{"-l", "-f", mountDir}).Wait()
			if err != nil {
				logger.Error("warning-umount-failed", err)
			} else {
				logger.Info("unmount-successful", lager.Data{"path": mountDir})
			}

			if err := m.osutil.Remove(mountDir); err != nil {
				logger.Error("purge-cannot-remove-directory", err, lager.Data{"name": mountDir, "path": path})
			}

			logger.Info("remove-directory-successful", lager.Data{"path": mountDir})
		}
	}
}

func NewSmbVolumeMountMask() (vmo.MountOptsMask, error) {
	allowed := []string{"mfsymlinks", "username", "password", "file_mode", "dir_mode", "ro", "domain", "vers", "sec", "version",
		"noserverino", "forceuid", "noforceuid", "forcegid", "noforcegid", "nodfs"}
	defaultMap := map[string]interface{}{}

	return vmo.NewMountOptsMask(
		allowed,
		defaultMap,
		map[string]string{"readonly": "ro", "version": "vers"},
		[]string{"source", "mount"},
		[]string{"username", "password"},
	)

}

func safeError(e error) error {
	if e == nil {
		return nil
	}
	return dockerdriver.SafeError{SafeDescription: e.Error()}
}
