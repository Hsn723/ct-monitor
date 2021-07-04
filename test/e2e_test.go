package test

import (
	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
	"time"
)

func buildAndTestDockerImage(tag string, t *testing.T) {
	t.Helper()
	buildOptions := &docker.BuildOptions{
		Tags: []string{tag},
	}
	docker.Build(t, "../", buildOptions)
	opts := &docker.RunOptions{
		Remove: true,
	}
	out := docker.Run(t, tag, opts)
	assert.Contains(t, out, "alert mail recipient missing")
}

func loadDockerImage(tag string, t *testing.T) {
	t.Helper()
	cmd := exec.Command("kind", "load", "docker-image", tag, "--name", "ct-monitor-kindtest")
	err := cmd.Run()
	assert.NoError(t, err)
}

func getConfigPath(t *testing.T) string {
	t.Helper()
	usr, err := user.Current()
	assert.NoError(t, err)
	return filepath.Join(usr.HomeDir, ".kube", "config")
}

func TestKubernetes(t *testing.T) {
	tag := "quay.io/hsn723/ct-monitor:latest"
	buildAndTestDockerImage(tag, t)
	loadDockerImage(tag, t)

	kubeResourcePath := "./install.yaml"
	options := k8s.NewKubectlOptions("kind-ct-monitor-kindtest", getConfigPath(t), "default")
	k8s.KubectlApply(t, options, kubeResourcePath)

	defer k8s.KubectlDelete(t, options, kubeResourcePath)
	listOptions := metav1.ListOptions{
		FieldSelector: "status.phase==Succeeded",
	}
	k8s.WaitUntilNumPodsCreated(t, options, listOptions, 1, 10, 10*time.Second)
}
