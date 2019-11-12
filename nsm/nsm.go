package nsm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/ghodss/yaml"
	"github.com/layer5io/meshery-nsm/meshes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/helm/pkg/chartutil"
)

// MeshName just returns the name of the mesh the client is representing
func (nsmClient *Client) MeshName(context.Context, *meshes.MeshNameRequest) (*meshes.MeshNameResponse, error) {
	return &meshes.MeshNameResponse{Name: "Network Service Mesh"}, nil
}

func (nsmClient *Client) createNamespace(ctx context.Context, namespace string) error {
	logrus.Infof("creating namespace: %s", namespace)
	yamlFileContents, err := nsmClient.executeTemplate(ctx, "", namespace, "namespace.yml")
	if err != nil {
		return err
	}
	if err := nsmClient.applyConfigChange(ctx, yamlFileContents, namespace, false, false); err != nil {
		return err
	}

	return nil
}
func (nsmClient *Client) applyConfigChange(ctx context.Context, yamlFileContents, namespace string, delete, isCustomOp bool) error {
	yamls, err := nsmClient.splitYAML(yamlFileContents)
	if err != nil {
		err = errors.Wrap(err, "error while splitting yaml")
		logrus.Error(err)
		return err
	}
	for _, yml := range yamls {
		if strings.TrimSpace(yml) != "" {
			if err := nsmClient.applyRulePayload(ctx, namespace, []byte(yml), delete, isCustomOp); err != nil {
				errStr := strings.TrimSpace(err.Error())
				if delete {
					if strings.HasSuffix(errStr, "not found") ||
						strings.HasSuffix(errStr, "the server could not find the requested resource") {
						// logrus.Debugf("skipping error. . .")
						continue
					}
				} else {
					if strings.HasSuffix(errStr, "already exists") {
						continue
					}
				}
				// logrus.Debugf("returning error: %v", err)
				return err
			}
		}
	}
	return nil
}

func (nsmClient *Client) splitYAML(yamlContents string) ([]string, error) {
	yamlDecoder, ok := NewDocumentDecoder(ioutil.NopCloser(bytes.NewReader([]byte(yamlContents)))).(*YAMLDecoder)
	if !ok {
		err := fmt.Errorf("unable to create a yaml decoder")
		logrus.Error(err)
		return nil, err
	}
	defer yamlDecoder.Close()
	var err error
	n := 0
	data := [][]byte{}
	ind := 0
	for err == io.ErrShortBuffer || err == nil {
		// for {
		d := make([]byte, 1000)
		n, err = yamlDecoder.Read(d)
		// logrus.Debugf("Read this: %s, count: %d, err: %v", d, n, err)
		if len(data) == 0 || len(data) <= ind {
			data = append(data, []byte{})
		}
		if n > 0 {
			data[ind] = append(data[ind], d...)
		}
		if err == nil {
			logrus.Debugf("..............BOUNDARY................")
			ind++
		}
	}
	result := make([]string, len(data))
	for i, row := range data {
		r := string(row)
		r = strings.Trim(r, "\x00")
		logrus.Debugf("ind: %d, data: %s", i, r)
		result[i] = r
	}
	return result, nil
}

func (nsmClient *Client) applyRulePayload(ctx context.Context, namespace string, newBytes []byte, delete, isCustomOp bool) error {
	if nsmClient.k8sDynamicClient == nil {
		return errors.New("mesh client has not been created")
	}
	// logrus.Debugf("received yaml bytes: %s", newBytes)
	jsonBytes, err := yaml.YAMLToJSON(newBytes)
	if err != nil {
		err = errors.Wrapf(err, "unable to convert yaml to json")
		logrus.Error(err)
		return err
	}
	// logrus.Debugf("created json: %s, length: %d", jsonBytes, len(jsonBytes))
	if len(jsonBytes) > 5 { // attempting to skip 'null' json
		data := &unstructured.Unstructured{}
		err = data.UnmarshalJSON(jsonBytes)
		if err != nil {
			err = errors.Wrapf(err, "unable to unmarshal json created from yaml")
			logrus.Error(err)
			return err
		}
		if data.IsList() {
			err = data.EachListItem(func(r runtime.Object) error {
				dataL, _ := r.(*unstructured.Unstructured)
				return nsmClient.executeRule(ctx, dataL, namespace, delete, isCustomOp)
			})
			return err
		}
		return nsmClient.executeRule(ctx, data, namespace, delete, isCustomOp)
	}
	return nil
}

func (nsmClient *Client) executeRule(ctx context.Context, data *unstructured.Unstructured, namespace string, delete, isCustomOp bool) error {
	// logrus.Debug("========================================================")
	// logrus.Debugf("Received data: %+#v", data)
	if namespace != "" {
		data.SetNamespace(namespace)
	}
	groupVersion := strings.Split(data.GetAPIVersion(), "/")
	logrus.Debugf("groupVersion: %v", groupVersion)
	var group, version string
	if len(groupVersion) == 2 {
		group = groupVersion[0]
		version = groupVersion[1]
	} else if len(groupVersion) == 1 {
		version = groupVersion[0]
	}

	kind := strings.ToLower(data.GetKind())
	switch kind {
	case "logentry":
		kind = "logentries"
	case "kubernetes":
		kind = "kuberneteses"
	case "podsecuritypolicy":
		kind = "podsecuritypolicies"
	case "serviceentry":
		kind = "serviceentries"
	default:
		kind += "s"
	}

	res := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: kind,
	}
	logrus.Debugf("Computed Resource: %+#v", res)

	if delete {
		return nsmClient.deleteResource(ctx, res, data)
	}
	trackRetry := 0
RETRY:
	if err := nsmClient.createResource(ctx, res, data); err != nil {
		if isCustomOp {
			if err := nsmClient.deleteResource(ctx, res, data); err != nil {
				return err
			}
			time.Sleep(time.Second)
			if err := nsmClient.createResource(ctx, res, data); err != nil {
				return err
			}
			// data1, err := iClient.getResource(ctx, res, data)
			// if err != nil {
			// 	return err
			// }
			// if err = iClient.updateResource(ctx, res, data1); err != nil {
			// 	return err
			// }
		} else {
			data1, err := nsmClient.getResource(ctx, res, data)
			if err != nil {
				return err
			}
			data.SetCreationTimestamp(data1.GetCreationTimestamp())
			data.SetGenerateName(data1.GetGenerateName())
			data.SetGeneration(data1.GetGeneration())
			data.SetSelfLink(data1.GetSelfLink())
			data.SetResourceVersion(data1.GetResourceVersion())
			// data.DeepCopyInto(data1)
			if err = nsmClient.updateResource(ctx, res, data); err != nil {
				if strings.Contains(err.Error(), "the server does not allow this method on the requested resource") {
					logrus.Info("attempting to delete resource. . . ")
					nsmClient.deleteResource(ctx, res, data)
					trackRetry++
					if trackRetry <= 3 {
						goto RETRY
					} // else return error
				}
				return err
			}
			// return err
		}
	}
	return nil
}

func (nsmClient *Client) createResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) error {
	_, err := nsmClient.k8sDynamicClient.Resource(res).Namespace(data.GetNamespace()).Create(data, metav1.CreateOptions{})
	if err != nil {
		err = errors.Wrapf(err, "unable to create the requested resource, attempting operation without namespace")
		logrus.Warn(err)
		_, err = nsmClient.k8sDynamicClient.Resource(res).Create(data, metav1.CreateOptions{})
		if err != nil {
			err = errors.Wrapf(err, "unable to create the requested resource, attempting to update")
			logrus.Error(err)
			return err
		}
	}

	logrus.Infof("Created Resource of type: %s and name: %s", data.GetKind(), data.GetName())
	return nil
}

func (nsmClient *Client) deleteResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) error {
	if nsmClient.k8sDynamicClient == nil {
		return errors.New("mesh client has not been created")
	}

	if res.Resource == "namespaces" && data.GetName() == "default" { // skipping deletion of default namespace
		return nil
	}

	// in the case with deployments, have to scale it down to 0 first and then delete. . . or else RS and pods will be left behind
	if res.Resource == "deployments" {
		data1, err := nsmClient.getResource(ctx, res, data)
		if err != nil {
			return err
		}
		depl := data1.UnstructuredContent()
		spec1 := depl["spec"].(map[string]interface{})
		spec1["replicas"] = 0
		data1.SetUnstructuredContent(depl)
		if err = nsmClient.updateResource(ctx, res, data1); err != nil {
			return err
		}
	}

	err := nsmClient.k8sDynamicClient.Resource(res).Namespace(data.GetNamespace()).Delete(data.GetName(), &metav1.DeleteOptions{})
	if err != nil {
		err = errors.Wrapf(err, "unable to delete the requested resource, attempting operation without namespace")
		logrus.Warn(err)

		err := nsmClient.k8sDynamicClient.Resource(res).Delete(data.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			err = errors.Wrapf(err, "unable to delete the requested resource")
			logrus.Error(err)
			return err
		}
	}
	logrus.Infof("Deleted Resource of type: %s and name: %s", data.GetKind(), data.GetName())
	return nil
}

func (nsmClient *Client) getResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	data1, err := nsmClient.k8sDynamicClient.Resource(res).Namespace(data.GetNamespace()).Get(data.GetName(), metav1.GetOptions{})
	if err != nil {
		err = errors.Wrap(err, "unable to retrieve the resource with a matching name, attempting operation without namespace")
		logrus.Warn(err)

		data1, err = nsmClient.k8sDynamicClient.Resource(res).Get(data.GetName(), metav1.GetOptions{})
		if err != nil {
			err = errors.Wrap(err, "unable to retrieve the resource with a matching name, while attempting to apply the config")
			logrus.Error(err)
			return nil, err
		}
	}
	logrus.Infof("Retrieved Resource of type: %s and name: %s", data.GetKind(), data.GetName())
	return data1, nil
}

func (nsmClient *Client) updateResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) error {
	if _, err := nsmClient.k8sDynamicClient.Resource(res).Namespace(data.GetNamespace()).Update(data, metav1.UpdateOptions{}); err != nil {
		err = errors.Wrap(err, "unable to update resource with the given name, attempting operation without namespace")
		logrus.Warn(err)

		if _, err = nsmClient.k8sDynamicClient.Resource(res).Update(data, metav1.UpdateOptions{}); err != nil {
			err = errors.Wrap(err, "unable to update resource with the given name, while attempting to apply the config")
			logrus.Error(err)
			return err
		}
	}
	logrus.Infof("Updated Resource of type: %s and name: %s", data.GetKind(), data.GetName())
	return nil
}

// ApplyOperation is a method invoked to apply a particular operation on the mesh in a namespace
func (nsmClient *Client) ApplyOperation(ctx context.Context, arReq *meshes.ApplyRuleRequest) (*meshes.ApplyRuleResponse, error) {
	if arReq == nil {
		return nil, errors.New("mesh client has not been created")
	}

	op, ok := supportedOps[arReq.OpName]
	if !ok {
		return nil, fmt.Errorf("operation id: %s, error: %s is not a valid operation name", arReq.OperationId, arReq.OpName)
	}
	if arReq.OpName == customOpCommand && arReq.CustomBody == "" {
		return nil, fmt.Errorf("operation id: %s, error: yaml body is empty for %s operation", arReq.OperationId, arReq.OpName)
	}

	var yamlFileContents string
	var err error
	isCustomOp := false
	var nsmFolderName, appName, svcName string

	if !arReq.DeleteOp {
		nsmClient.createNamespace(ctx, arReq.Namespace)
	}
	switch arReq.OpName {
	case customOpCommand:
		yamlFileContents = arReq.CustomBody
		isCustomOp = true
	case installICMPCommand:
		if nsmFolderName == "" {
			nsmFolderName = "icmp-responder"
			appName = "ICMP Application"
		}
		fallthrough
	case installVPNCommand:
		if nsmFolderName == "" {
			nsmFolderName = "vpn"
			appName = "VPN Application"
		}
		fallthrough
	case installVPNICMPCommand:
		if nsmFolderName == "" {
			nsmFolderName = "vpp-icmp-responder"
			appName = "VPN ICMP Application"
		}
		fallthrough
	case installNSMCommand:
		go func() {
			opName1 := "deploying"
			if arReq.DeleteOp {
				opName1 = "removing"
			}
			customConfig := ""
			if nsmFolderName == "" {
				nsmFolderName = "nsm"
				appName = "NSM"
				data, err := ioutil.ReadFile(path.Join("nsm", "config_templates/values.yaml"))
				if err != nil {
					err = errors.Wrapf(err, "unable to find the values.yml file")
					logrus.Error(err)
					nsmClient.eventChan <- &meshes.EventsResponse{
						OperationId: arReq.OperationId,
						EventType:   meshes.EventType_ERROR,
						Summary:     fmt.Sprintf("Error while %s %s", opName1, appName),
						Details:     err.Error(),
					}
					return
				}
				customConfig = string(data)
				logrus.Infof("the loaded file %s", customConfig)
			}
			if err = nsmClient.downloadNSM(); err != nil {
				nsmClient.eventChan <- &meshes.EventsResponse{
					OperationId: arReq.OperationId,
					EventType:   meshes.EventType_ERROR,
					Summary:     fmt.Sprintf("Error while %s %s", opName1, appName),
					Details:     err.Error(),
				}
				return
			}
			if err := nsmClient.executeNSMHelmInstall(ctx, arReq, customConfig, nsmFolderName); err != nil {
				nsmClient.eventChan <- &meshes.EventsResponse{
					OperationId: arReq.OperationId,
					EventType:   meshes.EventType_ERROR,
					Summary:     fmt.Sprintf("Error while %s %s", opName1, appName),
					Details:     err.Error(),
				}
				return
			}
			opName := "deployed"
			if arReq.DeleteOp {
				opName = "removed"
			}

			nsmClient.eventChan <- &meshes.EventsResponse{
				OperationId: arReq.OperationId,
				EventType:   meshes.EventType_INFO,
				Summary:     fmt.Sprintf("%s %s successfully", appName, opName),
				Details:     fmt.Sprintf("%s %s successfully", appName, opName),
			}

			return
		}()

		return &meshes.ApplyRuleResponse{
			OperationId: arReq.OperationId,
		}, nil
	case installHelloNSMApp:
		svcName = "appa"
		appName = "Hello NSM application"
		fallthrough
	default:
		yamlFileContents, err = nsmClient.executeTemplate(ctx, arReq.Username, arReq.Namespace, op.templateName)
		if err != nil {
			return nil, err
		}
	}

	go func() {
		logrus.Debug("in the routine. . . .")
		opName1 := "deploying"
		if arReq.DeleteOp {
			opName1 = "removing"
		}
		if err := nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp, isCustomOp); err != nil {
			nsmClient.eventChan <- &meshes.EventsResponse{
				OperationId: arReq.OperationId,
				EventType:   meshes.EventType_ERROR,
				Summary:     fmt.Sprintf("Error while %s \"%s\"", opName1, op.name),
				Details:     err.Error(),
			}
			return
		}
		opName := "deployed"
		if arReq.DeleteOp {
			opName = "removed"
		}
		detailedMsg := fmt.Sprintf("\"%s\" %s successfully", op.name, opName)
		if svcName != "" && !arReq.DeleteOp {
			ports, err := nsmClient.getSVCPort(ctx, svcName, arReq.Namespace)
			if err != nil {
				nsmClient.eventChan <- &meshes.EventsResponse{
					OperationId: arReq.OperationId,
					EventType:   meshes.EventType_WARN,
					Summary:     fmt.Sprintf("%s is deployed but unable to retrieve the port info for the service at the moment", appName),
					Details:     err.Error(),
				}
				return
			}
			var portMsg string
			if len(ports) == 1 {
				portMsg = fmt.Sprintf("The service is possibly available on port: %v", ports)
			} else if len(ports) > 1 {
				portMsg = fmt.Sprintf("The service is possibly available on one of the following ports: %v", ports)
			}
			detailedMsg = fmt.Sprintf("%s is now %s. %s", appName, opName, portMsg)
		}
		logrus.Debugf("details msg: %s", detailedMsg)
		nsmClient.eventChan <- &meshes.EventsResponse{
			OperationId: arReq.OperationId,
			EventType:   meshes.EventType_INFO,
			Summary:     fmt.Sprintf("\"%s\" %s successfully", op.name, opName),
			Details:     detailedMsg,
		}
	}()

	return &meshes.ApplyRuleResponse{
		OperationId: arReq.OperationId,
	}, nil
}

// installs NSM using helm
func (nsmClient *Client) executeNSMInstall(ctx context.Context, arReq *meshes.ApplyRuleRequest) error {
	data, err := ioutil.ReadFile(path.Join("nsm", "config_templates/values.yaml"))
	if err != nil {
		err = errors.Wrapf(err, "unable to find the values.yml file")
		logrus.Error(err)
		return err
	}
	logrus.Infof("the loaded file %s", data)

	return nsmClient.executeNSMHelmInstall(ctx, arReq, string(data), "nsm")
}

// installs any helm stuff as part of the NSM git repo
func (nsmClient *Client) executeNSMHelmInstall(ctx context.Context, arReq *meshes.ApplyRuleRequest, customValues, folderName string) error {
	logrus.Debugf("destination folder: %s", destinationFolder)
	chart, err := chartutil.Load(path.Join(destinationFolder, "deployments", "helm", folderName))
	if err != nil {
		err = errors.Wrapf(err, "Chart shows error")
		logrus.Error(err)
		return err
	}

	manifests, err := renderManifests(ctx, chart, "nsm", arReq.Namespace, "", customValues)
	if err != nil {
		err = errors.Wrapf(err, "render manifests error")
		logrus.Error(err)
		return err
	}

	for _, element := range manifests {
		if err = nsmClient.applyConfigChange(ctx, element.Content, arReq.Namespace, arReq.DeleteOp, false); err != nil {
			return err
		}
	}
	return nil
}

func (nsmClient *Client) executeTemplate(ctx context.Context, username, namespace, templateName string) (string, error) {
	tmpl, err := template.ParseFiles(path.Join("nsm", "config_templates", templateName))
	if err != nil {
		err = errors.Wrapf(err, "unable to parse template")
		logrus.Error(err)
		return "", err
	}
	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, map[string]string{
		"user_name": username,
		"namespace": namespace,
	})
	if err != nil {
		err = errors.Wrapf(err, "unable to execute template")
		logrus.Error(err)
		return "", err
	}
	return buf.String(), nil
}

//CreateMeshInstance is called from UI
func (nsmClient *Client) CreateMeshInstance(_ context.Context, k8sReq *meshes.CreateMeshInstanceRequest) (*meshes.CreateMeshInstanceResponse, error) {
	var k8sConfig []byte
	contextName := ""
	if k8sReq != nil {
		k8sConfig = k8sReq.K8SConfig
		contextName = k8sReq.ContextName
	}
	logrus.Debugf("received contextName: %s", contextName)

	ic, err := newClient(k8sConfig, contextName)
	if err != nil {
		err = errors.Wrapf(err, "unable to create a new NSM client")
		logrus.Error(err)
		return nil, err
	}
	nsmClient.k8sClientset = ic.k8sClientset
	nsmClient.k8sDynamicClient = ic.k8sDynamicClient
	nsmClient.eventChan = make(chan *meshes.EventsResponse, 100)
	nsmClient.config = ic.config
	return &meshes.CreateMeshInstanceResponse{}, nil
}

// StreamEvents - streams generated/collected events to the client
func (nsmClient *Client) StreamEvents(in *meshes.EventsRequest, stream meshes.MeshService_StreamEventsServer) error {
	logrus.Debugf("waiting on event stream. . .")
	for {
		select {
		case event := <-nsmClient.eventChan:
			logrus.Debugf("sending event: %+#v", event)
			if err := stream.Send(event); err != nil {
				err = errors.Wrapf(err, "unable to send event")

				// to prevent loosing the event, will re-add to the channel
				go func() {
					nsmClient.eventChan <- event
				}()
				logrus.Error(err)
				return err
			}
		default:
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// SupportedOperations - returns a list of supported operations on the mesh
func (nsmClient *Client) SupportedOperations(context.Context, *meshes.SupportedOperationsRequest) (*meshes.SupportedOperationsResponse, error) {
	supportedOpsCount := len(supportedOps)
	result := make([]*meshes.SupportedOperation, supportedOpsCount)
	i := 0
	for k, sp := range supportedOps {
		result[i] = &meshes.SupportedOperation{
			Key:      k,
			Value:    sp.name,
			Category: sp.opType,
		}
		i++
	}

	return &meshes.SupportedOperationsResponse{
		Ops: result,
	}, nil
}

func (iClient *Client) getSVCPort(ctx context.Context, svc, namespace string) ([]int64, error) {
	ns := &unstructured.Unstructured{}
	res := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "services",
	}
	ns.SetName(svc)
	ns.SetNamespace(namespace)
	ns, err := iClient.getResource(ctx, res, ns)
	if err != nil {
		err = errors.Wrapf(err, "unable to get service details")
		logrus.Error(err)
		return nil, err
	}
	svcInst := ns.UnstructuredContent()
	spec := svcInst["spec"].(map[string]interface{})
	ports, _ := spec["ports"].([]interface{})
	nodePorts := []int64{}
	for _, port := range ports {
		p, _ := port.(map[string]interface{})
		np, ok := p["nodePort"]
		if ok {
			npi, _ := np.(int64)
			nodePorts = append(nodePorts, npi)
		}
	}
	logrus.Debugf("retrieved svc: %+#v", ns)
	return nodePorts, nil
}
