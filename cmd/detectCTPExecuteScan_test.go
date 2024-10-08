package cmd

import (
	"github.com/SAP/jenkins-library/pkg/mock"
)

type detectCTPExecuteScanMockUtils struct {
	*mock.ExecMockRunner
	*mock.FilesMock
}

func newDetectCTPExecuteScanTestsUtils() detectCTPExecuteScanMockUtils {
	utils := detectCTPExecuteScanMockUtils{
		ExecMockRunner: &mock.ExecMockRunner{},
		FilesMock:      &mock.FilesMock{},
	}
	return utils
}