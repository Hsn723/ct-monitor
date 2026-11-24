//go:build test
// +build test

package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/Hsn723/ct-monitor/config"
	mock_monitor "github.com/Hsn723/ct-monitor/mock/monitor"
	"github.com/Hsn723/ct-monitor/monitor"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	testDomain      = "example.com"
	testOtherDomain = "example.net"
	testConfigFile  = "test/config.toml"
	testConfig      = `certspotter_endpoint = "https://api.example.com/v1/issuances"

[position_config]
filename = "position.toml"

[[domain]]
name = "example.com"

[[domain]]
name = "example.net"`

	testBarebonesConfig = `certspotter_endpoint = "https://api.example.com/v1/issuances"

[position_config]
filename = "position.toml"`
)

func setupMockMonitor(t *testing.T, monitorErrs []error, isCalled bool) *mock_monitor.MockIMonitor {
	ctrl := gomock.NewController(t)
	mockMonitor := mock_monitor.NewMockIMonitor(ctrl)
	err := errors.Join(monitorErrs...)
	times := 0
	if isCalled {
		times = 1
	}
	mockMonitor.EXPECT().Run().Return(err).Times(times)
	return mockMonitor
}

func TestRunRoot(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title           string
		isConfigPresent bool
		isConfigValid   bool
		monitorErrs     []error
		assertErr       assert.ErrorAssertionFunc
	}{
		{
			title:           "Success",
			isConfigPresent: true,
			isConfigValid:   true,
			assertErr:       assert.NoError,
		},
		{
			title:           "PartialFailure",
			isConfigPresent: true,
			isConfigValid:   true,
			monitorErrs:     []error{monitor.ErrCheckIssuances},
			assertErr:       assert.Error,
		},
		{
			title: "ConfigNotPresent",
			assertErr: func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
				return assert.ErrorIs(t, err, os.ErrNotExist, msgAndArgs...)
			},
		},
		{
			title:           "ConfigInvalid",
			isConfigPresent: true,
			assertErr:       assert.Error,
		},
		{
			title:           "MonitorFailure",
			isConfigPresent: true,
			isConfigValid:   true,
			monitorErrs:     []error{monitor.ErrCheckIssuances},
			assertErr: func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
				return assert.ErrorIs(t, err, monitor.ErrCheckIssuances, msgAndArgs...)
			},
		},
		{
			title:           "OtherFailure",
			isConfigPresent: true,
			isConfigValid:   true,
			monitorErrs:     []error{assert.AnError},
			assertErr:       assert.Error,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			fs := afero.NewMemMapFs()
			if tc.isConfigPresent {
				var content string
				if tc.isConfigValid {
					content = testConfig
				} else {
					content = "invalid config"
				}
				err := afero.WriteFile(fs, testConfigFile, []byte(content), 0644)
				assert.NoError(t, err)
			}
			mockMonitor := setupMockMonitor(t, tc.monitorErrs, tc.isConfigValid)
			app := &app{
				fs:         fs,
				configFile: testConfigFile,
				monitor:    mockMonitor,
			}
			tc.assertErr(t, app.runRoot(nil, nil))
		})
	}
}

func TestRunRootMonitorInstantiation(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, testConfigFile, []byte(testBarebonesConfig), 0644)
	assert.NoError(t, err)

	app := &app{
		fs:         fs,
		configFile: testConfigFile,
	}
	assert.NoError(t, app.runRoot(nil, nil))
	assert.IsType(t, &monitor.Monitor{}, app.monitor)
}

func TestRootCmd(t *testing.T) {
	t.Parallel()
	app := &app{
		fs: afero.NewMemMapFs(),
	}
	cmd := app.RootCmd()
	assert.Equal(t, "ct-monitor", cmd.Use)
	assert.Equal(t, "ct-monitor queries the certspotter API for new certificate issuances", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.True(t, cmd.Flags().Lookup("config") != nil)
	assert.Equal(t, config.DefaultConfigFile, cmd.Flags().Lookup("config").Value.String())
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
}
