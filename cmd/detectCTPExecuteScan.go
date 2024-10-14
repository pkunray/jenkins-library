package cmd

import (
	"io"
	"net/http"

	"github.com/SAP/jenkins-library/pkg/command"
	piperDocker "github.com/SAP/jenkins-library/pkg/docker"
	piperhttp "github.com/SAP/jenkins-library/pkg/http"
	"github.com/SAP/jenkins-library/pkg/log"
	"github.com/SAP/jenkins-library/pkg/orchestrator"
	"github.com/SAP/jenkins-library/pkg/piperutils"
	"github.com/SAP/jenkins-library/pkg/telemetry"
	"github.com/google/go-github/v45/github"
)

type detectCTPExecuteScanUtils interface {
	command.ExecRunner
	piperutils.FileUtils

	GetExitCode() int
	GetOsEnv() []string
	Stdout(out io.Writer)
	Stderr(err io.Writer)
	SetDir(dir string)
	SetEnv(env []string)
	RunExecutable(e string, p ...string) error
	RunShell(shell, script string) error

	DownloadFile(url, filename string, header http.Header, cookies []*http.Cookie) error

	GetIssueService() *github.IssuesService
	GetSearchService() *github.SearchService
	GetProvider() orchestrator.ConfigProvider
	GetDockerClient(options piperDocker.ClientOptions) piperDocker.Download

	FileExists(filename string) (bool, error)
}

type detectCTPExecuteScanUtilsBundle struct {
	*command.Command
	*piperutils.Files
	*piperhttp.Client
	issues   *github.IssuesService
	search   *github.SearchService
	provider orchestrator.ConfigProvider
}

func (d *detectCTPExecuteScanUtilsBundle) GetIssueService() *github.IssuesService {
	return d.issues
}

func (d *detectCTPExecuteScanUtilsBundle) GetSearchService() *github.SearchService {
	return d.search
}

func (d *detectCTPExecuteScanUtilsBundle) GetProvider() orchestrator.ConfigProvider {
	return d.provider
}

func (d *detectCTPExecuteScanUtilsBundle) GetDockerClient(options piperDocker.ClientOptions) piperDocker.Download {
	client := &piperDocker.Client{}
	client.SetOptions(options)

	return client
}

func newDetectCTPExecuteScanUtils(client *github.Client) detectCTPExecuteScanUtils {
	utils := detectCTPExecuteScanUtilsBundle{
		Command: &command.Command{
			ErrorCategoryMapping: map[string][]string{
				log.ErrorCompliance.String(): {
					"FAILURE_POLICY_VIOLATION - Detect found policy violations.",
				},
				log.ErrorConfiguration.String(): {
					"FAILURE_CONFIGURATION - Detect was unable to start due to issues with it's configuration.",
					"FAILURE_DETECTOR - Detect had one or more detector failures while extracting dependencies. Check that all projects build and your environment is configured correctly.",
					"FAILURE_SCAN - Detect was unable to run the signature scanner against your source. Check your configuration.",
				},
				log.ErrorInfrastructure.String(): {
					"FAILURE_PROXY_CONNECTIVITY - Detect was unable to use the configured proxy. Check your configuration and connection.",
					"FAILURE_BLACKDUCK_CONNECTIVITY - Detect was unable to connect to Black Duck. Check your configuration and connection.",
					"FAILURE_POLARIS_CONNECTIVITY - Detect was unable to connect to Polaris. Check your configuration and connection.",
				},
				log.ErrorService.String(): {
					"FAILURE_TIMEOUT - Detect could not wait for actions to be completed on Black Duck. Check your Black Duck server or increase your timeout.",
					"FAILURE_DETECTOR_REQUIRED - Detect did not run all of the required detectors. Fix detector issues or disable required detectors.",
					"FAILURE_BLACKDUCK_VERSION_NOT_SUPPORTED - Detect attempted an operation that was not supported by your version of Black Duck. Ensure your Black Duck is compatible with this version of detect.",
					"FAILURE_BLACKDUCK_FEATURE_ERROR - Detect encountered an error while attempting an operation on Black Duck. Ensure your Black Duck is compatible with this version of detect.",
					"FAILURE_GENERAL_ERROR - Detect encountered a known error, details of the error are provided.",
					"FAILURE_UNKNOWN_ERROR - Detect encountered an unknown error.",
					"FAILURE_MINIMUM_INTERVAL_NOT_MET - Detect did not wait the minimum required scan interval.",
				},
			},
		},
		Files:  &piperutils.Files{},
		Client: &piperhttp.Client{},
	}
	if client != nil {
		utils.issues = client.Issues
		utils.search = client.Search
	}

	utils.Stdout(log.Writer())
	utils.Stderr(log.Writer())

	provider, err := orchestrator.GetOrchestratorConfigProvider(nil)
	if err != nil {
		log.Entry().WithError(err).Warn("failed to get orchestrator config provider")
	}

	utils.provider = provider

	return &utils
}

func detectCTPExecuteScan(config detectCTPExecuteScanOptions, telemetryData *telemetry.CustomData, influx *detectCTPExecuteScanInflux) {
	detectConfig := detectExecuteScanOptions(config)
	detectInflux := detectExecuteScanInflux(*influx)

	detectExecuteScan(detectConfig, telemetryData, &detectInflux)
}
