// Copyright 2020 The Merlin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/gojek/merlin/batch"
	"github.com/gojek/merlin/batch/mocks"
	imageBuilderMock "github.com/gojek/merlin/imagebuilder/mocks"
	"github.com/gojek/merlin/mlp"
	"github.com/gojek/merlin/models"
	storageMock "github.com/gojek/merlin/storage/mocks"
)

const (
	envName  = "test-env"
	imageRef = "gojek/my-image:1"
)

var (
	now                    = time.Now()
	environmentLabel       = "dev"
	isDefaultPredictionJob = true
	predJobEnv             = &models.Environment{
		Id:                     1,
		Name:                   envName,
		IsPredictionJobEnabled: true,
		IsDefaultPredictionJob: &isDefaultPredictionJob,
		DefaultPredictionJobResourceRequest: &models.PredictionJobResourceRequest{
			DriverCpuRequest:      "1",
			DriverMemoryRequest:   "512Mi",
			ExecutorReplica:       1,
			ExecutorCpuRequest:    "2",
			ExecutorMemoryRequest: "1024Mi",
		},
	}
	project = mlp.Project{
		Id:     1,
		Name:   "my-project",
		Team:   "dsp",
		Stream: "dsp",
		Labels: mlp.Labels{
			{
				Key: "my-key", Value: "my-value",
			},
		},
	}
	model = &models.Model{
		Id:           1,
		ProjectId:    1,
		Project:      project,
		ExperimentId: 0,
		Name:         "my-model",
		Type:         models.ModelTypePyFuncV2,
	}
	version = &models.Version{
		Id:      3,
		ModelId: 1,
		Model:   model,
	}
	job = &models.PredictionJob{
		Id:   0,
		Name: fmt.Sprintf("%s-%s-%s", model.Name, version.Id, strconv.FormatInt(now.UnixNano(), 10)[:13]),
		Metadata: models.Metadata{
			Team:        project.Team,
			Stream:      project.Stream,
			App:         model.Name,
			Environment: environmentLabel,
			Labels:      project.Labels,
		},
		VersionId:       3,
		VersionModelId:  1,
		ProjectId:       models.Id(project.Id),
		EnvironmentName: predJobEnv.Name,
		Environment:     predJobEnv,
		Config: &models.Config{
			JobConfig:       nil,
			ResourceRequest: predJobEnv.DefaultPredictionJobResourceRequest,
			EnvVars: models.EnvVars{
				{
					Name:  "key",
					Value: "value",
				},
			},
		},
		Status: models.JobPending,
	}
	reqJob = &models.PredictionJob{
		VersionId:      3,
		VersionModelId: 1,
		Config: &models.Config{
			EnvVars: models.EnvVars{
				{
					Name:  "key",
					Value: "value",
				},
			},
		},
	}
)

func TestGetPredictionJob(t *testing.T) {
	svc, _, _, mockStorage := newMockPredictionJobService()
	mockStorage.On("Get", job.Id).Return(job, nil)
	j, err := svc.GetPredictionJob(predJobEnv, model, version, job.Id)
	assert.NoError(t, err)
	assert.Equal(t, job, j)
	mockStorage.AssertExpectations(t)
}

func TestListPredictionJob(t *testing.T) {
	jobs := []*models.PredictionJob{job}
	svc, _, _, mockStorage := newMockPredictionJobService()
	query := &ListPredictionJobQuery{
		Id:        1,
		Name:      "test",
		ModelId:   2,
		VersionId: 3,
		Status:    models.JobFailed,
		Error:     "runtime error",
	}

	expDbQuery := &models.PredictionJob{
		Id:             query.Id,
		Name:           query.Name,
		VersionId:      query.VersionId,
		VersionModelId: query.ModelId,
		ProjectId:      models.Id(project.Id),
		Status:         query.Status,
		Error:          query.Error,
	}
	mockStorage.On("List", expDbQuery).Return(jobs, nil)
	j, err := svc.ListPredictionJobs(project, query)
	assert.NoError(t, err)
	assert.Equal(t, jobs, j)
	mockStorage.AssertExpectations(t)
}

func TestCreatePredictionJob(t *testing.T) {
	svc, mockControllers, mockImageBuilder, mockStorage := newMockPredictionJobService()

	// test positive case
	savedJob := new(models.PredictionJob)
	err := copier.Copy(savedJob, job)
	savedJob.Config.ImageRef = imageRef

	mockStorage.On("Save", job).Return(nil)
	mockImageBuilder.On("BuildImage", project, model, version).Return(imageRef, nil)
	mockController := mockControllers[envName]
	mockController.(*mocks.Controller).On("Submit", savedJob, project.Name).Return(nil)

	j, err := svc.CreatePredictionJob(predJobEnv, model, version, reqJob)
	time.Sleep(10 * time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job, j)

	mockStorage.AssertExpectations(t)
	mockImageBuilder.AssertExpectations(t)
	mockController.(*mocks.Controller).AssertExpectations(t)
}

func TestStopPredictionJob(t *testing.T) {
	svc, mockControllers, mockImageBuilder, mockStorage := newMockPredictionJobService()

	// test positive case
	savedJob := new(models.PredictionJob)
	err := copier.Copy(savedJob, job)
	savedJob.Config.ImageRef = imageRef

	mockStorage.On("Get", job.Id).Return(job, nil)
	mockController := mockControllers[envName]
	mockController.(*mocks.Controller).On("Stop", job, project.Name).Return(nil)

	j, err := svc.StopPredictionJob(predJobEnv, model, version, job.Id)
	time.Sleep(10 * time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, job, j)

	mockStorage.AssertExpectations(t)
	mockImageBuilder.AssertExpectations(t)
	mockController.(*mocks.Controller).AssertExpectations(t)
}

func TestInvalidResourceRequest(t *testing.T) {
	tests := []struct {
		name            string
		resourceRequest *models.PredictionJobResourceRequest
		wantErrMsg      string
	}{
		{
			name: "invalid driver cpu request",
			resourceRequest: &models.PredictionJobResourceRequest{
				DriverCpuRequest: "1x",
			},
			wantErrMsg: fmt.Sprintf("invalid driver cpu request: 1x"),
		},
		{
			name: "invalid driver memory request",
			resourceRequest: &models.PredictionJobResourceRequest{
				DriverMemoryRequest: "1x",
			},
			wantErrMsg: fmt.Sprintf("invalid driver memory request: 1x"),
		},
		{
			name: "invalid executor cpu request",
			resourceRequest: &models.PredictionJobResourceRequest{
				ExecutorCpuRequest: "1x",
			},
			wantErrMsg: fmt.Sprintf("invalid executor cpu request: 1x"),
		},
		{
			name: "invalid executor memory request",
			resourceRequest: &models.PredictionJobResourceRequest{
				ExecutorMemoryRequest: "1x",
			},
			wantErrMsg: fmt.Sprintf("invalid executor memory request: 1x"),
		},
		{
			name: "invalid executor replica",
			resourceRequest: &models.PredictionJobResourceRequest{
				ExecutorReplica: -1,
			},
			wantErrMsg: fmt.Sprintf("invalid executor replica: -1"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, _, _, _ := newMockPredictionJobService()
			reqJob.Config = &models.Config{
				ResourceRequest: test.resourceRequest,
			}
			_, err := svc.CreatePredictionJob(predJobEnv, model, version, reqJob)
			assert.Error(t, err)
			assert.Equal(t, test.wantErrMsg, err.Error())
		})
	}
}

func TestPredictionJobService_ListContainers(t *testing.T) {
	project := mlp.Project{Id: 1, Name: "my-project"}
	model := &models.Model{Id: 1, Name: "model", Type: models.ModelTypeXgboost, Project: project, ProjectId: models.Id(project.Id)}
	version := &models.Version{Id: 1}
	job := &models.PredictionJob{Id: 2, VersionId: 1, VersionModelId: 1}

	type args struct {
		env     *models.Environment
		model   *models.Model
		version *models.Version
		job     *models.PredictionJob
	}

	type componentMock struct {
		imageBuilderContainer *models.Container
		modelContainers       []*models.Container
	}

	tests := []struct {
		name      string
		args      args
		mock      componentMock
		wantError bool
	}{
		{
			"success: non-pyfunc model",
			args{
				predJobEnv, model, version, job,
			},
			componentMock{
				nil,
				[]*models.Container{
					{
						Name:       "user-container",
						PodName:    "mymodel-2-predictor-default-hlqgv-deployment-6f478cbc67-mp7zf",
						Namespace:  project.Name,
						Cluster:    env.Cluster,
						GcpProject: env.GcpProject,
					},
				},
			},
			false,
		},
		{
			"success: pyfunc model",
			args{
				predJobEnv, model, version, job,
			},
			componentMock{
				&models.Container{
					Name:       "kaniko-0",
					PodName:    "pod-1",
					Namespace:  "mlp",
					Cluster:    env.Cluster,
					GcpProject: env.GcpProject,
				},
				[]*models.Container{
					{
						Name:       "user-container",
						PodName:    "mymodel-2-predictor-default-hlqgv-deployment-6f478cbc67-mp7zf",
						Namespace:  project.Name,
						Cluster:    env.Cluster,
						GcpProject: env.GcpProject,
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		imgBuilder := &imageBuilderMock.ImageBuilder{}
		imgBuilder.On("GetContainers", mock.Anything, mock.Anything, mock.Anything).
			Return(tt.mock.imageBuilderContainer, nil)

		svc, mockControllers, _, _ := newMockPredictionJobService()
		mockController := mockControllers[tt.args.env.Name]
		mockController.(*mocks.Controller).On("GetContainers", "my-project", "prediction-job-id=2").Return(tt.mock.modelContainers, nil)

		containers, err := svc.ListContainers(tt.args.env, tt.args.model, tt.args.version, tt.args.job)
		if !tt.wantError {
			assert.Nil(t, err, "unwanted error %v", err)
		} else {
			assert.NotNil(t, err, "expected error")
		}

		assert.NotNil(t, containers)
		expContainer := len(tt.mock.modelContainers)
		if tt.args.model.Type == models.ModelTypePyFunc {
			expContainer += 1
		}
		assert.Equal(t, expContainer, len(containers))
	}
}

func newMockPredictionJobService() (PredictionJobService, map[string]batch.Controller, *imageBuilderMock.ImageBuilder, *storageMock.PredictionJobStorage) {
	mockController := &mocks.Controller{}
	mockControllers := map[string]batch.Controller{
		predJobEnv.Name: mockController,
	}
	mockImageBuilder := &imageBuilderMock.ImageBuilder{}
	mockStorage := &storageMock.PredictionJobStorage{}
	mockClock := clock.NewFakeClock(now)
	return NewPredictionJobService(mockControllers, mockImageBuilder, mockStorage, mockClock, environmentLabel), mockControllers, mockImageBuilder, mockStorage
}