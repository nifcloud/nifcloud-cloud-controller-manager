package main

import (
	"github.com/nifcloud/nifcloud-cloud-controller-manager/pkg/cloudprovider/providers/nifcloud"
	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	cloudcontrollerconfig "k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/options"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // for client metric registration
	_ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration
	"k8s.io/klog/v2"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	opts, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.Fatalf("unable to initialize command options: %v", err)
	}

	controllerInitializers := app.DefaultInitFuncConstructors
	fss := cliflag.NamedFlagSets{}

	command := app.NewCloudControllerManagerCommand(opts, cloudInitializer, controllerInitializers, fss, wait.NeverStop)

	if err := command.Execute(); err != nil {
		klog.Fatalf("unable to execute command: %v", err)
	}
}

func cloudInitializer(config *cloudcontrollerconfig.CompletedConfig) cloudprovider.Interface {
	cloudConfig := config.ComponentConfig.KubeCloudShared.CloudProvider
	providerName := cloudConfig.Name

	if providerName == "" {
		providerName = nifcloud.ProviderName
	}

	if providerName != nifcloud.ProviderName {
		klog.Fatalf("unknown cloud provider %s, only %s is supported", providerName, nifcloud.ProviderName)
	}

	cloud, err := cloudprovider.InitCloudProvider(providerName, cloudConfig.CloudConfigFile)
	if err != nil {
		klog.Fatalf("cloud provider could not be initialized: %v", err)
	}
	if cloud == nil {
		klog.Fatal("cloud provider is nil")
	}

	return cloud
}
