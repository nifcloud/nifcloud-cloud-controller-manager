package nifcloud

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
	cloudprovider "k8s.io/cloud-provider"
)

var execCommand = exec.Command

// getInstanceUniqueIDFromProviderID converts provider id to instance id
// valid provider id:
//   - nifcloud:///<zone>/<InstanceId>
//   - nifcloud:////<InstanceId>
//   - <InstanceId>
func getInstanceUniqueIDFromProviderID(providerID string) (string, error) {
	s := string(providerID)

	if !strings.HasPrefix(s, "nifcloud://") {
		// Assume a bare aws volume id (vol-1234...)
		// Build a URL with an empty host (zone)
		s = "nifcloud://" + "/" + "/" + s
	}
	url, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("Invalid instance name (%s): %v", providerID, err)
	}
	if url.Scheme != "nifcloud" {
		return "", fmt.Errorf("Invalid scheme for NIFCLOUD instance (%s)", providerID)
	}

	instanceID := ""
	tokens := strings.Split(strings.Trim(url.Path, "/"), "/")
	if len(tokens) == 1 {
		// instanceId
		instanceID = tokens[0]
	} else if len(tokens) == 2 {
		// zone/instanceId
		instanceID = tokens[1]
	}

	if instanceID == "" || !nifcloudInstanceRegMatch.MatchString(instanceID) {
		return "", fmt.Errorf("Invalid format for NIFCLOUD instance (%s)", providerID)
	}

	return instanceID, nil
}

func isSingleInstance(instances []Instance, name string) error {
	if len(instances) == 0 {
		return cloudprovider.InstanceNotFound
	}
	if len(instances) > 1 {
		return fmt.Errorf("multiple instances (%d) found for instance: %q", len(instances), name)
	}

	return nil
}

func getInstanceIDFromGuestInfo() (string, error) {
	const cmd = "vmtoolsd --cmd 'info-get guestinfo.hostname'"
	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to parse command %q: %v", cmd, err)
	}
	out, err := execCommand(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("could not get instance id from vmtoolsd: %v (%v)", string(out), err)
	}

	instanceID := strings.TrimSpace(string(out))
	if len(instanceID) == 0 {
		return "", fmt.Errorf("guestinfo does not have hostname")
	}

	return instanceID, nil
}
