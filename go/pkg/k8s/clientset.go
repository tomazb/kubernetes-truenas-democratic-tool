package k8s

import (
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
)

const defaultClientTimeout = 30 * time.Second

func buildRESTConfig(config Config) (*rest.Config, error) {
	if config.Timeout == 0 {
		config.Timeout = defaultClientTimeout
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.QPS == 0 {
		config.QPS = 50.0
	}
	if config.Burst == 0 {
		config.Burst = 100
	}

	var restConfig *rest.Config
	var err error

	if config.InCluster {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		kubeconfigPath := config.Kubeconfig
		if kubeconfigPath == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfigPath = filepath.Join(home, ".kube", "config")
			}
		}

		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config from kubeconfig: %w", err)
		}
	}

	restConfig.Timeout = config.Timeout
	restConfig.QPS = config.QPS
	restConfig.Burst = config.Burst
	return restConfig, nil
}

// NewKubernetesClients creates typed clientsets for informers and watches.
func NewKubernetesClients(config Config) (kubernetes.Interface, snapshotclient.Interface, error) {
	restConfig, err := buildRESTConfig(config)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	snapshotClient, err := snapshotclient.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create snapshot client: %w", err)
	}

	return clientset, snapshotClient, nil
}
