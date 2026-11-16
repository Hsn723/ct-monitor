//go:build test
// +build test

package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLuaFilter(t *testing.T) {
	t.Parallel()
	assert.True(t, isLuaFilter("/etc/ct-monitor/filter.lua"))
	assert.True(t, isLuaFilter("FILTER.LUA"))
	assert.False(t, isLuaFilter(testFilterBin))
	assert.False(t, isLuaFilter("/etc/ct-monitor/lua"))
}
