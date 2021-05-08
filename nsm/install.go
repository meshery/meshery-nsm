package nsm

import (
	"fmt"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/status"
	mesherykube "github.com/layer5io/meshkit/utils/kubernetes"
)

func (mesh *Mesh) installNSMMesh(del bool, version, namespace string) (string, error) {
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

	err = mesh.applyHelmChart(del, version, namespace, "nsm")
	if err != nil {
		return st, ErrApplyHelmChart(err)
	}

	st = status.Installed
	if del {
		st = status.Removed
	}

	return st, nil
}

func (mesh *Mesh) applyHelmChart(del bool, version, namespace, chart string) error {
	kClient := mesh.MesheryKubeclient

	repo := "https://helm.nsm.dev/"

	err := kClient.ApplyHelmChart(mesherykube.ApplyHelmChartConfig{
		ChartLocation: mesherykube.HelmChartLocation{
			Repository: repo,
			Chart:      chart,
			Version:    version,
		},
		Namespace:       namespace,
		Delete:          del,
		CreateNamespace: true,
	})

	return err
}
