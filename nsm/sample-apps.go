package nsm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/status"
	mesherykube "github.com/layer5io/meshkit/utils/kubernetes"
)

func (mesh *Mesh) installNSMSampleApp(del bool, version, namespace string, kubeconfigs []string) (string, error) {
	st := status.Installing

	if del {
		st = status.Removing
	}

	if err := mesh.applyHelmChart(version, namespace, del, kubeconfigs); err != nil {
		return st, ErrSampleApp(err)
	}

	return status.Installed, nil
}

func (mesh *Mesh) installSampleApp(namespace string, del bool, templates []adapter.Template, kubeconfigs []string) (string, error) {
	st := status.Installing

	if del {
		st = status.Removing
	}

	for _, template := range templates {
		err := mesh.applyManifest([]byte(template.String()), del, namespace, kubeconfigs)
		if err != nil {
			return st, ErrSampleApp(err)
		}
	}

	return status.Installed, nil
}

func (mesh *Mesh) applyManifest(contents []byte, isDel bool, namespace string, kubeconfigs []string) error {
	var wg sync.WaitGroup
	var errs []error
	var errMx sync.Mutex
	for _, config := range kubeconfigs {
		wg.Add(1)
		go func(config string) {
			defer wg.Done()
			kClient, err := mesherykube.New([]byte(config))
			if err != nil {
				errMx.Lock()
				errs = append(errs, err)
				errMx.Unlock()
				return
			}
			err = kClient.ApplyManifest(contents, mesherykube.ApplyOptions{
				Namespace:    namespace,
				Update:       true,
				Delete:       isDel,
				IgnoreErrors: true,
			})
			if err != nil {
				errMx.Lock()
				errs = append(errs, err)
				errMx.Unlock()
				return
			}
		}(config)
	}
	wg.Wait()

	if len(errs) != 0 {
		return ErrLoadNamespace(mergeErrors(errs), namespace)
	}
	return nil
}

func mergeErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	var errMsgs []string

	for _, err := range errs {
		errMsgs = append(errMsgs, err.Error())
	}

	return fmt.Errorf(strings.Join(errMsgs, "\n"))
}
