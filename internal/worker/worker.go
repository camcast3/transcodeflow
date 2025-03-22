package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/internal/telemetry"

	"go.uber.org/zap"
)

const MAX_PARALLELIZATION = 5 //default this until config implemented

type WorkerService struct {
	*service.Services
	errorChannel chan error
}

type JobError struct {
	JobString string
	error
}

func NewWorkerService(svc *service.Services) *WorkerService {
	return &WorkerService{svc, make(chan error, MAX_PARALLELIZATION)}
}

func (w *WorkerService) Start(ctx context.Context) error {
	for i := range MAX_PARALLELIZATION {
		go w.getJobs(ctx, i)
	}
	return nil
}

func (w *WorkerService) getJobs(ctx context.Context, id int) {
	for {
		jobStr, err := w.Services.Redis.DequeueJob(ctx)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			return
		}
		telemetry.Logger.Info("Dequeued job", zap.Any("worker_ID", id))

		err = w.doTranscode(jobStr)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			return
		}
		telemetry.Logger.Info("Finished job", zap.Any("worker_ID", id))

		err = w.pushResult(jobStr)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			return
		}
		telemetry.Logger.Info("Pushed job result", zap.Any("worker_ID", id))
	}

}

func (w *WorkerService) pushResult(completedJob string) error {
	//panic("pushResult not implemented")
	telemetry.Logger.Info("Mock pushed job ", zap.Any("job_string", completedJob))
	return nil
}

func (w *WorkerService) doTranscode(jobStr string) error {
	//panic("DoTranscode not implemented")
	//Temporarily just print stuff for testing

	var job *model.Job
	err := json.Unmarshal([]byte(jobStr), job)
	if err != nil {
		return err
	}
	args := job.GetFFmpegCommand()
	cmd := exec.Command("echo", args...)
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Println(stdout)
	return nil
}
