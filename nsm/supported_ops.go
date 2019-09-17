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
	customOpCommand   = "custom"
	installNSMCommand = "nsm_install"
	//installVPNCommand  = "vpn_install"
	installICMPCommand = "icmp_install"
)

var supportedOps = map[string]supportedOperation{
	installNSMCommand: {
		name:   "Install Network Service Mesh",
		opType: meshes.OpCategory_INSTALL,
	},
	installICMPCommand: {
		name:   "Install ICMP Application",
		opType: meshes.OpCategory_INSTALL,
	},
	/*installVPNCommand: {
		name:   "Install VPN Application",
		opType: meshes.OpCategory_INSTALL,
	},*/

	customOpCommand: {
		name:   "Custom YAML",
		opType: meshes.OpCategory_CUSTOM,
	},
}
