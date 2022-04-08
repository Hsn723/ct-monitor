// +build e2e

package test

import (
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/gruntwork-io/terratest/modules/docker"
)

const (
	tag = "quay.io/hsn723/ct-monitor:latest"
)

func buildAndTestDockerImage(tag string) {
	buildOptions := &docker.BuildOptions{
		Tags: []string{tag},
	}
	docker.Build(GinkgoT(), "../", buildOptions)
	opts := &docker.RunOptions{
		Remove: true,
	}
	out := docker.Run(GinkgoT(), tag, opts)
	Expect(out).To(ContainSubstring("no such file or directory"))
}

func loadDockerImage(tag string) {
	cmd := exec.Command("kind", "load", "docker-image", tag, "--name", "ct-monitor-kindtest")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Second)
	SetDefaultEventuallyPollingInterval(100 * time.Millisecond)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	buildAndTestDockerImage(tag)
	loadDockerImage(tag)
})
