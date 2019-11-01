package nsm

import "github.com/layer5io/meshery-nsm/meshes"

type supportedOperation struct {
	// a friendly name
	name string
	// the template file name
	templateName string
	// the app label
	appLabel string
	// // returnLogs specifies if the operation logs should be returned
	// returnLogs bool
	opType meshes.OpCategory
}

const (
	customOpCommand       = "custom"
	installNSMCommand     = "nsm_install"
	installICMPCommand    = "icmp_sample_install"
	installVPNCommand     = "vpn_sample_install"
	installVPNICMPCommand = "vpn_icmp_sample_install"
	installHelloNSMApp    = "hello_nsm_app"
)

var supportedOps = map[string]supportedOperation{
	installNSMCommand: {
		name:   "Network Service Mesh",
		opType: meshes.OpCategory_INSTALL,
	},
	installICMPCommand: {
		name:   "ICMP Application",
		opType: meshes.OpCategory_SAMPLE_APPLICATION,
	},
	installVPNCommand: {
		name:   "VPN Application",
		opType: meshes.OpCategory_SAMPLE_APPLICATION,
	},
	installVPNICMPCommand: {
		name:   "VPN ICMP Application",
		opType: meshes.OpCategory_SAMPLE_APPLICATION,
	},
	installHelloNSMApp: {
		name:         "Hello NSM Application",
		opType:       meshes.OpCategory_SAMPLE_APPLICATION,
		templateName: "hello-nsm-sample-application.yaml",
	},
	customOpCommand: {
		name:   "Custom YAML",
		opType: meshes.OpCategory_CUSTOM,
	},
}
