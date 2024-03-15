package k8s

import (
	"context"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigMap configmap操作接口（增、删、改、查、更新等）
type ConfigMap interface {
	GetConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	CreateConfigMap(namespace string, configmap *corev1.ConfigMap) error
	DeleteConfigMap(namespace string, configmap *corev1.ConfigMap) error
	ListConfigMap(namespace string) (*corev1.ConfigMapList, error)
	CreateIfNotExistsConfigMap(namespace string, configmap *corev1.ConfigMap) error
	UpdateConfigMap(namespace string, configmap *corev1.ConfigMap) error
	CreateAndUpdateConfigMap(namespace string, configmap *corev1.ConfigMap) error
}

type CfgMapAbstraction struct {
	KubeClient kubernetes.Interface
	Logger     zap.Logger
}

func (c *CfgMapAbstraction) GetConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	configmap, err := c.KubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMap.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configmap, nil
}

func (c *CfgMapAbstraction) CreateConfigMap(namespace string, configmap *corev1.ConfigMap) error {
	_, err := c.KubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
	if err != nil {
		c.Logger.Error("create configmap failed", zap.Any("namespace", namespace),
			zap.Any("name", configmap.Name))
		return err
	}
	return nil
}

func (c *CfgMapAbstraction) DeleteConfigMap(namespace string, configmap *corev1.ConfigMap) error {
	return c.KubeClient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(),
		configmap.Name, metav1.DeleteOptions{})
}

func (c *CfgMapAbstraction) ListConfigMap(namespace string) (*corev1.ConfigMapList, error) {
	return c.KubeClient.CoreV1().ConfigMaps(namespace).List(context.TODO(),
		metav1.ListOptions{})
}

func (c *CfgMapAbstraction) CreateIfNotExistsConfigMap(namespace string, configmap *corev1.ConfigMap) error {
	_, err := c.GetConfigMap(namespace, configmap)
	if err != nil {
		if errors.IsNotFound(err) {
			c.Logger.Info("GetConfigMap is not Found, The configMap will be created",
				zap.Any("namespace", namespace),
				zap.Any("name", configmap.Name))
			err := c.CreateConfigMap(namespace, configmap)
			if err != nil {
				c.Logger.Error("Description Failed to create configmap",
					zap.Any("namespace", namespace),
					zap.Any("errors", err))
				return err
			}
		}
	}
	return nil
}

func (c *CfgMapAbstraction) UpdateConfigMap(namespace string, configmap *corev1.ConfigMap) error {
	_, err := c.KubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
	if err != nil {
		c.Logger.Error("Description Failed to update configmap", zap.Any("errors", err),
			zap.Any("namespace", namespace),
			zap.Any("name", configmap.Name))
		return err
	}
	c.Logger.Info("configmap is updated successfully", zap.Any("namespace", namespace), zap.Any("name", configmap.Name))
	return nil
}

// CreateAndUpdateConfigMap 查寻configMap是否存在，不存在则创建，存在则更新
func (c *CfgMapAbstraction) CreateAndUpdateConfigMap(namespace string, configmap *corev1.ConfigMap) error {
	cacheCfgMap, err := c.GetConfigMap(namespace, configmap)
	if err != nil {
		if errors.IsNotFound(err) {
			c.Logger.Info("GetConfigMap is not found, The ConfigMap will be Created")
			err := c.CreateConfigMap(namespace, configmap)
			if err != nil {
				c.Logger.Error("Create ConfigMap failed",
					zap.Any("errors", err), zap.Any("name",
						configmap.Name))
				return err
			}
		}
	}
	configmap.ResourceVersion = cacheCfgMap.ResourceVersion
	return c.UpdateConfigMap(namespace, configmap)
}

func NewCfgMapAbstraction(kubeclient kubernetes.Interface, logger zap.Logger) *CfgMapAbstraction {
	return &CfgMapAbstraction{
		Logger:     logger,
		KubeClient: kubeclient,
	}
}
