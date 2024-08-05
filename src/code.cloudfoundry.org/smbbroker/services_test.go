package main_test

import (
	. "code.cloudfoundry.org/smbbroker"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v11/domain"
)

var _ = Describe("Services", func() {
	var (
		services Services
	)

	BeforeEach(func() {
		var err error
		services, err = NewServicesFromConfig("./default_services.json")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("List", func() {
		It("returns the list of services", func() {
			Expect(services.List()).To(Equal([]domain.Service{
				{
					ID:            "9db9cca4-8fd5-4b96-a4c7-0a48f47c3bad",
					Name:          "smb",
					Description:   "Existing SMB shares (see: https://code.cloudfoundry.org/smb-volume-release/)",
					Bindable:      true,
					PlanUpdatable: false,
					Tags:          []string{"smb"},
					Requires:      []domain.RequiredPermission{"volume_mount"},

					Plans: []domain.ServicePlan{
						{
							ID:          "0da18102-48dc-46d0-98b3-7a4ff6dc9c54",
							Name:        "Existing",
							Description: "A preexisting share",
						},
					},
				},
			}))
		})
	})
})
