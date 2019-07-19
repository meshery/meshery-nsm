package nsm

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	. "gopkg.in/src-d/go-git.v4/_examples"
)

var (
	destinationFolder = path.Join(os.TempDir(), "NSM")
)

func (nsmClient *NSMClient) downloadNSM() {
	fmt.Println("downloading the project NSM ")
	//        Info("git clone %s %s --recursive", "https://github.com/networkservicemesh/networkservicemesh.git","/home/harshini/Documents/networkservicemesh")

	fmt.Println("Creating new directory in ", destinationFolder)
	dFile, err := os.Create(destinationFolder)
	if err != nil {
		err = errors.Wrapf(err, "unable to create a file on the filesystem at %s", destinationFolder)
		logrus.Error(err)

	}
	defer dFile.Close()
	
	fmt.Println("Created a new folder ")

	cloneOptions := git.CloneOptions{
		URL:           "https://github.com/networkservicemesh/networkservicemesh",
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	start := time.Now()
	r, err := git.PlainClone(destinationFolder, false, &cloneOptions)
	elapsed := time.Now().Sub(start)
	logrus.Printf("Cloned in %s in the path %s", elapsed,destinationFolder)
        logrus.Printf("The repo object is %s ",r)
        //CheckIfError(err)

	// Getting the latest commit on the current branch
	Info("git log -1")

	// ... retrieving the branch being pointed by HEAD
	ref, err := r.Head()
	//CheckIfError(err)

	// ... retrieving the commit object
	commit, err := r.CommitObject(ref.Hash())
	//CheckIfError(err)
	fmt.Println(commit)

        

	

}
