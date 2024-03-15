/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nacosv1 "xhx-jspt.cloud/mdpnacos/api/v1"
)

// MdpnacosReconciler reconciles a Mdpnacos object
type MdpnacosReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nacos.xhx-jspt.cloud,resources=mdpnacos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nacos.xhx-jspt.cloud,resources=mdpnacos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nacos.xhx-jspt.cloud,resources=mdpnacos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Mdpnacos object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *MdpnacosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	//初始化日志框架zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Printf("日志同步失败，请检查,错误为%s", err)
		}
	}(logger)

	cont := context.Background()

	//启动reconcle逻辑
	logger.Info("启动Reconcle逻辑")
	//实例化数据结构,接收、存储查询到的结构体
	instance := &nacosv1.Mdpnacos{}

	//通过客户端查询集群中已有的Mdpnacos资源对象
	err = r.Get(cont, req.NamespacedName, instance)
	if err != nil {
		//如果没有实例就返回空结果，外部就不回立即调用reconcile方法
		if errors.IsNotFound(err) {
			logger.Info("instance is not found,maybe remove")
			return reconcile.Result{}, nil
		}
		logger.Error("Get集群实例失败", zap.Any("error：", err))
		//返回错误信息给外部调用方
		return ctrl.Result{}, err
	}
	logger.Info("打印instance", zap.Any("instance", instance))

	//初始化STS数据结构
	nacosSTS := &appsv1.StatefulSet{}

	//使用客户端查询集群中STS
	err = r.Get(cont, req.NamespacedName, nacosSTS)

	//处理查询后无结果或异常的情况
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("nacos STS not exists, Start creating an sts and headless")
			//先创建headless service
			if err = CreateSvcIfNotExist(cont, r, instance, req); err != nil {
				logger.Error("The headless service Creation failure ")
				return ctrl.Result{}, err
			}
			logger.Info("The headless is created successfully")
			//初始化pvc数据结构
			nacospvc := &corev1.PersistentVolumeClaim{}

			//使用客户端查询集群中的pvc
			err = r.Get(cont, req.NamespacedName, nacospvc)
			fmt.Printf("nacospvc=%s", nacospvc)
			if errors.IsNotFound(err) {
				logger.Info("pvc does not exist")
				if _, err = CreatePVCAndPV(cont, instance, r, req); err != nil {
					logger.Error("nacos pv or pv Creation failure")
					return ctrl.Result{}, err
				}
				logger.Info("nacos pv or pvc creation successfully")
			}

			//创建STS资源对象
			if err = NewCreateSts(cont, r, instance); err != nil {
				logger.Error("nacos sts creation failure")
				return ctrl.Result{}, err
			}
			logger.Info("nacos sts creation successfully")
		}
	}

	//如果在集群中查询到sts则执行下面逻辑
	//获取集群中当前副本数量
	currencyReplicas := *nacosSTS.Spec.Replicas
	logger.Info("currencyReplicas=", zap.Any("currencyReplicas", currencyReplicas))

	//计算期望的副本数量
	expectReplicas := CalculateReplicasNum(instance)
	logger.Info("expectReplicas=", zap.Any("expectReplicas", expectReplicas))

	//如果实际副本数量与期望副本数量相等，则直接返回
	if currencyReplicas == expectReplicas {
		logger.Info("The current number of replicas equals the actual number of replicas", zap.Any("currencyReplica",
			currencyReplicas), zap.Any("expectReplicas", expectReplicas))
		return ctrl.Result{}, err
	}

	//如果当前副本数与期望副本数不想等，则需要调整
	logger.Info("Change the number of current replicas to the expected value", zap.Any("currencyReplicas",
		currencyReplicas), zap.Any("expectReplicas", expectReplicas))
	*nacosSTS.Spec.Replicas = expectReplicas

	//通过客户端更新sts实例数
	if err = r.Update(ctx, nacosSTS); err != nil {
		logger.Error("update currentstatus error")
		return ctrl.Result{}, err
	}

	//nacos更新成功后需要更新状态
	if err = UpdateCurrentStatus(ctx, r, instance); err != nil {
		logger.Error("update status error")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MdpnacosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nacosv1.Mdpnacos{}).
		Complete(r)
}

// AverageQPSAllCurrentInstances 获取当前所有nacos实例Qps平均值
func AverageQPSAllCurrentInstances() {
	//TODO
}

// TODO
// QpsConversionPodNum 根据Qps平均值计算需要多少pod来负载，计算出pod数量
// 计算pod数量时也需要结合Mdpnacos Yml传入的pod数量的最大最小值进行限制
// 也需要结合伸缩系数进行弹性伸缩频率、伸缩幅度的控制
func QpsConversionPodNum(spec *nacosv1.Mdpnacos) int32 {
	//单个实例的Qps
	singleAppQps := *(spec.Spec.ScalingPolicy.ScaleQpsPolicy.SingleAppQps)

	//期望的总负载Qps
	allAppQps := *(spec.Spec.ScalingPolicy.ScaleQpsPolicy.AllAppQpss)

	//期望的实例数量
	replicaSpec := allAppQps / singleAppQps

	//TODO 这里是否需要考虑副本数量一直创建的问题，应该有个上限

	if allAppQps%singleAppQps > 0 {
		replicaSpec++
	}
	return replicaSpec
}

//TODO
//结合CPU、MEM使用率进行弹性伸缩
//yml文件中需要传入期望的内存和CPU期望的平均值
//扩容时，高于设置的最大值则扩容，冷却期为0s
//缩容时，低于最小值缩容，冷却期为300s

// CreateSvcIfNotExist 检查service是否已创建，未创建则创建该service
func CreateSvcIfNotExist(con context.Context, nacos *MdpnacosReconciler, mdpnacos *nacosv1.Mdpnacos, req ctrl.Request) error {
	//初始化日志框架zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Println("日志同步失败，请检查")
		}
	}(logger)

	//logout := nacos.Log.WithValues("func", "createSvc")

	service := &corev1.Service{}

	//如果能正常get到service，则无需再次创建，该函数直接返回即可
	err = nacos.Get(con, req.NamespacedName, service)
	if err == nil {
		logger.Info("Service 存在，不需要在创建")
		return nil
	}

	//如果Get方法报非“not found”就返回错误
	if !errors.IsNotFound(err) {
		logger.Error("query service error")
	}

	//构造service数据结构，如果查询到service不存在则创建该 headless service
	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mdpnacos.Namespace,
			Name:      mdpnacos.Name,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       8080,
				TargetPort: intstr.FromInt32(*mdpnacos.Spec.TargetPort), //该字段为申明service暴露的端口
			},
			},
			Selector: map[string]string{
				"app": mdpnacos.Spec.STSSelector,
			},
		},
	}

	//设置自定义CRD资源与部署控制器deployment的关联关系
	logger.Info("设置关联关系")
	err = controllerutil.SetControllerReference(mdpnacos, service, nacos.Scheme)
	if err != nil {
		logger.Error("SetControllerReference 设置资源关联关系报错")
		return err
	}

	//排除一些异常情况后，我们开始创建service
	logger.Info("start create service")
	err = nacos.Create(con, service)
	if err != nil {
		logger.Error("create service error")
		return err
	}

	logger.Info("create service success")
	return nil
}

// NewCreateSts 创建sts资源对象
func NewCreateSts(ctx context.Context, nacosRec *MdpnacosReconciler, mdpnacos *nacosv1.Mdpnacos) error {
	//TODO
	//初始化日志框架zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Println("日志同步失败，请检查")
		}
	}(logger)
	//计算期望的pod数量
	exceptPodNum := QpsConversionPodNum(mdpnacos)

	//实例化一个数据结构，用于创建sts
	newSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:       mdpnacos.Name,
			Namespace:  mdpnacos.Spec.STSnamespace,
			Generation: 2,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: pointer.Int32(exceptPodNum),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": mdpnacos.Spec.STSSelector,
				},
			},
			ServiceName:          mdpnacos.Spec.HeadlessSvcName,
			RevisionHistoryLimit: pointer.Int32(10),
			PodManagementPolicy:  "OrderedReady",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"pod.alpha.kubernetes.io/initialized": "true",
					},
					Labels: map[string]string{
						"app": mdpnacos.Spec.STSSelector,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            mdpnacos.Spec.STSname,
							Image:           mdpnacos.Spec.Image,
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								{
									Name:          "client",
									Protocol:      "TCP",
									ContainerPort: 8848,
								},
								{
									Name:          "client-rpc",
									Protocol:      "TCP",
									ContainerPort: 9848,
								},
								{
									Name:          "raft-rpc",
									Protocol:      "TCP",
									ContainerPort: 9849,
								},
								{
									Name:          "old-raft-rpc",
									Protocol:      "TCP",
									ContainerPort: 7848,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("500Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("500Mi"),
								},
							},
						},
					},
					DNSPolicy:     "ClusterFirst",
					RestartPolicy: "Always",
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{nacosv1.STS_APP_NAME_TAG},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: func() *int32 {
						p := int32(0)
						return &p
					}(),
				},
			},
		},
	}

	//建立资源Mdpnacos与sts的关联关系，当MdpNacos删除市会将对应的sts资源对象一起删除
	logger.Info("设置sts与Mdpnacos资源的关联关系")
	err = controllerutil.SetControllerReference(mdpnacos, newSts, nacosRec.Scheme)

	if err != nil {
		logger.Error("设置Mdpnacos与deployment关联关系失败")
		return err
	}

	logger.Info("start create deployment")
	err = nacosRec.Create(ctx, newSts)
	if err != nil {
		logger.Error("自动创建sts失败%v")
		return err
	}

	logger.Info("自动创建naocs sts成功！")
	return nil
}

func UpdateCurrentStatus(con context.Context, nacosRes *MdpnacosReconciler, mdpnacos *nacosv1.Mdpnacos) error {
	//logout := nacosRes.Log.WithValues("func", "UpdateCurrentStatus")
	//初始化日志框架zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Println("日志同步失败，请检查")
		}
	}(logger)

	//单个pod的QPS
	singlePodQPS := *(mdpnacos.Spec.ScalingPolicy.ScaleQpsPolicy.SingleAppQps)

	//pod总数
	replicas := QpsConversionPodNum(mdpnacos)

	//当pod创建完后，当前系统的总QPS计算方法为：singlePodQOS * replicas
	//初始化该字段
	if nil == mdpnacos.Status.AppCurrentQps {
		mdpnacos.Status.AppCurrentQps = new(int32)
	}

	//当前系统的总QPS
	*(mdpnacos.Status.AppCurrentQps) = singlePodQPS * replicas

	logger.Info(fmt.Sprintf("SinglePodQPS [%d], replicas [%d], AppCurrentqps [%d]", singlePodQPS, replicas, *(mdpnacos.Status.AppCurrentQps)))

	if err := nacosRes.Update(con, mdpnacos); err != nil {
		logger.Error("update instance error")
		return err
	}
	return nil
}

// CreatePVCAndPV 创建持久化本地存储pv、pvc
// 持久化目录为/app/appuser/nacosData
func CreatePVCAndPV(con context.Context, mdpnacos *nacosv1.Mdpnacos, r *MdpnacosReconciler, req ctrl.Request) (ctrl.Result, error) {
	//初始化日志框架zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Println("日志同步失败，请检查")
		}
	}(logger)

	myResource := &nacosv1.Mdpnacos{}
	if err := r.Get(context.TODO(), req.NamespacedName, myResource); err != nil {
		return ctrl.Result{}, err
	}

	//实例化pv
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pv-%s", mdpnacos.Spec.STSname),
			Namespace: mdpnacos.Spec.STSnamespace,
		},
		Spec: mdpnacos.Spec.PVSpec,
	}

	//实例化pvc
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pvc-%s", mdpnacos.Spec.STSname),
			Namespace: mdpnacos.Spec.STSnamespace,
		},
		Spec: mdpnacos.Spec.PVCSpec,
	}

	//创建pv
	err = r.Create(con, pv)
	if err != nil {
		logger.Error("create pv error")
		return ctrl.Result{}, err
	}

	logger.Info("create pv success")

	//创建pvc
	err = r.Create(con, pvc)
	if err != nil {
		logger.Error("create pv error")
		return ctrl.Result{}, err
	}
	logger.Info("create pvc success")
	return ctrl.Result{}, nil
}

// CalculateReplicasNum 计算pod实例数
func CalculateReplicasNum(mdpN *nacosv1.Mdpnacos) int32 {
	//初始化日志框架zap
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Println("日志同步失败，请检查")
		}
	}(logger)

	//校验MdpNacos yaml中给定的值是否符合规则
	//nacos集群模式最小副本数不能低于3
	if *mdpN.Spec.MaxMumPod <= *mdpN.Spec.MinMumPod {
		logger.Error("MaxMumPod should not be less than or equal to MinMumPod,The minimum number of instances is 3")
		panic(err)
	} else if *mdpN.Spec.MinMumPod < 3 {
		logger.Error("The minimum number of instances is 3")
		panic(err)
	}

	//根据最大实例数量与最小实例数量的控制范围确定最终的实例个数
	replicanum := QpsConversionPodNum(mdpN)
	if *mdpN.Spec.MinMumPod <= replicanum && replicanum <= *mdpN.Spec.MaxMumPod {
		logger.Info("计算出的副本数量在规划范围内", zap.Any("The replicas number is", replicanum))
		return replicanum
	} else if replicanum < *mdpN.Spec.MinMumPod {
		logger.Info("Cannot be lower than the minimum value", zap.Any("replicanum", replicanum), zap.Any("期望的最小副本数为：", *mdpN.Spec.MinMumPod))
		return *mdpN.Spec.MinMumPod
	} else if replicanum > *mdpN.Spec.MaxMumPod {
		logger.Info("Not higher than the maximum")
		return *mdpN.Spec.MaxMumPod
	}
	return replicanum
}
