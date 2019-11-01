package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func main() {
	serviceURL := os.Getenv("URL")
	if serviceURL == "" {
		logrus.Fatal("please provide the service URL in an environment variable: URL")
	}
	serviceURLInst, _ := url.Parse(serviceURL)
	serviceName := serviceURLInst.Hostname()

	if serviceName == "" {
		logrus.Fatal("please provide a valid service URL in an environment variable: URL")
	}

	ipCIDR := os.Getenv("IP_ADDRESS")
	if ipCIDR == "" {
		logrus.Fatal("please provide the NSM IP CIDR in an environment variable: IP_ADDRESS")
	}

	_, ipNet, err := net.ParseCIDR(ipCIDR)
	if err != nil {
		err = errors.Wrap(err, "unable to parse IP_ADDRESS CIDR")
		logrus.Fatal(err)
	}

	retryCount := 0
	totalRetryCount := 10
	sleepTime := time.Second * 10

RETRY:
	ips, err := fetchNSMIPAddresses(serviceName, ipNet)
	if err != nil {
		logrus.Error(err)
		if retryCount < totalRetryCount {
			logrus.Infof("retrying in 10 sec: %v", sleepTime)
			time.Sleep(sleepTime)
			retryCount++
			goto RETRY
		} else {
			logrus.Fatal("unable to find a NSM IP address even after a few retries. . . bailing out")
			return
		}
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("received a request")

		rand.Seed(time.Now().UnixNano())
		serviceURLInst.Host = ips[rand.Intn(len(ips))]
		logrus.Infof("computed service url: %s", serviceURLInst.String())
		hc := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest(http.MethodGet, serviceURLInst.String(), nil)
		resp, err := hc.Do(req)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError)+" 1", http.StatusInternalServerError)
			logrus.Errorf("there was an error 1: %v\n", err)
			return
		}
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError)+" 2", http.StatusInternalServerError)
			logrus.Errorf("there was an error 2: %v\n", err)
			return
		}
		w.Write([]byte(fmt.Sprintf("Received data: %s \nFrom NSM IP: %s", data, serviceURLInst.Hostname())))
		logrus.Info("request processed successfully")
	})

	http.ListenAndServe(":80", nil)
}

func getK8SClientSet() (*kubernetes.Clientset, *rest.Config, error) {
	var clientConfig *rest.Config
	var err error
	clientConfig, err = rest.InClusterConfig()
	if err != nil {
		err = errors.Wrap(err, "unable to load in-cluster kubeconfig")
		logrus.Error(err)
		return nil, nil, err
	}

	clientConfig.Timeout = 2 * time.Second
	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		err = errors.Wrap(err, "unable to create client set")
		logrus.Error(err)
		return nil, nil, err
	}
	return clientset, clientConfig, nil
}

func fetchNSMIPAddresses(serviceName string, ipNet *net.IPNet) ([]string, error) {
	clientset, config, err := getK8SClientSet()
	if err != nil {
		return nil, err
	}

	// get the pods for the service
	namespacelist, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		err = errors.Wrap(err, "unable to get the list of namespaces")
		logrus.Error(err)
		return nil, err
	}

	var svc *v1.Service
	for _, ns := range namespacelist.Items {
		logrus.Debugf("Listing services in namespace %q", ns.GetName())

		svcClient := clientset.CoreV1().Services(ns.GetName())
		svc, err = svcClient.Get(serviceName, metav1.GetOptions{})
		if err != nil {
			err = errors.Wrapf(err, "unable to get service %s in the %s namespace", serviceName, ns.GetName())
			logrus.Error(err)
			continue
		}
	}
	if svc == nil {
		return nil, fmt.Errorf("unable to find the service")
	}
	// now that we have found the svc, lets try to fetch the pod names

	pods, err := clientset.CoreV1().Pods(svc.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
	})
	if err != nil {
		err = errors.Wrap(err, "unable to get the list of pods for the labels from svc")
		logrus.Error(err)
		return nil, err
	}
	cmd := []string{"sh", "-c", "ip addr | grep inet | awk '{print $2}' | cut -d / -f 1"}
	var nsmIPs []string
	for _, pod := range pods.Items {
		for _, cont := range pod.Spec.Containers {
			nsmIP, err := execIntoPodToFindNSMIPAddr(clientset, config, ipNet, pod.Name, pod.Namespace, cont.Name, cmd)
			if err != nil {
				logrus.Error(err)
				continue
			}
			// we are just focusing on the first port of the chosen container
			if len(cont.Ports) > 0 {
				nsmIP = fmt.Sprintf("%s:%d", nsmIP, cont.Ports[0].ContainerPort)
			}
			logrus.Infof("nsm ip: %s", nsmIP)
			nsmIPs = append(nsmIPs, nsmIP)
		}
	}
	if len(nsmIPs) > 0 {
		return nsmIPs, nil
	}
	return nil, fmt.Errorf("unable to find any NSM ips")
}

// cmd: ip addr | grep inet | awk '{print $2}' | cut -d / -f 1
func execIntoPodToFindNSMIPAddr(clientset *kubernetes.Clientset, config *rest.Config, ipNet *net.IPNet, podName,
	namespace, container string, cmd []string) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec").Param("container", container)
	option := &v1.PodExecOptions{
		Command: cmd,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}
	req = req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}
	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", err
	}
	ips := strings.Split(stdout.String(), "\r\n")
	for _, ipaddrStr := range ips {
		ipaddr := net.ParseIP(ipaddrStr)
		if ipaddr == nil {
			logrus.Warnf("could not parse IP string: %s", ipaddrStr)
			continue
		}
		if ipNet.Contains(ipaddr) {
			logrus.Infof("found a NSM address: %s", ipaddrStr)
			return ipaddrStr, nil
		}
	}
	return "", fmt.Errorf("could not find a NSM address")
}
