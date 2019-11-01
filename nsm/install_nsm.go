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
	"context"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/manifest"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/timeconv"
)

const (
	repoURL = "https://github.com/networkservicemesh/networkservicemesh.git"
)

var (
	destinationFolder = path.Join(os.TempDir(), "NetworkServiceMesh")
)

// Git clones the networkservicemesh repo
func (nsmClient *Client) downloadNSM() error {
	_, err := os.Stat(destinationFolder)
	if os.IsNotExist(err) {
		err := os.MkdirAll(destinationFolder, os.ModePerm)
		if err != nil {
			err = errors.Wrapf(err, "Unable to create a folder  %s", destinationFolder)
			logrus.Error(err)
			return err
		}

		// CLean up temporary directory when done.
		//	defer os.RemoveAll(dir)

		// Clone the repository into the temp dir
		logrus.Infof("Cloning NSM repo...")
		if _, err = git.PlainClone(destinationFolder, false, &git.CloneOptions{
			URL: repoURL,
		}); err != nil {
			logrus.Errorf("Error Cloning the repo", err)
			return err
		}

		logrus.Infof("Clone of NSM repo completed in %s", destinationFolder)
		return nil
	}
	return err
}

func renderManifests(ctx context.Context, c *chart.Chart, releaseName, namespace, kubeVersion, customConfig string) ([]manifest.Manifest, error) {
	if strings.TrimSpace(customConfig) != "" {
		c.Values.Raw = customConfig
	}
	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      releaseName,
			IsInstall: true,
			Time:      timeconv.Now(),
			Namespace: namespace,
		},
		KubeVersion: kubeVersion,
	}
	config := &chart.Config{Raw: "{}", Values: map[string]*chart.Value{}}

	renderedTemplates, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return nil, err
	}
	manifests := manifest.SplitManifests(renderedTemplates)
	return tiller.SortByKind(manifests), nil
}
