package k8s

import (
	"context"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type JobOptions interface {
	GetJob(namespace, name string) (*batchv1.Job, error)
	CreateJob(namespace string, job *batchv1.Job) error
	CreateJobIfNotExistsJob(namespace string, job *batchv1.Job) error
}

type JobService struct {
	KubeClient kubernetes.Interface
	Logger     zap.Logger
}

func (s *JobService) GetJob(namespace, name string) (*batchv1.Job, error) {
	job, err := s.KubeClient.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		s.Logger.Error("GetJob failed", zap.Error(err))
		return nil, err
	}
	return job, err

}

func (s *JobService) CreateJob(namespace string, job *batchv1.Job) error {
	_, err := s.KubeClient.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		s.Logger.Error("create job failed", zap.Any("namespace", namespace), zap.Any("name", job.Name))
		return err
	}
	s.Logger.Info("CreateJob is being created", zap.Any("namespace", namespace), zap.Any("name", job.Name))
	return nil
}

func (s *JobService) CreateJobIfNotExistsJob(namespace string, job *batchv1.Job) error {
	_, err := s.GetJob(namespace, job.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			s.Logger.Info("Check that the job does not exist and a new job is to be created", zap.Any("namespace", namespace), zap.Any("name", job.Name))
			err := s.CreateJob(namespace, job)
			if err != nil {
				s.Logger.Error("CreateJob failed", zap.Error(err))
				return err
			}
		}
		s.Logger.Info("job creation succeeds", zap.Any("namespace", namespace), zap.Any("name", job.Name))
	}
	s.Logger.Info("job already exists in the namespace", zap.Any("namespace", namespace), zap.Any("name", job.Name))
	return nil
}
