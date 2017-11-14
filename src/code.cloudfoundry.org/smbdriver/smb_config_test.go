package smbdriver_test

import (
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "code.cloudfoundry.org/smbdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func map2string(entry map[string]string, joinKeyVal string, joinElemnts string) string {
	return strings.Join(map2slice(entry, joinKeyVal), joinElemnts)
}

func mapstring2mapinterface(entry map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, 0)

	for k, v := range entry {
		result[k] = v
	}

	return result
}

func map2slice(entry map[string]string, joinKeyVal string) []string {
	result := make([]string, 0)

	for k, v := range entry {
		result = append(result, fmt.Sprintf("%s%s%s", k, joinKeyVal, v))
	}

	return result
}

func mapint2slice(entry map[string]interface{}, joinKeyVal string) []string {
	result := make([]string, 0)

	for k, v := range entry {
		switch v.(type) {
		case int:
			result = append(result, fmt.Sprintf("%s%s%s", k, joinKeyVal, strconv.FormatInt(int64(v.(int)), 10)))

		case string:
			result = append(result, fmt.Sprintf("%s%s%s", k, joinKeyVal, v.(string)))

		case bool:
			result = append(result, fmt.Sprintf("%s%s%s", k, joinKeyVal, strconv.FormatBool(v.(bool))))
		}

	}

	return result
}

func inSliceString(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}

	return false
}

func inMapInt(list map[string]interface{}, key string, val interface{}) bool {
	for k, v := range list {
		if k != key {
			continue
		}

		if v == val {
			return true
		} else {
			return false
		}
	}

	return false
}

var _ = Describe("ConfigDetails", func() {
	var (
		logger lager.Logger

		AbitraryConfig  map[string]interface{}
		IgnoreConfigKey []string

		MountsAllowed   []string
		MountsOptions   map[string]string
		MountsMandatory []string

		config *Config

		errorEntries error
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-config")
	})

	Context("Given no mandatory and empty params", func() {
		BeforeEach(func() {
			AbitraryConfig = make(map[string]interface{}, 0)
			IgnoreConfigKey = make([]string, 0)

			MountsAllowed = make([]string, 0)
			MountsOptions = make(map[string]string, 0)
			MountsMandatory = make([]string, 0)

			config = NewSmbConfig()
			config.ReadConf(strings.Join(MountsAllowed, ","), map2string(MountsOptions, ":", ","), MountsMandatory)

			logger.Debug("debug-config-initiated", lager.Data{"config": config})

			errorEntries = config.SetEntries(AbitraryConfig, IgnoreConfigKey)
			logger.Debug("debug-config-updated", lager.Data{"config": config})
		})

		It("should returns empty allowed list", func() {
			Expect(len(config.Allowed)).To(Equal(0))
		})

		It("should returns empty forced list", func() {
			Expect(len(config.Forced)).To(Equal(0))
		})

		It("should returns empty options list", func() {
			Expect(len(config.Options)).To(Equal(0))
		})

		It("should returns no missing mandatory fields", func() {
			Expect(len(config.CheckMandatory())).To(Equal(0))
		})

		It("should returns no error on given client arbitrary config", func() {
			Expect(errorEntries).To(BeNil())
		})

		It("should returns no mount command parameters", func() {
			Expect(len(config.MakeParams())).To(Equal(0))
		})

		It("should returns no MountOptions struct", func() {
			Expect(len(config.MakeConfig())).To(Equal(0))
		})
	})

	Context("Given mount mandatory and empty params", func() {
		BeforeEach(func() {
			AbitraryConfig = make(map[string]interface{}, 0)
			IgnoreConfigKey = make([]string, 0)

			MountsAllowed = make([]string, 0)
			MountsOptions = make(map[string]string, 0)
			MountsMandatory = []string{"username", "password"}

			config = NewSmbConfig()
			config.ReadConf(strings.Join(MountsAllowed, ","), map2string(MountsOptions, ":", ","), MountsMandatory)
			logger.Debug("debug-config-initiated", lager.Data{"config": config})

			errorEntries = config.SetEntries(AbitraryConfig, IgnoreConfigKey)
			logger.Debug("debug-config-updated", lager.Data{"config": config})
		})

		It("should returns empty allowed list", func() {
			Expect(len(config.Allowed)).To(Equal(0))
		})

		It("should returns empty forced list", func() {
			Expect(len(config.Forced)).To(Equal(0))
		})

		It("should returns empty options list", func() {
			Expect(len(config.Options)).To(Equal(0))
		})

		It("should flow the mandatory config as missing mandatory field", func() {
			Expect(len(config.CheckMandatory())).To(Equal(2))
			Expect(config.CheckMandatory()).To(Equal(MountsMandatory))
		})

		It("should occures an error because there are missing fields", func() {
			Expect(errorEntries).To(HaveOccurred())
		})

		It("should returns no mount command parameters", func() {
			Expect(len(config.MakeParams())).To(Equal(0))
		})

		It("should returns no MountOptions struct", func() {
			Expect(len(config.MakeConfig())).To(Equal(0))
		})
	})

	Context("Given mandatory, allowed and default params", func() {
		BeforeEach(func() {
			AbitraryConfig = make(map[string]interface{}, 0)
			IgnoreConfigKey = make([]string, 0)

			MountsAllowed = []string{"uid", "gid", "username", "password"}
			MountsMandatory = []string{"uid", "gid"}
			MountsOptions = map[string]string{
				"uid": "1003",
				"gid": "1001",
			}

			config = NewSmbConfig()
			config.ReadConf(strings.Join(MountsAllowed, ","), map2string(MountsOptions, ":", ","), MountsMandatory)
			logger.Debug("debug-config-initiated", lager.Data{"config": config})
		})

		It("should flow the allowed list", func() {
			Expect(config.Allowed).To(Equal(MountsAllowed))
		})

		It("should return empty forced list", func() {
			Expect(len(config.Forced)).To(Equal(0))
		})

		It("should flow the default params as options list", func() {
			Expect(config.Options).To(Equal(MountsOptions))
		})

		It("should return empty missing mandatory field", func() {
			Expect(len(config.CheckMandatory())).To(Equal(0))
		})

		Context("Given empty arbitrary params without any params", func() {
			BeforeEach(func() {
				errorEntries = config.SetEntries(AbitraryConfig, IgnoreConfigKey)
				logger.Debug("debug-config-updated", lager.Data{"config": config, "mount": config})
			})

			It("should return nil result on setting end users'entries", func() {
				Expect(errorEntries).To(BeNil())
			})

			It("flow the mount default options into the mount command parameters ", func() {
				actualRes := config.MakeParams()
				expectRes := map2slice(MountsOptions, "=")

				for _, exp := range expectRes {
					logger.Debug("checking-actual-res-contains-part", lager.Data{"actualRes": actualRes, "part": exp})
					Expect(inSliceString(actualRes, exp)).To(BeTrue())
				}

				for _, exp := range actualRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "part": exp})
					Expect(inSliceString(expectRes, exp)).To(BeTrue())
				}
			})

			It("should flow the mount default options into the MountOptions struct", func() {
				actualRes := config.MakeConfig()
				expectRes := mapstring2mapinterface(MountsOptions)

				for k, exp := range expectRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "key": k, "val": exp})
					Expect(inMapInt(actualRes, k, exp)).To(BeTrue())
				}

				for k, exp := range actualRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "key": k, "val": exp})
					Expect(inMapInt(expectRes, k, exp)).To(BeTrue())
				}
			})
		})

		Context("Given bad arbitrary params", func() {
			BeforeEach(func() {
				AbitraryConfig = map[string]interface{}{
					"missing": true,
					"wrong":   1234,
					"search":  "notfound",
				}
				IgnoreConfigKey = make([]string, 0)

				errorEntries = config.SetEntries(AbitraryConfig, IgnoreConfigKey)
				logger.Debug("debug-config-updated", lager.Data{"config": config})
			})

			It("should occured an error", func() {
				Expect(errorEntries).To(HaveOccurred())
				logger.Debug("debug-config-updated-with-entry", lager.Data{"config": config})
			})

			It("should flow the mount default options into the mount command parameters ", func() {
				actualRes := config.MakeParams()
				expectRes := map2slice(MountsOptions, "=")

				for _, exp := range expectRes {
					logger.Debug("checking-actual-res-contains-part", lager.Data{"actualRes": actualRes, "part": exp})
					Expect(inSliceString(actualRes, exp)).To(BeTrue())
				}

				for _, exp := range actualRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "part": exp})
					Expect(inSliceString(expectRes, exp)).To(BeTrue())
				}
			})

			It("should flow the mount default options into the MountOptions struct", func() {
				actualRes := config.MakeConfig()
				expectRes := mapstring2mapinterface(MountsOptions)

				for k, exp := range expectRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "key": k, "val": exp})
					Expect(inMapInt(actualRes, k, exp)).To(BeTrue())
				}

				for k, exp := range actualRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "key": k, "val": exp})
					Expect(inMapInt(expectRes, k, exp)).To(BeTrue())
				}
			})
		})

		Context("Given good arbitrary params", func() {
			BeforeEach(func() {
				AbitraryConfig = map[string]interface{}{
					"uid": "1234",
					"gid": "5678",
				}
				IgnoreConfigKey = make([]string, 0)

				errorEntries = config.SetEntries(AbitraryConfig, IgnoreConfigKey)
				logger.Debug("debug-config-updated", lager.Data{"config": config})
			})

			It("should not occured an error, return nil", func() {
				Expect(errorEntries).To(BeNil())
			})

			It("should flow the arbitrary config into the mount command parameters ", func() {
				actualRes := config.MakeParams()
				expectRes := mapint2slice(AbitraryConfig, "=")

				for _, exp := range expectRes {
					logger.Debug("checking-actual-res-contains-part", lager.Data{"actualRes": actualRes, "part": exp})
					Expect(inSliceString(actualRes, exp)).To(BeTrue())
				}

				for _, exp := range actualRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "part": exp})
					Expect(inSliceString(expectRes, exp)).To(BeTrue())
				}
			})

			It("should flow the arbitrary config into the MountOptions struct", func() {
				actualRes := config.MakeConfig()
				expectRes := AbitraryConfig

				for k, exp := range expectRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "key": k, "val": exp})
					Expect(inMapInt(actualRes, k, exp)).To(BeTrue())
				}

				for k, exp := range actualRes {
					logger.Debug("checking-expect-res-contains-part", lager.Data{"expectRes": expectRes, "key": k, "val": exp})
					Expect(inMapInt(expectRes, k, exp)).To(BeTrue())
				}
			})
		})
	})

	Context("Given mandatory and default params but with empty allowed", func() {
		BeforeEach(func() {
			AbitraryConfig = make(map[string]interface{}, 0)
			IgnoreConfigKey = make([]string, 0)

			MountsAllowed = make([]string, 0)
			MountsMandatory = []string{"uid", "gid"}
			MountsOptions = map[string]string{
				"uid": "1004",
				"gid": "1002",
			}
			config = NewSmbConfig()
			config.ReadConf(strings.Join(MountsAllowed, ","), map2string(MountsOptions, ":", ","), MountsMandatory)
			logger.Debug("debug-config-initiated", lager.Data{"config": config})

			errorEntries = config.SetEntries(AbitraryConfig, IgnoreConfigKey)
			logger.Debug("debug-config-updated", lager.Data{"config": config})
		})

		It("should return empty allowed list", func() {
			Expect(len(config.Allowed)).To(Equal(0))
		})

		It("should flow the default list as forced", func() {
			Expect(config.Forced).To(Equal(MountsOptions))
		})

		It("should return empty options list", func() {
			Expect(len(config.Options)).To(Equal(0))
		})

		It("should return empty missing mandatory field", func() {
			Expect(len(config.CheckMandatory())).To(Equal(0))
		})
	})
})
