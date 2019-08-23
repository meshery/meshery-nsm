package nsm

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/ghodss/yaml"
	"github.com/layer5io/meshery-nsm/meshes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/cert"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
)

func (nsmClient *NSMClient) MeshName(context.Context, *meshes.MeshNameRequest) (*meshes.MeshNameResponse, error) {
	return &meshes.MeshNameResponse{Name: "Network Service Mesh"}, nil
}

func (nsmClient *NSMClient) createNamespace(ctx context.Context, namespace string) error {
	logrus.Infof("creating namespace: %s", namespace)
	yamlFileContents, err := nsmClient.executeTemplate(ctx, "", namespace, "namespace.yml")
	if err != nil {
		return err
	}
	if err := nsmClient.applyConfigChange(ctx, yamlFileContents, namespace, false); err != nil {
		return err
	}

	return nil
}
func (nsmClient *NSMClient) applyConfigChange(ctx context.Context, yamlFileContents, namespace string, delete bool) error {
	// yamls := strings.Split(yamlFileContents, "---")
	yamls, err := nsmClient.splitYAML(yamlFileContents)
	if err != nil {
		err = errors.Wrap(err, "error while splitting yaml")
		logrus.Error(err)
		return err
	}

	for _, yml := range yamls {
		if strings.TrimSpace(yml) != "" {
			if err := nsmClient.applyRulePayload(ctx, namespace, []byte(yml), delete); err != nil {
				errStr := strings.TrimSpace(err.Error())
				if delete && (strings.HasSuffix(errStr, "not found") ||
					strings.HasSuffix(errStr, "the server could not find the requested resource")) {
					// logrus.Debugf("skipping error. . .")
					continue
				}
				// logrus.Debugf("returning error: %v", err)
				return err
			}
		}
	}
	return nil
}
func (nsmClient *NSMClient) splitYAML(yamlContents string) ([]string, error) {
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

func (nsmClient *NSMClient) applyRulePayload(ctx context.Context, namespace string, newBytes []byte, delete bool) error {
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
				return nsmClient.executeRule(ctx, dataL, namespace, delete)
			})
			return err
		}
		return nsmClient.executeRule(ctx, data, namespace, delete)
	}
	return nil
}

func (nsmClient *NSMClient) executeRule(ctx context.Context, data *unstructured.Unstructured, namespace string, delete bool) error {
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

	if err := nsmClient.createResource(ctx, res, data); err != nil {
		data1, err := nsmClient.getResource(ctx, res, data)
		if err != nil {
			return err
		}
		if err = nsmClient.updateResource(ctx, res, data1); err != nil {
			return err
		}
	}
	return nil
}

func (nsmClient *NSMClient) createResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) error {
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

func (nsmClient *NSMClient) deleteResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) error {
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

func (nsmClient *NSMClient) getResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) (*unstructured.Unstructured, error) {
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

func (nsmClient *NSMClient) updateResource(ctx context.Context, res schema.GroupVersionResource, data *unstructured.Unstructured) error {
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

func (nsmClient *NSMClient) ApplyOperation(ctx context.Context, arReq *meshes.ApplyRuleRequest) (*meshes.ApplyRuleResponse, error) {
	if arReq == nil {
		return nil, errors.New("mesh client has not been created")
	}

	op, ok := supportedOps[arReq.OpName]
	if !ok {
		return nil, fmt.Errorf("error: %s is not a valid operation name", arReq.OpName)
	}

	if arReq.OpName == "custom" && arReq.CustomBody == "" {
		return nil, fmt.Errorf("error: yaml body is empty for %s operation", arReq.OpName)
	}

	var yamlFileContents string
	var err error
	installWithmTLS := false
	nsmClient.downloadNSM()

	logrus.Infof("The NSM folder is deleted")
	switch arReq.OpName {
	case customOpCommand:
		yamlFileContents = arReq.CustomBody
	case installNSMCommand:
		go func() {
			opName1 := "deploying"
			if arReq.DeleteOp {
				opName1 = "removing"
			} else {

				nsmClient.createNamespace(ctx, "nsm-system")
				nsmClient.DeployAdmissionWebhook(arReq.Namespace)
			}
			if err := nsmClient.executeInstall(ctx, installWithmTLS, arReq); err != nil {
				nsmClient.eventChan <- &meshes.EventsResponse{
					EventType: meshes.EventType_ERROR,
					Summary:   fmt.Sprintf("Error while %s NSM", opName1),
					Details:   err.Error(),
				}
				return
			}
			opName := "deployed"
			if arReq.DeleteOp {
				opName = "removed"
			}
			nsmClient.eventChan <- &meshes.EventsResponse{
				EventType: meshes.EventType_INFO,
				Summary:   fmt.Sprintf("NSM %s successfully", opName),
			}

			return
		}()

		return &meshes.ApplyRuleResponse{}, nil

	case installICMPCommand:
		go func() {
			opName1 := "deploying"
			if arReq.DeleteOp {
				opName1 = "removing"
			}
			if err := nsmClient.executeICMPInstall(ctx, arReq); err != nil {
				nsmClient.eventChan <- &meshes.EventsResponse{
					EventType: meshes.EventType_ERROR,
					Summary:   fmt.Sprintf("Error while %s the ICMP Application", opName1),
					Details:   err.Error(),
				}
				return
			}
			opName := "deployed"
			if arReq.DeleteOp {
				opName = "removed"
			}
			nsmClient.eventChan <- &meshes.EventsResponse{
				EventType: meshes.EventType_INFO,
				Summary:   fmt.Sprintf(" ICMP app %s successfully", opName),
				Details:   fmt.Sprintf("The ICMP app is now %s.", opName),
			}
			return
		}()

		return &meshes.ApplyRuleResponse{}, nil

	default:
		yamlFileContents, err = nsmClient.executeTemplate(ctx, arReq.Username, arReq.Namespace, op.templateName)
		if err != nil {
			return nil, err
		}
	}

	if err := nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp); err != nil {
		return nil, err
	}

	return &meshes.ApplyRuleResponse{}, nil
}

func (nsmClient *NSMClient) executeInstall(ctx context.Context, installmTLS bool, arReq *meshes.ApplyRuleRequest) error {

	var yamlFileContents string
	var err error
	/*args := []string{"-c", webhookcertificatefile}
	//cmd := exec.Command("/bin/sh", args...)
	/*out, err := exec.Command("/bin/sh", args...).Output()
	if err != nil {
		logrus.Errorf("Error while executing the shell script :", err)
	}
	cmd := exec.Command("/bin/sh", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	sterr := cmd.Run()
	if sterr != nil {
		logrus.Errorf("Error while executing the shell script :", sterr, stderr.String())

	}
	//fmt.Println("Result: " + out.String())

	logrus.Infof("Output for shell script : %s ", out.String())*/

	//logrus.Infof("WebhookCert file : %s", cmd.Stdout)
	yamlFileContents, err = nsmClient.getClusterRoleadminYAML()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getClusterBindingYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getClusterRoleViewYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getCrdNetworkserviceManagerYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getCrdNetworkserviceYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getCrdNetworkserviceEndpointYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getNsmgrYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getAdmissionWebhookYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getCrossconnectYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getVppagentdataplaneYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)
	yamlFileContents, err = nsmClient.getSkydiveYaml()
	nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp)

	if err != nil {
		return err
	}
	/*if err := nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp); err != nil {
		return err
	}*/

	return nil
}

func (nsmClient *NSMClient) executeTemplate(ctx context.Context, username, namespace, templateName string) (string, error) {
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
func (nsmClient *NSMClient) CreateMeshInstance(_ context.Context, k8sReq *meshes.CreateMeshInstanceRequest) (*meshes.CreateMeshInstanceResponse, error) {
	var k8sConfig []byte
	contextName := ""
	if k8sReq != nil {
		k8sConfig = k8sReq.K8SConfig
		contextName = k8sReq.ContextName
	}
	// logrus.Debugf("received k8sConfig: %s", k8sConfig)
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
func (nsmClient *NSMClient) StreamEvents(in *meshes.EventsRequest, stream meshes.MeshService_StreamEventsServer) error {
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

//func (nsmClient *NSMClient) createsecrets(ctx context.Context, namespace string) {
/* executing script part */
/*generate csr.conf*/
/*	csr :=
		`[req]
	req_extensions = v3_req
	distinguished_name = req_distinguished_name
	[req_distinguished_name]
	[ v3_req ]
	basicConstraints = CA:FALSE
	keyUsage = nonRepudiation, digitalSignature, keyEncipherment
	extendedKeyUsage = serverAuth
	subjectAltName = @alt_names
	[alt_names]
	DNS.1 = nsm-admission-webhook-svc
	DNS.2 = nsm-admission-webhook-svc.` + namespace + `
	DNS.3 = nsm-admission-webhook-svc.` + namespace + `.svc`

	f, err := os.Create("/tmp/csr.conf")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = f.WriteString(csr)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}
	/*execute openssh server command to generate server.pem in tmp directory */
/*os.Create("/tmp/server-key.pem")
os.Create("/tmp/server.csr")
opensslcommand := `openssl genrsa -out /tmp/server-key.pem 2048 ; openssl req -new -key /tmp/server-key.pem -subj "//CN=nsm-admission-webhook-svc.` + namespace + `.svc/" -out /tmp/server.csr -config /tmp/csr.conf`

cmd := exec.Command("/bin/bash", "-c", opensslcommand)
var out bytes.Buffer
var stderr bytes.Buffer
cmd.Stdout = &out
cmd.Stderr = &stderr
sterr := cmd.Run()
if sterr != nil {
	logrus.Errorf("Error while creating secret key :", sterr, stderr.String())

}
logrus.Infof("Result: " + out.String())

/* storing secret key in a variable*/
/*data, err := ioutil.ReadFile("/tmp/server.csr")
if err != nil {
	fmt.Println("File reading error", err)
	return
}
fmt.Println("Contents of file:", string(data))*/
/*key := exec.Command("/bin/bash", "-c", "base64 < /tmp/server.csr | tr -d '\n'")
	key.Stdout = &out
	key.Stderr = &stderr
	sterr = key.Run()
	if sterr != nil {
		logrus.Errorf("Error while reading base64 of secret key :", sterr, stderr.String())

	}
	logrus.Infof("base64 : " + out.String())

	yamlFileContents := `apiVersion: certificates.k8s.io/v1beta1
   kind: CertificateSigningRequest
   metadata:
     name:nsm-admission-webhook-svc.nsm-system
   spec:
     groups:
       - system:authenticated
         request: ` + out.String() + `usages:
- digital signature
- key encipherment
- server auth`
	file, er := os.Create("/tmp/NetworkServiceMesh/CSR.yaml")
	if er != nil {
		fmt.Println(er)
		return
	}
	_, err = file.WriteString(yamlFileContents)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}
	yamlcontent, _ := nsmClient.getComponentYAML("/tmp/NetworkServiceMesh/CSR.yaml")
	nsmClient.applyConfigChange(ctx, yamlcontent, namespace, false)*/
/*created the certificate signing request */
/* approving the certificate and creating secret using kubectl */

//}
// DeployAdmissionWebhook - Setup Admission Webhook
func (nsmClient *NSMClient) DeployAdmissionWebhook(namespace string) (*arv1beta1.MutatingWebhookConfiguration, *v1.Service) {
	name := "nsm-admission-webhook"
	_, caCert := CreateAdmissionWebhookSecret(nsmClient, name, namespace)
	awc := CreateMutatingWebhookConfiguration(nsmClient, caCert, name, namespace)
	//	awDeployment := CreateAdmissionWebhookDeployment(nsmClient, name, image, namespace)
	awService := CreateAdmissionWebhookService(nsmClient, name, namespace)
	return awc, awService
}
func CreateAdmissionWebhookSecret(nsmClient *NSMClient, name, namespace string) (*v1.Secret, []byte) {

	caCertSpec := &cert.Config{
		CommonName: "admission-controller-ca",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	caCert, caKey, err := pkiutil.NewCertificateAuthority(caCertSpec)
	if err != nil {
		logrus.Infof("Not able to create the admission webhook secret 1")
	}

	certSpec := &cert.Config{
		CommonName: name + "-svc",
		AltNames: cert.AltNames{
			DNSNames: []string{
				name + "-svc." + namespace,
				name + "-svc." + namespace + ".svc",
			},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	cer, key, err := pkiutil.NewCertAndKey(caCert, caKey, certSpec)
	if err != nil {
		logrus.Infof("Not able to create the ca cert and key  ")
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	keyPem := pem.EncodeToMemory(block)

	block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cer.Raw,
	}
	certPem := pem.EncodeToMemory(block)

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-certs",
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key.pem":  keyPem,
			"cert.pem": certPem,
		},
	}
	awSecret, err := nsmClient.CreateSecret(secret, namespace)
	if err != nil {
		logrus.Infof("Not able to create the admission webhook secret ")
	}

	block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	}
	caCertPem := pem.EncodeToMemory(block)

	return awSecret, caCertPem
}
func CreateMutatingWebhookConfiguration(nsmClient *NSMClient, certPem []byte, name, namespace string) *arv1beta1.MutatingWebhookConfiguration {
	servicePath := "/mutate"

	mutatingWebhookConf := &arv1beta1.MutatingWebhookConfiguration{

		TypeMeta: metav1.TypeMeta{
			Kind: "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-cfg",
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		Webhooks: []arv1beta1.Webhook{
			{
				Name: "admission-webhook.networkservicemesh.io",
				ClientConfig: arv1beta1.WebhookClientConfig{
					Service: &arv1beta1.ServiceReference{
						Namespace: namespace,
						Name:      name + "-svc",
						Path:      &servicePath,
					},
					CABundle: certPem,
				},
				Rules: []arv1beta1.RuleWithOperations{
					{
						Operations: []arv1beta1.OperationType{
							arv1beta1.Create,
						},
						Rule: arv1beta1.Rule{
							APIGroups:   []string{"apps", "extensions", ""},
							APIVersions: []string{"v1", "v1beta1"},
							Resources:   []string{"deployments", "services", "pods"},
						},
					},
				},
			},
		},
	}
	awc, err := nsmClient.CreateMutatingWebhookConfiguration(mutatingWebhookConf)
	if err != nil {
		logrus.Infof("Not able to create the admission webhook mutating configuration  ")
	}

	return awc
}
func (nsmClient *NSMClient) CreateMutatingWebhookConfiguration(mutatingWebhookConf *arv1beta1.MutatingWebhookConfiguration) (*arv1beta1.MutatingWebhookConfiguration, error) {
	awc, err := nsmClient.k8sClientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(mutatingWebhookConf)
	if err != nil {
		logrus.Errorf("Error creating MutatingWebhookConfiguration: %v %v", awc, err)
	}
	logrus.Infof("MutatingWebhookConfiguration is created: %v", awc)
	return awc, err
}
func CreateAdmissionWebhookService(nsmClient *NSMClient, name, namespace string) *v1.Service {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-svc",
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			},
			Selector: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
	}
	awService, err := nsmClient.CreateService(service, namespace)
	if err != nil {
		logrus.Infof("Not able to create the admission webhook service ")
	}

	return awService
}
func (nsmClient *NSMClient) CreateService(service *v1.Service, namespace string) (*v1.Service, error) {
	_ = nsmClient.k8sClientset.CoreV1().Services(namespace).Delete(service.Name, &metav1.DeleteOptions{})
	s, err := nsmClient.k8sClientset.CoreV1().Services(namespace).Create(service)
	if err != nil {
		logrus.Errorf("Error creating service: %v %v", s, err)
	}

	return s, err
}

func (nsmClient *NSMClient) CreateSecret(secret *v1.Secret, namespace string) (*v1.Secret, error) {
	s, err := nsmClient.k8sClientset.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		logrus.Errorf("Error creating secret: %v %v", s, err)
	}

	return s, err
}

func (nsmClient *NSMClient) executeICMPInstall(ctx context.Context, arReq *meshes.ApplyRuleRequest) error {

	yamlFileContents, err := nsmClient.getICMPAppYaml()
	if err != nil {
		return err
	}
	if err := nsmClient.applyConfigChange(ctx, yamlFileContents, arReq.Namespace, arReq.DeleteOp); err != nil {
		return err
	}
	return nil
}

func (nsmClient *NSMClient) SupportedOperations(context.Context, *meshes.SupportedOperationsRequest) (*meshes.SupportedOperationsResponse, error) {
	result := map[string]string{}
	for key, op := range supportedOps {
		result[key] = op.name
	}
	return &meshes.SupportedOperationsResponse{
		Ops: result,
	}, nil
}

func RemoveDir(directory string) error {
	d, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(directory, name))
		if err != nil {
			return err
		}
	}
	return nil
}
