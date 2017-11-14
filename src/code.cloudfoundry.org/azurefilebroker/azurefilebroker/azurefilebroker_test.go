package azurefilebroker_test

import (
	. "code.cloudfoundry.org/azurefilebroker/azurefilebroker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configuration", func() {
	var (
		config             Configuration
		subscriptionID     string
		resourceGroupName  string
		storageAccountName string
	)

	JustBeforeEach(func() {
		config = Configuration{
			SubscriptionID:     subscriptionID,
			ResourceGroupName:  resourceGroupName,
			StorageAccountName: storageAccountName,
		}
	})

	Context("Given all required params", func() {
		BeforeEach(func() {
			subscriptionID = "a"
			resourceGroupName = "b"
			storageAccountName = "c"
		})

		It("should not raise an error", func() {
			err := config.ValidateForAzureFileShare()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Missing subscription_id", func() {
		BeforeEach(func() {
			subscriptionID = ""
			resourceGroupName = "b"
			storageAccountName = "c"
		})

		It("should raise an error", func() {
			err := config.ValidateForAzureFileShare()
			Expect(err).To(MatchError("Missing required parameters: subscription_id"))
		})
	})

	Context("Missing resource_group_name", func() {
		BeforeEach(func() {
			subscriptionID = "a"
			resourceGroupName = ""
			storageAccountName = "c"
		})

		It("should raise an error", func() {
			err := config.ValidateForAzureFileShare()
			Expect(err).To(MatchError("Missing required parameters: resource_group_name"))
		})
	})

	Context("Missing storage_account_name", func() {
		BeforeEach(func() {
			subscriptionID = "a"
			resourceGroupName = "b"
			storageAccountName = ""
		})

		It("should raise an error", func() {
			err := config.ValidateForAzureFileShare()
			Expect(err).To(MatchError("Missing required parameters: storage_account_name"))
		})
	})

	Context("Missing all required params", func() {
		BeforeEach(func() {
			subscriptionID = ""
			resourceGroupName = ""
			storageAccountName = ""
		})

		It("should raise an error", func() {
			err := config.ValidateForAzureFileShare()
			Expect(err).To(MatchError("Missing required parameters: subscription_id, resource_group_name, storage_account_name"))
		})
	})
})

var _ = Describe("BindOptions", func() {
	var (
		options BindOptions
	)
	BeforeEach(func() {
		options = BindOptions{
			FileShareName: "a",
			UID:           "2000",
			GID:           "1000",
			FileMode:      "777",
			DirMode:       "666",
			Readonly:      true,
			Vers:          "b",
			Mount:         "c",
		}
	})

	Context("ToMap", func() {
		It("Should return expected map", func() {
			ret := options.ToMap()
			Ω(ret).Should(Equal(map[string]string{
				"uid":       "2000",
				"gid":       "1000",
				"file_mode": "777",
				"dir_mode":  "666",
				"readonly":  "true",
				"vers":      "b",
			}))
		})
	})
})
