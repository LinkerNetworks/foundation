package workspacefs

import (
	"os"
	"testing"

	"bitbucket.org/linkernetworks/aurora/src/config"
	"bitbucket.org/linkernetworks/aurora/src/entity"
	"bitbucket.org/linkernetworks/aurora/src/service/kubernetes"
	"bitbucket.org/linkernetworks/aurora/src/service/mongo"
	"bitbucket.org/linkernetworks/aurora/src/service/redis"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

const (
	testingConfigPath = "../../../config/testing.json"
)

func TestWorkspaceServiceWakeup(t *testing.T) {
	if _, defined := os.LookupEnv("TEST_K8S"); !defined {
		t.SkipNow()
		return
	}

	//Get mongo service
	cf := config.MustRead(testingConfigPath)

	kubernetesService := kubernetes.NewFromConfig(cf.Kubernetes)
	mongoService := mongo.New(cf.Mongo.Url)
	redisService := redis.New(cf.Redis)
	clientset, err := kubernetesService.CreateClientset()
	assert.NoError(t, err)
	fs := New(cf, mongoService, clientset, redisService)

	// proxyURL := "/v1/workspaces/proxy/"
	context := mongoService.NewSession()
	defer context.Close()

	vName := "testmount"
	workspace := entity.Workspace{
		ID:    bson.NewObjectId(),
		Name:  "testing workspace",
		Type:  "general",
		Owner: bson.NewObjectId(),
		MainVolume: entity.PersistentVolumeClaim{
			Name: vName,
		},
	}

	err = context.C(entity.WorkspaceCollectionName).Insert(workspace)
	assert.NoError(t, err)
	defer context.C(entity.WorkspaceCollectionName).Remove(bson.M{"_id": workspace.ID})

	_, err = fs.WakeUp(&workspace)
	assert.NoError(t, err)
	newWP := entity.Workspace{}

	//Check the PodName has been update
	context.C(entity.WorkspaceCollectionName).Find(bson.M{"_id": workspace.ID}).One(&newWP)
	assert.Equal(t, newWP.DeploymentID(), entity.WorkspacePodNamePrefix+workspace.ID.Hex())

	_, err = fs.Delete(&workspace)
	assert.NoError(t, err)

	volumes, err := fs.GetKubeVolume(&workspace)
	assert.NoError(t, err)
	assert.Equal(t, volumes[0].ClaimName, vName)
	assert.Equal(t, volumes[0].VolumeMount.Name, vName)
	assert.Equal(t, volumes[0].VolumeMount.MountPath, WorkspaceMainVolumeMountPoint)
}
