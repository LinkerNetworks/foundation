package notebookspawner

import (
	"bitbucket.org/linkernetworks/aurora/src/config"
	"bitbucket.org/linkernetworks/aurora/src/entity"
	"bitbucket.org/linkernetworks/aurora/src/service/kubernetes"
	"bitbucket.org/linkernetworks/aurora/src/service/mongo"
	"bitbucket.org/linkernetworks/aurora/src/service/notebookspawner/notebook"

	// import global logger
	"bitbucket.org/linkernetworks/aurora/src/logger"

	"gopkg.in/mgo.v2/bson"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const PodNamePrefix = "pod-"

// Object as Pod
type PodFactory interface {
	NewPod(podName string) v1.Pod
}

type PodLabelProvider interface {
	PodLabels() map[string]string
}

type ProxyInfoProvider interface {
	Host() string
	Port() string
	BaseURL() string
}

type DeploymentIDProvider interface {
	DeploymentID() string
}

type PodDeployment interface {
	DeploymentIDProvider
	PodFactory
}

type NotebookPodDeployment interface {
	PodDeployment
	ProxyInfoProvider
}

type NotebookSpawnerService struct {
	Config     config.Config
	Mongo      *mongo.MongoService
	Kubernetes *kubernetes.Service
	namespace  string
}

func New(c config.Config, m *mongo.MongoService, k *kubernetes.Service) *NotebookSpawnerService {
	return &NotebookSpawnerService{c, m, k, "default"}
}

func (s *NotebookSpawnerService) Sync(notebookID bson.ObjectId, pod v1.Pod) error {
	var context = s.Mongo.NewContext()
	defer context.Close()

	podStatus := pod.Status

	info := &entity.NotebookProxyInfo{
		IP: podStatus.PodIP,

		// TODO: extract this as the service configuration
		Port: notebook.NotebookContainerPort,

		// TODO: pull the pod info to another section
		Phase:     podStatus.Phase,
		Message:   podStatus.Message,
		Reason:    podStatus.Reason,
		StartTime: podStatus.StartTime,
	}

	q := bson.M{"_id": notebookID}
	m := bson.M{"$set": bson.M{"pod": info}}
	return context.C(entity.NotebookCollectionName).Update(q, m)
}

func (s *NotebookSpawnerService) DeployPod(notebook PodDeployment) error {
	return nil
}

func (s *NotebookSpawnerService) Start(nb *entity.Notebook) error {
	clientset, err := s.Kubernetes.CreateClientset()
	if err != nil {
		return err
	}

	// TODO: load workspace to ensure the workspace exists
	workspace := filepath.Join(s.Config.Data.BatchDir, "batch-"+nb.WorkspaceID.Hex())

	// Start pod for notebook in workspace(batch)
	knb := notebook.KubeNotebook{
		Notebook:  nb,
		Name:      nb.ID.Hex(),
		Workspace: workspace,
		ProxyURL:  s.Config.Jupyter.BaseUrl,
		Image:     nb.Image,
	}

	podName := "pod-" + knb.DeploymentID()
	pod := knb.NewPod(podName)

	_, err = clientset.Core().Pods(s.namespace).Create(&pod)
	if err != nil {
		return err
	}

	var signal = make(chan bool, 1)
	go func() {
		context := s.Mongo.NewContext()
		defer context.Close()
		o, stop := trackPod(clientset, podName, s.namespace)
	Watch:
		for {
			pod := <-o
			switch phase := pod.Status.Phase; phase {
			case "Pending":
				// updateNotebookProxyInfo(context, knb.Name, pod.Status)
				// Check all containers status in a pod. can't be ErrImagePull or ImagePullBackOff
				for _, c := range pod.Status.ContainerStatuses {
					waitingReason := c.State.Waiting.Reason
					if waitingReason == "ErrImagePull" || waitingReason == "ImagePullBackOff" {
						logger.Errorf("Container is waiting. Reason %s\n", waitingReason)
						break Watch
					}
				}
			case "Running", "Failed", "Succeeded", "Unknown":
				logger.Infof("Notebook %s is %s\n", podName, phase)
				// updateNotebookProxyInfo(context, knb.Name, pod.Status)
				break Watch
			}

		}
		var e struct{}
		signal <- true
		stop <- e
		close(stop)
		close(signal)
		close(o)
	}()
	return nil
}

func (s *NotebookSpawnerService) Stop(nb *entity.Notebook) error {
	clientset, err := s.Kubernetes.CreateClientset()
	if err != nil {
		return err
	}

	knb := notebook.KubeNotebook{
		Notebook: nb,
		Name:     nb.ID.Hex(),
	}

	podName := "pod-" + knb.DeploymentID()
	err = clientset.Core().Pods(s.namespace).Delete(podName, metav1.NewDeleteOptions(0))
	if err != nil {
		return err
	}
	return nil
}
