//go:build e2e
// +build e2e

package test

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	kubeResourcePath = "./testdata/install.yaml"
)

func getConfigPath() string {
	usr, err := user.Current()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(usr.HomeDir, ".kube", "config")
}

var _ = Describe("ct-monitor", func() {
	kubectlOpts := k8s.NewKubectlOptions("kind-ct-monitor-kindtest", getConfigPath(), "default")

	BeforeEach(func() {
		k8s.KubectlApply(GinkgoT(), kubectlOpts, kubeResourcePath)
	})
	AfterEach(func() {
		k8s.KubectlDelete(GinkgoT(), kubectlOpts, kubeResourcePath)
	})

	setCronjobStatus := func(suspend bool) {
		status := strconv.FormatBool(suspend)
		By(fmt.Sprintf("setting cronjob suspend status to %s", status))
		k8s.RunKubectl(GinkgoT(), kubectlOpts, "patch", "cronjobs", "ct-monitor", "-p", fmt.Sprintf("{\"spec\": {\"suspend\": %s }}", status))
	}

	checkLogOutput := func(label string, contains, notContains []string) {
		out, err := k8s.RunKubectlAndGetOutputE(GinkgoT(), kubectlOpts, "logs", "-l", label)
		Expect(err).NotTo(HaveOccurred())
		for _, contain := range contains {
			Expect(out).To(ContainSubstring(contain))
		}
		for _ , notContain := range notContains {
			Expect(out).NotTo(ContainSubstring(notContain))
		}
	}

	It("should run cronjob", func() {
		By("listing pods")
		postfixListOpts := metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=postfix",
		}
		ctMonitorListOpts := metav1.ListOptions{
			FieldSelector: "status.phase==Succeeded",
			LabelSelector: "app.kubernetes.io/name=ct-monitor",
		}
		k8s.WaitUntilNumPodsCreated(GinkgoT(), kubectlOpts, postfixListOpts, 1, 5, 10*time.Second)
		k8s.WaitUntilNumPodsCreated(GinkgoT(), kubectlOpts, ctMonitorListOpts, 1, 10, 10*time.Second)

		setCronjobStatus(true)

		By("ensuring email report has been sent")
		checkLogOutput("app.kubernetes.io/name=ct-monitor", nil, []string{"error"})
		checkLogOutput("app.kubernetes.io/name=postfix", []string{"from=", "removed"}, nil)

		setCronjobStatus(false)

		By("checking for new pod")
		k8s.WaitUntilNumPodsCreated(GinkgoT(), kubectlOpts, ctMonitorListOpts, 2, 10, 10*time.Second)

		setCronjobStatus(true)

		By("making sure positions file has been used")
		checkLogOutput("app.kubernetes.io/name=ct-monitor", []string{"no new issuances observed"}, []string{"error"})
	})
})
