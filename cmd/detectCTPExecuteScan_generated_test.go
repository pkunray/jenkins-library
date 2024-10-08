//go:build unit
// +build unit

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectCTPExecuteScanCommand(t *testing.T) {
	t.Parallel()

	testCmd := DetectCTPExecuteScanCommand()

	// only high level testing performed - details are tested in step generation procedure
	assert.Equal(t, "detectCTPExecuteScan", testCmd.Use, "command name incorrect")

}
