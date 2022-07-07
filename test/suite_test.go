//go:build e2e
// +build e2e

package test

import (
	"os/exec"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/docker"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	tag             = "quay.io/hsn723/ct-monitor:latest"
	mockUpstreamTag = "quay.io/hsn723/mock-upstream:latest"
)

func buildDockerImage(tag, path string) {
	buildOptions := &docker.BuildOptions{
		Tags: []string{tag},
	}
	docker.Build(GinkgoT(), path, buildOptions)
}

func testDockerImage(tag string) {
	opts := &docker.RunOptions{
		Remove: true,
	}
	out := docker.Run(GinkgoT(), tag, opts)
	Expect(out).To(ContainSubstring("no such file or directory"))
}

func buildAndTestDockerImage(tag, path string) {
	buildDockerImage(tag, path)
	testDockerImage(tag)
}

func loadDockerImage(tag string) {
	cmd := exec.Command("kind", "load", "docker-image", tag, "--name", "ct-monitor-kindtest")
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	buildAndTestDockerImage(tag, "../")
	loadDockerImage(tag)
	buildDockerImage(mockUpstreamTag, "mockupstream/")
	loadDockerImage(mockUpstreamTag)
})
