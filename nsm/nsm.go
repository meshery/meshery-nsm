// Package nsm - Common operations for the adapter
package nsm

import (
	"context"
	"fmt"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshery-adapter-library/common"
	adapterconfig "github.com/layer5io/meshery-adapter-library/config"
	"github.com/layer5io/meshery-adapter-library/meshes"
	"github.com/layer5io/meshery-adapter-library/status"
	internalconfig "github.com/layer5io/meshery-nsm/internal/config"
	"github.com/layer5io/meshkit/errors"
	"github.com/layer5io/meshkit/logger"
	"github.com/layer5io/meshkit/utils/events"
)

const (
	// SMIManifest is the manifest.yaml file for smi conformance tool
	SMIManifest = "https://raw.githubusercontent.com/layer5io/learn-layer5/master/smi-conformance/manifest.yml"
)

// Mesh represents the nsm-mesh adapter and embeds adapter.Adapter
type Mesh struct {
	adapter.Adapter // Type Embedded
}

// New initializes treafik-mesh handler.
func New(c adapterconfig.Handler,
	l logger.Handler,
	kc adapterconfig.Handler,
	ev *events.EventStreamer,
) adapter.Handler {
	return &Mesh{
		Adapter: adapter.Adapter{
			Config:            c,
			KubeconfigHandler: kc,
			Log:               l,
			EventStreamer:     ev,
		},
	}
}

// ApplyOperation applies the operation on nsm mesh
func (mesh *Mesh) ApplyOperation(ctx context.Context, opReq adapter.OperationRequest) error {
	kubeConfigs := opReq.K8sConfigs
	operations := make(adapter.Operations)
	err := mesh.Config.GetObject(adapter.OperationsKey, &operations)
	if err != nil {
		return err
	}

	e := &meshes.EventsResponse{
		OperationId:   opReq.OperationID,
		Summary:       status.Deploying,
		Details:       "Operation is not supported",
		Component:     internalconfig.ServerConfig["type"],
		ComponentName: internalconfig.ServerConfig["name"],
	}

	switch opReq.OperationName {
	case internalconfig.NSMMeshOperation:
		go func(hh *Mesh, ee *meshes.EventsResponse) {
			version := string(operations[opReq.OperationName].Versions[0])
			stat, err := hh.installNSMMesh(opReq.IsDeleteOperation, version, opReq.Namespace, kubeConfigs)
			if err != nil {
				summary := fmt.Sprintf("Error while %s NSM service mesh", stat)
				e.Details = err.Error()
				hh.streamErr(summary, ee, err)
				return
			}
			ee.Summary = fmt.Sprintf("NSM service mesh %s successfully", stat)
			ee.Details = fmt.Sprintf("The NSM service mesh is now %s.", stat)
			hh.StreamInfo(e)
		}(mesh, e)
	case common.BookInfoOperation, common.HTTPBinOperation, common.ImageHubOperation, common.EmojiVotoOperation:
		go func(hh *Mesh, ee *meshes.EventsResponse) {
			appName := operations[opReq.OperationName].AdditionalProperties[common.ServiceName]
			stat, err := hh.installSampleApp(opReq.Namespace, opReq.IsDeleteOperation, operations[opReq.OperationName].Templates, kubeConfigs)
			if err != nil {
				summary := fmt.Sprintf("Error while %s %s application", stat, appName)
				e.Details = err.Error()
				hh.streamErr(summary, ee, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s application %s successfully", appName, stat)
			ee.Details = fmt.Sprintf("The %s application is now %s.", appName, stat)
			hh.StreamInfo(e)
		}(mesh, e)
	case common.CustomOperation:
		go func(hh *Mesh, ee *meshes.EventsResponse) {
			stat, err := hh.applyCustomOperation(opReq.Namespace, opReq.CustomBody, opReq.IsDeleteOperation, kubeConfigs)
			if err != nil {
				summary := fmt.Sprintf("Error while %s custom operation", stat)
				e.Details = err.Error()
				hh.streamErr(summary, ee, err)
				return
			}
			ee.Summary = fmt.Sprintf("Manifest %s successfully", status.Deployed)
			ee.Details = ""
			hh.StreamInfo(e)
		}(mesh, e)
	case internalconfig.NSMICMPResponderSampleApp, internalconfig.NSMVPPICMPResponderSampleApp, internalconfig.NSMVPMSampleApp:
		go func(hh *Mesh, ee *meshes.EventsResponse) {
			version := string(operations[internalconfig.NSMMeshOperation].Versions[0])
			// chart := operations[opReq.OperationName].AdditionalProperties[internalconfig.HelmChart]
			appName := operations[opReq.OperationName].AdditionalProperties[common.ServiceName]

			stat, err := hh.installNSMSampleApp(opReq.IsDeleteOperation, version, opReq.Namespace, kubeConfigs)
			if err != nil {
				summary := fmt.Sprintf("Error while %s %s application", stat, appName)
				e.Details = err.Error()
				hh.streamErr(summary, ee, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s application %s successfully", appName, stat)
			ee.Details = fmt.Sprintf("The %s application is now %s.", appName, stat)
			hh.StreamInfo(e)
		}(mesh, e)
	case common.SmiConformanceOperation:
		go func(hh *Mesh, ee *meshes.EventsResponse) {
			name := operations[opReq.OperationName].Description
			_, err := hh.RunSMITest(adapter.SMITestOptions{
				Ctx:         context.TODO(),
				OperationID: ee.OperationId,
				Namespace:   "meshery",
				Manifest:    string(operations[opReq.OperationName].Templates[0]),
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Kubeconfigs: []string{},
			})
			if err != nil {
				summary := fmt.Sprintf("Error while %s %s test", status.Running, name)
				e.Details = err.Error()
				hh.streamErr(summary, ee, err)
				return
			}
			ee.Summary = fmt.Sprintf("%s test %s successfully", name, status.Completed)
			ee.Details = ""
			hh.StreamInfo(e)
		}(mesh, e)
	default:
		summary := "invalid request"
		mesh.streamErr(summary, e, ErrOpInvalid)
	}

	return nil
}

func (mesh *Mesh) streamErr(summary string, e *meshes.EventsResponse, err error) {
	e.Summary = summary
	e.Details = err.Error()
	e.ErrorCode = errors.GetCode(err)
	e.ProbableCause = errors.GetCause(err)
	e.SuggestedRemediation = errors.GetRemedy(err)
	mesh.StreamErr(e, err)
}
