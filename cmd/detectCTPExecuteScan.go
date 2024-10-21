package cmd

import (
	"context"
	"fmt"
	"strings"

	piperGithub "github.com/SAP/jenkins-library/pkg/github"
	"github.com/SAP/jenkins-library/pkg/golang"
	"github.com/SAP/jenkins-library/pkg/log"
	"github.com/SAP/jenkins-library/pkg/maven"
	"github.com/SAP/jenkins-library/pkg/npm"
	"github.com/SAP/jenkins-library/pkg/telemetry"
	"github.com/pkg/errors"
)

func detectCTPExecuteScan(config detectCTPExecuteScanOptions, _ *telemetry.CustomData, influx *detectCTPExecuteScanInflux) {

	influx.step_data.fields.detect = false

	ctx, client, err := piperGithub.
		NewClientBuilder(config.GithubToken, config.GithubAPIURL).
		WithTrustedCerts(config.CustomTLSCertificateLinks).Build()
	if err != nil {
		log.Entry().WithError(err).Warning("Failed to get GitHub client")
	}

	// Log config and workspace content for debug purpose
	if log.IsVerbose() {
		logConfigInVerboseMode(detectExecuteScanOptions(config))
		logWorkspaceContent()
	}

	if config.PrivateModules != "" && config.PrivateModulesGitToken != "" {
		//configuring go private packages
		if err := golang.PrepareGolangPrivatePackages("detectExecuteStep", config.PrivateModules, config.PrivateModulesGitToken); err != nil {
			log.Entry().Warningf("couldn't set private packages for golang, error: %s", err.Error())
		}
	}

	utils := newDetectUtils(client)
	if err := runDetect4CTP(ctx, config, utils, influx); err != nil {
		log.Entry().
			WithError(err).
			Fatal("failed to execute detect scan")
	}

	influx.step_data.fields.detect = true
}

func runDetect4CTP(ctx context.Context, config4CTP detectCTPExecuteScanOptions, utils detectUtils, influx4CTP *detectCTPExecuteScanInflux) error {
	// detect execution details, see https://synopsys.atlassian.net/wiki/spaces/INTDOCS/pages/88440888/Sample+Synopsys+Detect+Scan+Configuration+Scenarios+for+Black+Duck

	config := detectExecuteScanOptions(config4CTP)
	influx := detectExecuteScanInflux(*influx4CTP)

	err := getDetectScript(config, utils)
	if err != nil {
		return fmt.Errorf("failed to download 'detect.sh' script: %w", err)
	}
	defer func() {
		err := utils.FileRemove("detect.sh")
		if err != nil {
			log.Entry().Warnf("failed to delete 'detect.sh' script: %v", err)
		}
	}()
	err = utils.Chmod("detect.sh", 0o700)
	if err != nil {
		return err
	}

	if config.InstallArtifacts {

		log.Entry().Infof("#### installArtifacts - start")
		err := maven.InstallMavenArtifacts(&maven.EvaluateOptions{
			M2Path:              config.M2Path,
			ProjectSettingsFile: config.ProjectSettingsFile,
			GlobalSettingsFile:  config.GlobalSettingsFile,
		}, utils)
		if err != nil {
			return err
		}
		log.Entry().Infof("#### installArtifacts - end")
	}

	if config.BuildMaven {
		log.Entry().Infof("#### BuildMaven - start")
		mavenConfig := setMavenConfig(config)
		mavenUtils := maven.NewUtilsBundle()

		err := runMavenBuild(&mavenConfig, nil, mavenUtils, &mavenBuildCommonPipelineEnvironment{})
		if err != nil {
			return err
		}
		log.Entry().Infof("#### BuildMaven - end")
	}

	// Install NPM dependencies
	if config.InstallNPM {
		log.Entry().Infof("#### InstallNPM - start")
		npmExecutor := npm.NewExecutor(npm.ExecutorOptions{DefaultNpmRegistry: config.DefaultNpmRegistry})

		buildDescriptorList := config.BuildDescriptorList
		if len(buildDescriptorList) == 0 {
			buildDescriptorList = []string{"package.json"}
		}

		err := npmExecutor.InstallAllDependencies(buildDescriptorList)
		if err != nil {
			return err
		}
		log.Entry().Infof("#### InstallNPM - end")
	}

	// for MTA
	if config.BuildMTA {
		log.Entry().Infof("#### BuildMTA - start")
		mtaConfig := setMTAConfig(config)
		mtaUtils := newMtaBuildUtilsBundle()

		err := runMtaBuild(mtaConfig, &mtaBuildCommonPipelineEnvironment{}, mtaUtils)
		if err != nil {
			return err
		}
		log.Entry().Infof("#### BuildMTA - end")
	}

	blackduckSystem := newBlackduckSystem(config)

	args := []string{"./detect.sh"}
	args, err = addDetectArgs(args, config, utils, blackduckSystem, NO_VERSION_SUFFIX, NO_VERSION_SUFFIX)
	if err != nil {
		return err
	}
	script := strings.Join(args, " ")

	envs := []string{"BLACKDUCK_SKIP_PHONE_HOME=true"}
	envs = append(envs, config.CustomEnvironmentVariables...)

	utils.SetDir(".")
	utils.SetEnv(envs)

	err = mapDetectError(utils.RunShell("/bin/bash", script), config, utils)
	if config.ScanContainerDistro != "" {
		imageError := mapDetectError(runDetectImages(ctx, config, utils, blackduckSystem, &influx, blackduckSystem), config, utils)
		if imageError != nil {
			if err != nil {
				err = errors.Wrapf(err, "error during scanning images: %q", imageError.Error())
			} else {
				err = imageError
			}
		}
	}

	return err
}
