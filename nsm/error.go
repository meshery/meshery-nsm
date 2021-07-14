package nsm

import (
	"github.com/layer5io/meshkit/errors"
)

var (
	// Error code for failed service mesh installation

	// ErrInstallNSMCode represents the errors which are generated
	// during nsm mesh install process
	ErrInstallNSMCode = "1003"

	// ErrCreatingHelmIndexCode represents the errors which are generated
	// during creation of helm index
	ErrCreatingHelmIndexCode = "1004"

	// ErrEntryWithAppVersionNotExistsCode represents the error which is generated
	// when no entry is found with specified name and app version
	ErrEntryWithAppVersionNotExistsCode = "1005"

	// ErrHelmRepositoryNotFoundCode represents the error which is generated when
	// no valid helm repository is found
	ErrHelmRepositoryNotFoundCode = "1006"

	// ErrDecodeYamlCode represents the error which is generated when yaml
	// decode process fails
	ErrDecodeYamlCode = "1007"

	// ErrMeshConfigCode represents the errors which are generated
	// when an invalid mesh config is found
	ErrMeshConfigCode = "1008"

	// ErrApplyHelmChartCode represents the error which are generated
	// during the process of applying helm chart
	ErrApplyHelmChartCode = "1009"

	// ErrConvertingAppVersionToChartVersionCode represents the errors which are generated
	// during the process of converting app version to chart version
	ErrConvertingAppVersionToChartVersionCode = "1010"

	// ErrCreatingNSMMeshClientCode represents the errors which are generated
	// during creating nsm client process
	ErrCreatingNSMMeshClientCode = "1011"

	// ErrSampleAppCode represents the errors which are generated
	// duing sample app installation
	ErrSampleAppCode = "1012"

	// ErrCustomOperationCode represents the errors which are generated
	// when an invalid addon operation is requested
	ErrCustomOperationCode = "1013"

	// ErrOpInvalidCode represents the errors which are generated
	// when an invalid operation is requested
	ErrOpInvalidCode = "1014"

	// ErrOpInvalid represents the errors which are generated
	// when an invalid operation is requested
	ErrOpInvalid = errors.New(ErrOpInvalidCode, errors.Alert, []string{"Invalid operation"}, []string{}, []string{}, []string{})
)

// ErrInstallNSM is the error for install mesh
func ErrInstallNSM(err error) error {
	return errors.New(ErrInstallNSMCode, errors.Alert, []string{"Error with nsm operation: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrCreatingHelmIndex is the error for creating helm index
func ErrCreatingHelmIndex(err error) error {
	return errors.New(ErrCreatingHelmIndexCode, errors.Alert, []string{"Error with nsm operation: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrEntryWithAppVersionNotExists is the error when an entry with the given app version is not found
func ErrEntryWithAppVersionNotExists(entry, appVersion string) error {
	return errors.New(ErrEntryWithAppVersionNotExistsCode, errors.Alert, []string{"Entry: ", entry, "with app version : ", appVersion, "does not exists"}, []string{}, []string{}, []string{})
}

// ErrHelmRepositoryNotFound is the error when no valid remote helm repository is found
func ErrHelmRepositoryNotFound(repo string, err error) error {
	return errors.New(ErrHelmRepositoryNotFoundCode, errors.Alert, []string{"Either the repo : ", repo, "does not exists or is corrupt: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrDecodeYaml is the error when the yaml unmarshal fails
func ErrDecodeYaml(err error) error {
	return errors.New(ErrDecodeYamlCode, errors.Alert, []string{"Error decoding yaml: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrMeshConfig is the error for mesh config
func ErrMeshConfig(err error) error {
	return errors.New(ErrMeshConfigCode, errors.Alert, []string{"Error configuration mesh: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrApplyHelmChart is the error for applying helm chart
func ErrApplyHelmChart(err error) error {
	return errors.New(ErrApplyHelmChartCode, errors.Alert, []string{"Error applying helm chart: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrConvertingAppVersionToChartVersion is the error for converting app version to chart version
func ErrConvertingAppVersionToChartVersion(err error) error {
	return errors.New(ErrConvertingAppVersionToChartVersionCode, errors.Alert, []string{"Error converting app version to chart version: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrCreatingNSMMeshClient is the error for streaming event
func ErrCreatingNSMMeshClient(err error) error {
	return errors.New(ErrCreatingNSMMeshClientCode, errors.Alert, []string{"Unable to create a new nsm client: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrSampleApp is the error for streaming event
func ErrSampleApp(err error) error {
	return errors.New(ErrSampleAppCode, errors.Alert, []string{"Error with sample app operation: ", err.Error()}, []string{}, []string{}, []string{})
}

// ErrCustomOperation is the error for streaming event
func ErrCustomOperation(err error) error {
	return errors.New(ErrCustomOperationCode, errors.Alert, []string{"Error with custom operation: ", err.Error()}, []string{}, []string{}, []string{})
}
