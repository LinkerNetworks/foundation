package notebookspawner

import (
	"os"
	"testing"

	"bitbucket.org/linkernetworks/aurora/src/config"
	"bitbucket.org/linkernetworks/aurora/src/entity"
	"bitbucket.org/linkernetworks/aurora/src/service/kubernetes"
	"bitbucket.org/linkernetworks/aurora/src/service/mongo"
	"bitbucket.org/linkernetworks/aurora/src/service/notebookspawner/notebook"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

const (
	NOTEBOOK_NAME  = "5a094b3f2517e191e088e65c"
	NOTEBOOK_IMAGE = "jupyter/minimal-notebook"

	testingConfigPath = "../../../config/testing.json"
)

func TestStartInternalService(t *testing.T) {
	if _, defined := os.LookupEnv("TEST_K8S"); !defined {
		t.SkipNow()
		return
	}

	//Get mongo service
	cf := config.Read(testingConfigPath)

	kubernetesService := kubernetes.NewFromConfig(cf.Kubernetes)
	clientset, err := kubernetesService.CreateClientset()
	assert.NoError(t, err)

	mongoService := mongo.NewMongoService(cf.Mongo.Url)

	spawner := New(cf, mongoService, kubernetesService)

	batchDir := "batch-" + NOTEBOOK_NAME
	baseURL := "/v1/notebooks/proxy/"

	context := mongoService.NewContext()
	defer context.Close()

	workspace := entity.Workspace{
		ID:        bson.NewObjectId(),
		Name:      "testing workspace",
		Type:      "general",
		Directory: batchDir,
	}
	err := context.C(entity.WorkspaceCollectionName).Insert(workspace)
	assert.NoError(t, err)

	notebook := entity.Notebook{
		ID:          bson.NewObjectId(),
		Pod:         entity.NotebookProxyInfo{},
		WorkspaceID: workspace.ID,
	}
	err = context.C(entity.NotebookCollectionName).Insert(notebook)
	assert.NoError(t, err)

	/*
		knb := notebook.KubeNotebook{
			Name:      NOTEBOOK_NAME,
			Workspace: batchDir,
			ProxyURL:  baseURL,
			Image:     NOTEBOOK_IMAGE,
		}
		nbs := NewNotebookService(clientset, mongoService)
		signal, err := nbs.Start(knb)
		assert.NoError(t, err)
		assert.NotNil(t, signal)
		<-signal
	*/
}

/*
func TestStopInternalService(t *testing.T) {
	if _, defined := os.LookupEnv("TEST_K8S"); !defined {
		t.SkipNow()
		return
	}

	cf := config.Read(testingConfigPath)
	kubernetesService := kubernetes.NewFromConfig(cf.Kubernetes)
	clientset, err := kubernetesService.CreateClientset()
	assert.NoError(t, err)

	mongoService := mongo.NewMongoService(cf.Mongo.Url)

	nbs := NewNotebookService(clientset, mongoService)
	knb := notebook.KubeNotebook{
		Name: NOTEBOOK_NAME,
	}
	err = nbs.Stop(knb)
	assert.NoError(t, err)

	context := mongoService.NewContext()
	defer context.Close()

	err = context.C(entity.NotebookCollectionName).Remove(bson.M{"_id": bson.ObjectIdHex(NOTEBOOK_NAME)})
	assert.NoError(t, err)
}
*/
