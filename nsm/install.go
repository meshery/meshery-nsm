package nsm

import (
	"fmt"
	"sync"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/status"
	mesherykube "github.com/layer5io/meshkit/utils/kubernetes"
)

func (mesh *Mesh) installNSMMesh(del bool, version, namespace string, kubeconfigs []string) (string, error) {
	mesh.Log.Debug(fmt.Sprintf("Requested install of version: %s", version))
	mesh.Log.Debug(fmt.Sprintf("Requested action is delete: %v", del))
	mesh.Log.Debug(fmt.Sprintf("Requested action is in namespace: %s", namespace))

	st := status.Installing
	if del {
		st = status.Removing
	}

	err := mesh.Config.GetObject(adapter.MeshSpecKey, mesh)
	if err != nil {
		return st, ErrMeshConfig(err)
	}

	if err := mesh.applyHelmChart(version, namespace, del, kubeconfigs); err != nil {
		return st, ErrApplyHelmChart(err)
	}

	st = status.Installed
	if del {
		st = status.Removed
	}

	return st, nil
}

func (mesh *Mesh) applyHelmChart(version, namespace string, isDel bool, kubeconfigs []string) error {
	repo := "https://helm.nsm.dev/"

	var act mesherykube.HelmChartAction
	if isDel {
		act = mesherykube.UNINSTALL
	} else {
		act = mesherykube.INSTALL
	}
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
			err = kClient.ApplyHelmChart(mesherykube.ApplyHelmChartConfig{
				ChartLocation: mesherykube.HelmChartLocation{
					Repository: repo,
					// Chart:      chart,
					Version: version,
				},
				Namespace:       namespace,
				Action:          act,
				CreateNamespace: true,
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
		return mergeErrors(errs)
	}
	return nil
}
