package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/test/mocks"

	"github.com/stretchr/testify/mock"
)

func TestProcessJobs(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	workerSvc := NewWorkerService(svc, 2, func(model.Job) (string, error) { return "job output", nil }, nil)

	jobs := []model.Job{
		{
			InputFilePath:  "/path/to/input.mp4",
			OutputFilePath: "/path/to/output.mkv",
			SimpleOptions: &model.SimpleOptions{
				QualityPreset:           model.PresetBalanced,
				UseHardwareAcceleration: true,
				AudioQuality:            "medium",
			},
		},
		{
			InputFilePath:  "other/path/to/input.mp4",
			OutputFilePath: "other/path/to/output.mkv",
			SimpleOptions: &model.SimpleOptions{
				QualityPreset:           model.PresetBalanced,
				UseHardwareAcceleration: true,
				AudioQuality:            "medium",
			},
		},
		{
			InputFilePath:  "some/path/to/input.mp4",
			OutputFilePath: "some/path/to/output.mkv",
		},
	}

	results := []string{}
	for _, j := range jobs {
		jobBytes, _ := json.Marshal(j)
		redisMock.On("DequeueJob", mock.Anything).Return(string(jobBytes), nil).Once()

		var unmarshaledJob model.Job
		json.Unmarshal(jobBytes, &unmarshaledJob)

		result, _ := json.Marshal(model.JobResult{Job: unmarshaledJob, Output: "job output", Error: nil})
		results = append(results, string(result))
		redisMock.On("EnqueueJobResult", mock.Anything, mock.Anything).Return(nil)
	}
	ctx, _ := context.WithTimeout(context.TODO(), 3*time.Second)
	redisMock.On("DequeueJob", ctx).Return("", errors.New("cancelled")).Run(func(args mock.Arguments) { <-ctx.Done() })

	workerSvc.Start(ctx)
	workerSvc.Wait()

	for _, r := range results {
		redisMock.AssertCalled(t, "EnqueueJobResult", mock.Anything, r)
	}
}

func TestJobsTaskFails(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	workerSvc := NewWorkerService(svc, 2, func(model.Job) (string, error) { return "job output", nil }, nil)
	mockFunc := func(model.Job) (string, error) {
		workerSvc.WorkFunc = func(model.Job) (string, error) {
			return "job failed", errors.New("job failed")
		}
		return "job output", nil
	}

	workerSvc.WorkFunc = mockFunc

	job := model.Job{
		InputFilePath:  "some/path/to/input.mp4",
		OutputFilePath: "some/path/to/output.mkv",
	}

	results := []string{}

	jobBytes, _ := json.Marshal(job)
	redisMock.On("DequeueJob", mock.Anything).Return(string(jobBytes), nil).Twice()

	var unmarshaledJob model.Job
	json.Unmarshal(jobBytes, &unmarshaledJob)

	result, _ := json.Marshal(model.JobResult{Job: unmarshaledJob, Output: "job output", Error: nil})
	results = append(results, string(result))
	badResult, _ := json.Marshal(model.JobResult{Job: unmarshaledJob, Output: "job failed", Error: errors.New("job failed")})
	results = append(results, string(badResult))

	redisMock.On("EnqueueJobResult", mock.Anything, mock.Anything).Return(nil)
	ctx, _ := context.WithTimeout(context.TODO(), 3*time.Second)
	redisMock.On("DequeueJob", ctx).Return("", errors.New("cancelled")).Run(func(args mock.Arguments) { <-ctx.Done() })

	workerSvc.Start(ctx)
	workerSvc.Wait()

	for _, r := range results {
		redisMock.AssertCalled(t, "EnqueueJobResult", mock.Anything, r)
	}
}

func TestInternalWorkerError(t *testing.T) {
	// Create mocks
	metricsMock := mocks.NewMetricsClient(t)
	redisMock := mocks.NewRedisClient(t)

	// Create services container
	svc := &service.Services{
		Metrics: metricsMock,
		Redis:   redisMock,
	}

	errorHandlerMock := mocks.InternalErrorHandler{}
	errorHandlerMock.On("HandleError", mock.Anything).Return()

	workerSvc := NewWorkerService(svc, 2, func(model.Job) (string, error) { return "job output", nil }, &errorHandlerMock)

	job := model.Job{
		InputFilePath:  "some/path/to/input.mp4",
		OutputFilePath: "some/path/to/output.mkv",
	}

	jobBytes, _ := json.Marshal(job)
	redisMock.On("DequeueJob", mock.Anything).Return("", errors.New("failed dequeue")).Once()
	redisMock.On("DequeueJob", mock.Anything).Return(string(jobBytes), nil).Once()

	var unmarshaledJob model.Job
	json.Unmarshal(jobBytes, &unmarshaledJob)

	result, _ := json.Marshal(model.JobResult{Job: unmarshaledJob, Output: "job output", Error: nil})

	redisMock.On("EnqueueJobResult", mock.Anything, mock.Anything).Return(nil)

	ctx, _ := context.WithTimeout(context.TODO(), 3*time.Second)
	redisMock.On("DequeueJob", ctx).Return("", errors.New("cancelled")).Run(func(args mock.Arguments) { <-ctx.Done() })

	workerSvc.Start(ctx)
	workerSvc.Wait()

	redisMock.AssertCalled(t, "EnqueueJobResult", mock.Anything, string(result))
	errorHandlerMock.AssertCalled(t, "HandleError", JobError{"", errors.New("failed dequeue")})
}
