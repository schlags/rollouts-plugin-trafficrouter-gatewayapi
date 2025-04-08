package utils

import (
	"encoding/json"

	pluginTypes "github.com/argoproj/argo-rollouts/utils/plugin/types"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here
	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, pluginTypes.RpcError{ErrorString: err.Error()}
	}
	return config, nil
}

func SetupLog() *log.Entry {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(
		&log.JSONFormatter{
		},
	)
	return log.WithFields(log.Fields{"plugin": "trafficrouter"})
}

func GetOrCreateConfigMap(name string, options CreateConfigMapOptions) (*v1.ConfigMap, error) {
	clientset := options.Clientset
	ctx := options.Ctx
	configMap, err := clientset.Get(ctx, name, metav1.GetOptions{})
	if err != nil && !kubeErrors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		return configMap, err
	}
	configMap.Name = name
	configMap, err = clientset.Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return configMap, err
}

func GetConfigMapData(configMap *v1.ConfigMap, configMapKey string, destination any) error {
	if configMap.Data != nil && configMap.Data[configMapKey] != "" {
		err := json.Unmarshal([]byte(configMap.Data[configMapKey]), &destination)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateConfigMapData(configMap *v1.ConfigMap, configMapData any, options UpdateConfigMapOptions) error {
	clientset := options.Clientset
	rawConfigMapData, err := json.Marshal(configMapData)
	if err != nil {
		return err
	}
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	configMap.Data[options.ConfigMapKey] = string(rawConfigMapData)
	_, err = clientset.Update(options.Ctx, configMap, metav1.UpdateOptions{})
	return err
}

func DoTransaction(logCtx *log.Entry, taskList ...Task) error {
	var err, reverseErr error
	for index, task := range taskList {
		err = task.Action()
		if err == nil {
			continue
		}
		logCtx.Error(err.Error())
		for i := index - 1; i > -1; i-- {
			reverseErr = taskList[i].ReverseAction()
			if reverseErr != nil {
				return reverseErr
			}
		}
		return err
	}
	return nil
}
