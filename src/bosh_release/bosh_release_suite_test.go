package bosh_release_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestBoshReleaseTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BoshReleaseTest Suite")
}

var repBuildPackagePath string
var release_path string
var stemcell_path string
var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(10 * time.Minute)

	release_path = os.Getenv("RELEASE_PATH")

	var err error
	repBuildPackagePath, err = gexec.BuildIn(release_path, "bosh_release/assets/rep")
	Expect(err).ShouldNot(HaveOccurred())

	if !hasStemcell() {
		uploadStemcell()
	}

	deploy()
})

func deploy(opsfiles ...string) {
	deployCmd := []string{"deploy",
		"-n",
		"-d",
		"bosh_release_test",
		"./smbdriver-manifest.yml",
		"-v", fmt.Sprintf("path_to_smb_volume_release=%s", release_path),
	}

	updatedDeployCmd := make([]string, len(deployCmd))
	copy(updatedDeployCmd, deployCmd)
	for _, optFile := range opsfiles {
		updatedDeployCmd = append(updatedDeployCmd, "-o", optFile)
	}

	boshDeployCmd := exec.Command("bosh", updatedDeployCmd...)
	session, err := gexec.Start(boshDeployCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 60*time.Minute).Should(gexec.Exit(0))

	disableHealthCheck := exec.Command("bosh", "update-resurrection", "off")
	session, err = gexec.Start(disableHealthCheck, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 60*time.Minute).Should(gexec.Exit(0))
}

func hasStemcell() bool {
	boshStemcellsCmd := exec.Command("bosh", "stemcells", "--json")
	stemcellOutput := gbytes.NewBuffer()
	session, err := gexec.Start(boshStemcellsCmd, stemcellOutput, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 1*time.Minute).Should(gexec.Exit(0))
	boshStemcellsOutput := &BoshStemcellsOutput{}
	Expect(json.Unmarshal(stemcellOutput.Contents(), boshStemcellsOutput)).Should(Succeed())
	return len(boshStemcellsOutput.Tables[0].Rows) > 0
}

func uploadStemcell() {

	stemcell_path = os.Getenv("STEMCELL_PATH")
	boshUsCmd := exec.Command("bosh", "upload-stemcell", stemcell_path)
	session, err := gexec.Start(boshUsCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 20*time.Minute).Should(gexec.Exit(0))
}

func findProcessState(processName string) string {

	boshIsCmd := exec.Command("bosh", "instances", "--ps", "--details", "--json", "--column=process", "--column=process_state")

	boshInstancesOutput := gbytes.NewBuffer()
	session, err := gexec.Start(boshIsCmd, boshInstancesOutput, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	session = session.Wait(1 * time.Minute)
	Expect(session.ExitCode()).To(Equal(0), string(boshInstancesOutput.Contents()))

	instancesOutputJson := &BoshInstancesOutput{}
	err = json.Unmarshal(boshInstancesOutput.Contents(), instancesOutputJson)
	Expect(err).NotTo(HaveOccurred())

	for _, row := range instancesOutputJson.Tables[0].Rows {
		if row.Process == processName {
			return row.ProcessState
		}
	}

	return ""
}

type BoshInstancesOutput struct {
	Tables []struct {
		Content string `json:"Content"`
		Header  struct {
			Process      string `json:"process"`
			ProcessState string `json:"process_state"`
		} `json:"Header"`
		Rows []struct {
			Process      string `json:"process"`
			ProcessState string `json:"process_state"`
		} `json:"Rows"`
		Notes interface{} `json:"Notes"`
	} `json:"Tables"`
	Blocks interface{} `json:"Blocks"`
	Lines  []string    `json:"Lines"`
}

type BoshStemcellsOutput struct {
	Tables []struct {
		Content string `json:"Content"`
		Header  struct {
			Cid     string `json:"cid"`
			Cpi     string `json:"cpi"`
			Name    string `json:"name"`
			Os      string `json:"os"`
			Version string `json:"version"`
		} `json:"Header"`
		Rows []struct {
			Cid     string `json:"cid"`
			Cpi     string `json:"cpi"`
			Name    string `json:"name"`
			Os      string `json:"os"`
			Version string `json:"version"`
		} `json:"Rows"`
		Notes []string `json:"Notes"`
	} `json:"Tables"`
	Blocks interface{} `json:"Blocks"`
	Lines  []string    `json:"Lines"`
}
