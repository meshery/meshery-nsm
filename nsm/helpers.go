package nsm

import (
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
)

func cloneRepo(repoURL string, destFolder string) error {
	err := createRepoFolder(destFolder)
	if err != nil {
		return err
	}

	// Clone the repository into the temp dir
	logrus.Infof("Cloning repo %f...", repoURL)
	if _, err = git.PlainClone(destFolder, false, &git.CloneOptions{
		URL:          repoURL,
		SingleBranch: true,
		Depth:        1,
	}); err != nil {
		logrus.Errorf("Error Cloning the repo", err)
		return err
	}

	logrus.Infof("Clone of repo %s completed in %s", repoURL, destFolder)
	return nil

}

func createRepoFolder(destFolder string) error {

	_, err := os.Stat(destFolder)
	if os.IsNotExist(err) {
		err := os.MkdirAll(destFolder, os.ModePerm)
		if err != nil {
			err = errors.Wrapf(err, "Unable to create a folder  %s", destFolder)
			logrus.Error(err)
			return err
		}
	}
	return err
}
