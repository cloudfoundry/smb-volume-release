package azurefilebroker_test

import (
	. "code.cloudfoundry.org/azurefilebroker/azurefilebroker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AzureConfig", func() {
	var (
		azureconfig *AzureConfig
	)

	Context("Given all required params", func() {
		BeforeEach(func() {
			azureconfig = NewAzureConfig("environment", "tenanID", "clientID", "clientSecret", "", "", "")
		})

		It("should not raise an error", func() {
			err := azureconfig.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Missing environment", func() {
		BeforeEach(func() {
			azureconfig = NewAzureConfig("", "tenanID", "clientID", "clientSecret", "", "", "")
		})

		It("should raise an error", func() {
			err := azureconfig.Validate()
			Expect(err).To(MatchError("Missing required parameters: environment"))
		})
	})

	Context("Missing tenanID", func() {
		BeforeEach(func() {
			azureconfig = NewAzureConfig("environment", "", "clientID", "clientSecret", "", "", "")
		})

		It("should raise an error", func() {
			err := azureconfig.Validate()
			Expect(err).To(MatchError("Missing required parameters: tenanID"))
		})
	})

	Context("Missing clientID", func() {
		BeforeEach(func() {
			azureconfig = NewAzureConfig("environment", "tenanID", "", "clientSecret", "", "", "")
		})

		It("should raise an error", func() {
			err := azureconfig.Validate()
			Expect(err).To(MatchError("Missing required parameters: clientID"))
		})
	})

	Context("Missing clientSecret", func() {
		BeforeEach(func() {
			azureconfig = NewAzureConfig("environment", "tenanID", "clientID", "", "", "", "")
		})

		It("should raise an error", func() {
			err := azureconfig.Validate()
			Expect(err).To(MatchError("Missing required parameters: clientSecret"))
		})
	})

	Context("Missing all required params", func() {
		BeforeEach(func() {
			azureconfig = NewAzureConfig("", "", "", "", "", "", "")
		})

		It("should raise an error", func() {
			err := azureconfig.Validate()
			Expect(err).To(MatchError("Missing required parameters: environment, tenanID, clientID, clientSecret"))
		})
	})
})

var _ = Describe("AzureStackConfig", func() {
	var (
		azureStackConfig *AzureStackConfig
	)

	Context("Given all required params", func() {
		BeforeEach(func() {
			azureStackConfig = NewAzureStackConfig("azureStackDomain", "azureStackAuthentication", "azureStackResource", "azureStackEndpointPrefix")
		})

		It("should not raise an error", func() {
			err := azureStackConfig.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Missing azureStackDomain", func() {
		BeforeEach(func() {
			azureStackConfig = NewAzureStackConfig("", "azureStackAuthentication", "azureStackResource", "azureStackEndpointPrefix")
		})

		It("should raise an error", func() {
			err := azureStackConfig.Validate()
			Expect(err).To(MatchError("Missing required parameters when 'environment' is 'AzureStack': azureStackDomain"))
		})
	})

	Context("Missing azureStackAuthentication", func() {
		BeforeEach(func() {
			azureStackConfig = NewAzureStackConfig("azureStackDomain", "", "azureStackResource", "azureStackEndpointPrefix")
		})

		It("should raise an error", func() {
			err := azureStackConfig.Validate()
			Expect(err).To(MatchError("Missing required parameters when 'environment' is 'AzureStack': azureStackAuthentication"))
		})
	})

	Context("Missing azureStackResource", func() {
		BeforeEach(func() {
			azureStackConfig = NewAzureStackConfig("azureStackDomain", "azureStackAuthentication", "", "azureStackEndpointPrefix")
		})

		It("should raise an error", func() {
			err := azureStackConfig.Validate()
			Expect(err).To(MatchError("Missing required parameters when 'environment' is 'AzureStack': azureStackResource"))
		})
	})

	Context("Missing azureStackEndpointPrefix", func() {
		BeforeEach(func() {
			azureStackConfig = NewAzureStackConfig("azureStackDomain", "azureStackAuthentication", "azureStackResource", "")
		})

		It("should raise an error", func() {
			err := azureStackConfig.Validate()
			Expect(err).To(MatchError("Missing required parameters when 'environment' is 'AzureStack': azureStackEndpointPrefix"))
		})
	})

	Context("Missing all required params", func() {
		BeforeEach(func() {
			azureStackConfig = NewAzureStackConfig("", "", "", "")
		})

		It("should raise an error", func() {
			err := azureStackConfig.Validate()
			Expect(err).To(MatchError("Missing required parameters when 'environment' is 'AzureStack': azureStackDomain, azureStackAuthentication, azureStackResource, azureStackEndpointPrefix"))
		})
	})
})

var _ = Describe("AzurefilebrokerCloudConfig", func() {
	var (
		cloudConfig *CloudConfig
		azure       *AzureConfig
		control     *ControlConfig
		azureStack  *AzureStackConfig
	)

	JustBeforeEach(func() {
		control = NewControlConfig(false, false, false, true)
		cloudConfig = NewAzurefilebrokerCloudConfig(azure, control, azureStack)
	})

	Context("Given all required params", func() {
		Context("When environment is not AzureStack", func() {
			BeforeEach(func() {
				azure = NewAzureConfig("Azure", "tenanID", "clientID", "clientSecret", "", "", "")
				azureStack = NewAzureStackConfig("", "", "", "")
			})

			It("should not raise an error", func() {
				err := cloudConfig.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When environment is AzureStack", func() {
			BeforeEach(func() {
				azure = NewAzureConfig("AzureStack", "tenanID", "clientID", "clientSecret", "", "", "")
				azureStack = NewAzureStackConfig("azureStackDomain", "azureStackAuthentication", "azureStackResource", "azureStackEndpointPrefix")
			})

			It("should not raise an error", func() {
				err := cloudConfig.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("Missing params for AzureStack", func() {
		BeforeEach(func() {
			azure = NewAzureConfig("AzureStack", "tenanID", "clientID", "clientSecret", "", "", "")
			azureStack = NewAzureStackConfig("", "", "", "")
		})

		It("should raise an error", func() {
			err := cloudConfig.Validate()
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("MountConfig", func() {
	var (
		config *MountConfig
	)

	BeforeEach(func() {
		config = NewAzurefilebrokerMountConfig()
	})

	Context("Copy", func() {
		BeforeEach(func() {
			config.Allowed = []string{"a", "b", "c"}
			config.Forced = map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
			}
			config.Options = map[string]string{
				"a1": "11",
				"b1": "22",
				"c1": "33",
				"d1": "44",
			}
		})

		It("Should return a full copy", func() {
			newConfig := config.Copy()
			Ω(newConfig).Should(Equal(config))
		})
	})

	Context("SetEntries", func() {
		var (
			opts map[string]string
		)

		BeforeEach(func() {
			config.Allowed = []string{"a", "b", "c"}
		})

		Context("Given allowd options", func() {
			BeforeEach(func() {
				opts = map[string]string{
					"a": "1",
					"b": "2",
					"c": "3",
				}
			})

			It("Should not raise an error", func() {
				err := config.SetEntries(opts)
				Expect(err).NotTo(HaveOccurred())
				Ω(config.Options).Should(Equal(opts))
			})
		})

		Context("Given not allowd options", func() {
			BeforeEach(func() {
				opts = map[string]string{
					"a":  "1",
					"b1": "2",
					"c":  "3",
				}
			})

			It("Should raise an error", func() {
				err := config.SetEntries(opts)
				Expect(err).To(MatchError("Not allowed options : b1"))
			})
		})
	})

	Context("MakeConfig", func() {
		var (
			options map[string]string
			result  map[string]interface{}
		)
		BeforeEach(func() {
			options = map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			}
			config.Allowed = []string{"a", "b", "c"}
			config.SetEntries(options)
		})

		Context("No forced options is given", func() {
			BeforeEach(func() {
				result = map[string]interface{}{
					"a": "1",
					"b": "2",
					"c": "3",
				}
			})
			It("Should return given options", func() {
				params := config.MakeConfig()
				Ω(params).Should(Equal(result))
			})
		})

		Context("Given forced options", func() {
			BeforeEach(func() {
				config.Forced = map[string]string{
					"b": "11",
					"d": "4",
				}
				result = map[string]interface{}{
					"a": "1",
					"b": "11",
					"c": "3",
					"d": "4",
				}
			})

			It("Should return given options", func() {
				params := config.MakeConfig()
				Ω(params).Should(Equal(result))
			})
		})
	})

	Context("ReadConf", func() {
		var (
			emptyMap    map[string]string
			emptyArray  []string
			allowedFlag string
			defaultFlag string
			err         error
		)

		BeforeEach(func() {
			emptyMap = make(map[string]string)
			emptyArray = []string{}
		})

		JustBeforeEach(func() {
			err = config.ReadConf(allowedFlag, defaultFlag)
		})

		Context("Given empty flags", func() {
			BeforeEach(func() {
				allowedFlag = ""
				defaultFlag = ""
			})

			It("Should not raise an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Ω(config.Allowed).Should(Equal(emptyArray))
				Ω(config.Options).Should(Equal(emptyMap))
				Ω(config.Forced).Should(Equal(emptyMap))
			})
		})

		Context("Only allowed flag is given", func() {
			var (
				allowedInConfig []string
			)

			BeforeEach(func() {
				allowedFlag = "a,b,c"
				defaultFlag = ""
				allowedInConfig = []string{"a", "b", "c"}
			})

			It("Should not raise an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Ω(config.Allowed).Should(Equal(allowedInConfig))
				Ω(config.Options).Should(Equal(emptyMap))
				Ω(config.Forced).Should(Equal(emptyMap))
			})
		})

		Context("Only default flag is given", func() {
			var (
				forcedInConfig map[string]string
			)

			BeforeEach(func() {
				allowedFlag = ""
				defaultFlag = "a:1,b:2,c:3"
				forcedInConfig = map[string]string{
					"a": "1",
					"b": "2",
					"c": "3",
				}
			})

			It("Should not raise an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Ω(config.Allowed).Should(Equal(emptyArray))
				Ω(config.Options).Should(Equal(emptyMap))
				Ω(config.Forced).Should(Equal(forcedInConfig))
			})
		})

		Context("Both flags are given", func() {
			var (
				allowedInConfig []string
				optionsInConfig map[string]string
				forcedInConfig  map[string]string
			)

			BeforeEach(func() {
				allowedFlag = "a,b"
				defaultFlag = "a:1,b:2,c:3"
				allowedInConfig = []string{"a", "b"}
				optionsInConfig = map[string]string{
					"a": "1",
					"b": "2",
				}
				forcedInConfig = map[string]string{
					"c": "3",
				}
			})

			It("Should not raise an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Ω(config.Allowed).Should(Equal(allowedInConfig))
				Ω(config.Options).Should(Equal(optionsInConfig))
				Ω(config.Forced).Should(Equal(forcedInConfig))
			})
		})
	})
})
