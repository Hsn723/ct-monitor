//go:build e2e
// +build e2e

package test

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
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

	suspendCronjob := func() {
		status := strconv.FormatBool(true)
		By(fmt.Sprintf("setting cronjob suspend status to %s", status))
		k8s.RunKubectl(GinkgoT(), kubectlOpts, "patch", "cronjobs", "ct-monitor", "-p", fmt.Sprintf("{\"spec\": {\"suspend\": %s }}", status))
	}

	triggerJob := func() {
		By("triggering job from cronjob")
		jobName := fmt.Sprintf("ct-monitor-%s", uuid.NewString())
		k8s.RunKubectl(GinkgoT(), kubectlOpts, "create", "job", "--from=cronjob/ct-monitor", jobName)
	}

	checkLogOutput := func(label string, contains, notContains []string) {
		out, err := k8s.RunKubectlAndGetOutputE(GinkgoT(), kubectlOpts, "logs", "-l", label)
		Expect(err).NotTo(HaveOccurred())
		for _, contain := range contains {
			Expect(out).To(ContainSubstring(contain))
		}
		for _, notContain := range notContains {
			Expect(out).NotTo(ContainSubstring(notContain))
		}
	}

	BeforeEach(func() {
		k8s.KubectlApply(GinkgoT(), kubectlOpts, kubeResourcePath)
		suspendCronjob()
		postfixListOpts := metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=postfix",
		}
		k8s.WaitUntilNumPodsCreated(GinkgoT(), kubectlOpts, postfixListOpts, 1, 5, 10*time.Second)
		postfixPods := k8s.ListPods(GinkgoT(), kubectlOpts, postfixListOpts)
		for _, pod := range postfixPods {
			k8s.WaitUntilPodAvailable(GinkgoT(), kubectlOpts, pod.Name, 10, 10*time.Second)
		}
	})

	AfterEach(func() {
		k8s.KubectlDelete(GinkgoT(), kubectlOpts, kubeResourcePath)
	})

	It("should run cronjob", func() {
		triggerJob()

		By("listing pods")
		ctMonitorListOpts := metav1.ListOptions{
			FieldSelector: "status.phase==Succeeded",
			LabelSelector: "app.kubernetes.io/name=ct-monitor",
		}

		k8s.WaitUntilNumPodsCreated(GinkgoT(), kubectlOpts, ctMonitorListOpts, 1, 10, 10*time.Second)

		By("ensuring email report has been sent")
		checkLogOutput("app.kubernetes.io/name=ct-monitor", nil, []string{"error"})
		checkLogOutput("app.kubernetes.io/name=postfix", []string{"from=", "removed"}, nil)

		triggerJob()

		By("checking for new pod")
		k8s.WaitUntilNumPodsCreated(GinkgoT(), kubectlOpts, ctMonitorListOpts, 2, 10, 10*time.Second)

		By("making sure positions file has been used")
		checkLogOutput("app.kubernetes.io/name=ct-monitor", []string{"no new issuances observed"}, []string{"error"})
	})
})
