package nsm

import (
	"fmt"

	"github.com/layer5io/meshkit/errors"
)

var (
	// Error code for failed service mesh installation

	// ErrInstallNSMCode represents the errors which are generated
	// during nsm mesh install process
	ErrInstallNSMCode = "nsm_test_code"

	// ErrCreatingHelmIndexCode represents the errors which are generated
	// during creation of helm index
	ErrCreatingHelmIndexCode = "nsm_test_code"

	// ErrEntryWithAppVersionNotExistsCode represents the error which is generated
	// when no entry is found with specified name and app version
	ErrEntryWithAppVersionNotExistsCode = "nsm_test_code"

	// ErrHelmRepositoryNotFoundCode represents the error which is generated when
	// no valid helm repository is found
	ErrHelmRepositoryNotFoundCode = "nsm_test_code"

	// ErrDecodeYamlCode represents the error which is generated when yaml
	// decode process fails
	ErrDecodeYamlCode = "nsm_test_code"

	// ErrMeshConfigCode represents the errors which are generated
	// when an invalid mesh config is found
	ErrMeshConfigCode = "nsm_test_code"

	// ErrApplyHelmChartCode represents the error which are generated
	// during the process of applying helm chart
	ErrApplyHelmChartCode = "nsm_test_code"

	// ErrConvertingAppVersionToChartVersionCode represents the errors which are generated
	// during the process of converting app version to chart version
	ErrConvertingAppVersionToChartVersionCode = "nsm_test_code"

	// ErrCreatingNSMMeshClientCode represents the errors which are generated
	// during creating nsm client process
	ErrCreatingNSMMeshClientCode = "nsm_test_code"

	// ErrSampleAppCode represents the errors which are generated
	// duing sample app installation
	ErrSampleAppCode = "nsm_test_code"

	// ErrCustomOperationCode represents the errors which are generated
	// when an invalid addon operation is requested
	ErrCustomOperationCode = "nsm_test_code"

	// ErrOpInvalid represents the errors which are generated
	// when an invalid operation is requested
	ErrOpInvalid = errors.NewDefault(errors.ErrOpInvalid, "Invalid operation")
)

// ErrInstallNSM is the error for install mesh
func ErrInstallNSM(err error) error {
	return errors.NewDefault(ErrInstallNSMCode, fmt.Sprintf("Error with nsm operation: %s", err.Error()))
}

// ErrCreatingHelmIndex is the error for creating helm index
func ErrCreatingHelmIndex(err error) error {
	return errors.NewDefault(ErrInstallNSMCode, fmt.Sprintf("Error with nsm operation: %s", err.Error()))
}

// ErrEntryWithAppVersionNotExists is the error when an entry with the given app version is not found
func ErrEntryWithAppVersionNotExists(entry, appVersion string) error {
	return errors.NewDefault(
		ErrEntryWithAppVersionNotExistsCode,
		fmt.Sprintf("entry %s with app version %s does not exists", entry, appVersion),
	)
}

// ErrHelmRepositoryNotFound is the error when no valid remote helm repository is found
func ErrHelmRepositoryNotFound(repo string, err error) error {
	return errors.NewDefault(
		ErrHelmRepositoryNotFoundCode,
		fmt.Sprintf("either the repo %s does not exists or is corrupt: %v", repo, err),
	)
}

// ErrDecodeYaml is the error when the yaml unmarshal fails
func ErrDecodeYaml(err error) error {
	return errors.NewDefault(
		ErrDecodeYamlCode,
		fmt.Sprintf("error decoding yaml: %v", err),
	)
}

// ErrMeshConfig is the error for mesh config
func ErrMeshConfig(err error) error {
	return errors.NewDefault(ErrMeshConfigCode, fmt.Sprintf("Error configuration mesh: %s", err.Error()))
}

// ErrApplyHelmChart is the error for applying helm chart
func ErrApplyHelmChart(err error) error {
	return errors.NewDefault(ErrApplyHelmChartCode, fmt.Sprintf("error applying helm chart: %s", err.Error()))
}

// ErrConvertingAppVersionToChartVersion is the error for converting app version to chart version
func ErrConvertingAppVersionToChartVersion(err error) error {
	return errors.NewDefault(
		ErrConvertingAppVersionToChartVersionCode,
		fmt.Sprintf("error converting app version to chart version: %s", err.Error()),
	)
}

// ErrCreatingNSMMeshClient is the error for streaming event
func ErrCreatingNSMMeshClient(err error) error {
	return errors.NewDefault(
		ErrCreatingNSMMeshClientCode,
		fmt.Sprintf("Unable to create a new nsm client %s", err.Error()),
	)
}

// ErrSampleApp is the error for streaming event
func ErrSampleApp(err error) error {
	return errors.NewDefault(ErrSampleAppCode, fmt.Sprintf("Error with sample app operation: %s", err.Error()))
}

// ErrCustomOperation is the error for streaming event
func ErrCustomOperation(err error) error {
	return errors.NewDefault(ErrCustomOperationCode, fmt.Sprintf("Error with custom operation: %s", err.Error()))
}
