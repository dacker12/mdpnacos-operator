package v1

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

// TODO
// 计划在这里些一些最底层的数据、状态、处理等的工具方法或函数
// 获取pod Cpu平均使用率
//func GetPodCpuUsage(nacosSpec *Mdpnacos) (int32, error) {
//	clientSet, err := CreateClientSet()
//	if err != nil {
//		log.Fatal("Failed to create client: %v", err)
//	}
//
//	pods, err := ListPodsCreatedBySts(clientSet, nacosSpec.Spec.STSnamespace, nacosSpec)
//}

// 获取pod实例
func ListPodsCreatedBySts(clientset *kubernetes.Clientset, namespace string, nacosSpec *Mdpnacos) ([]v1.Pod, error) {
	//假设知道创建的sts有特定的标签
	//TODO
	//STSSelector 传进来的键值对是否要转换一下
	//labelselector := "k8s-app=tigera-operator"

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: nacosSpec.Spec.STSSelector})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

// 创建clientset客户端
func CreateClientSet() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	fmt.Println(kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
