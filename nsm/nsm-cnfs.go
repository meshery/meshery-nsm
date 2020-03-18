//Package nsm ...
package nsm

import (
	"io/ioutil"
	"log"
	"os"
	"path"
)

const (
	examplesRepoURL = "https://github.com/networkservicemesh/examples.git"
)

var (
	examplesDestDir = path.Join(os.TempDir(), "NSMExamples")
	appsDir         = examplesDestDir + "/examples"
)

func downloadExampleApps() []string {
	err := cloneCNFExampleApps()
	if err != nil {
		log.Fatalf("failed to clone CNF example apps %v", err)
	}
	examplesList, err := generatExampleAppsCommands()
	if err != nil {
		log.Fatalf("failed to generat example apps commands %v", err)
	}
	return examplesList

}

func cloneCNFExampleApps() error {
	err := cloneRepo(examplesRepoURL, examplesDestDir)

	return err
}

func generatExampleAppsCommands() ([]string, error) {

	files, err := ioutil.ReadDir("./" + appsDir)
	if err != nil {
		log.Fatalf("failed to read dir %s, %v", appsDir, err)
	}

	examplesList := []string{}

	for _, f := range files {
		if f.IsDir() {
			examplesList = append(examplesList, f.Name())
		}
	}

	return examplesList, nil
}
