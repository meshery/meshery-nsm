// Copyright 2019 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nsm

import (
	// "io"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	git "gopkg.in/src-d/go-git.v4" // Apache v2.0

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	repoURL     = "https://github.com/networkservicemesh/networkservicemesh.git"
	cachePeriod = 24 * time.Hour
)

func (nsmClient *NSMClient) downloadNSM() {
	fmt.Println("Attempting to download the NSM project...")
	//        Info("git clone %s %s --recursive", "https://github.com/networkservicemesh/networkservicemesh.git","/home/harshini/Documents/networkservicemesh")

	// Create temp directory to clone the repository within.
	fmt.Println("Creating temporary directory for repo clone...")
	dir, err := ioutil.TempDir("", "temp-clone")
	if err != nil {
		err = errors.Wrapf(err, "Unable to create temporary directory on the filesystem")
		logrus.Error(err)
	}

	// CLean up temporary directory when done.
	defer os.RemoveAll(dir)

	// Clone the repository into the temp dir
	fmt.Println("Cloning NSM repo...")
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: repoURL,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Clone of NSM repo complete.")
}
