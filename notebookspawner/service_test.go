package notebookspawner

import (
	"os"
	"testing"

	"bitbucket.org/linkernetworks/aurora/src/config"
	"bitbucket.org/linkernetworks/aurora/src/entity"
	"bitbucket.org/linkernetworks/aurora/src/service/kubernetes"
	"bitbucket.org/linkernetworks/aurora/src/service/mongo"
	"bitbucket.org/linkernetworks/aurora/src/service/redis"
	// "bitbucket.org/linkernetworks/aurora/src/service/notebookspawner/notebook"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

const (
	testingConfigPath = "../../../config/testing.json"
)

func TestNotebookSpawnerService(t *testing.T) {
	if _, defined := os.LookupEnv("TEST_K8S"); !defined {
		t.SkipNow()
		return
	}

	var notebookImage = "jupyter/minimal-notebook"
	var err error

	//Get mongo service
	cf := config.MustRead(testingConfigPath)

	kubernetesService := kubernetes.NewFromConfig(cf.Kubernetes)
	mongoService := mongo.New(cf.Mongo.Url)
	redisService := redis.NewService(cf.Redis)
	spawner := New(cf, mongoService, kubernetesService, redisService)

	// proxyURL := "/v1/notebooks/proxy/"
	context := mongoService.NewSession()
	defer context.Close()

	userId := bson.NewObjectId()

	workspace := entity.Workspace{
		ID:    bson.NewObjectId(),
		Name:  "testing workspace",
		Type:  "general",
		Owner: userId,
	}

	err = context.C(entity.WorkspaceCollectionName).Insert(workspace)
	assert.NoError(t, err)
	defer context.C(entity.WorkspaceCollectionName).Remove(bson.M{"_id": workspace.ID})

	notebookID := bson.NewObjectId()
	notebook := entity.Notebook{
		ID:          notebookID,
		Image:       notebookImage,
		WorkspaceID: workspace.ID,
		Url:         cf.Jupyter.BaseUrl + "/" + notebookID.Hex(),
		CreatedBy:   userId,
	}
	err = context.C(entity.NotebookCollectionName).Insert(notebook)
	assert.NoError(t, err)
	defer context.C(entity.NotebookCollectionName).Remove(bson.M{"_id": notebook.ID})

	_, err = spawner.Start(&notebook)
	assert.NoError(t, err)

	err = spawner.Sync(&notebook)
	assert.NoError(t, err)

	_, err = spawner.Stop(&notebook)
	assert.NoError(t, err)

	err = spawner.Sync(&notebook)
	assert.NoError(t, err)

}
