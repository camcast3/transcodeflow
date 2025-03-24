package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/internal/telemetry"

	"go.uber.org/zap"
)

var wg sync.WaitGroup

const MAX_PARALLELIZATION = 4 //default this until config implemented

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
		wg.Add(1)
		go w.getJobs(ctx, i)
	}

	go func() {
		for err := range w.errorChannel {
			telemetry.Logger.Error(fmt.Sprintf("Error: %e", err))
		}
	}()
	go func() {
		wg.Wait()
		close(w.errorChannel)
	}()

	wg.Wait()
	return nil
}

func (w *WorkerService) getJobs(ctx context.Context, id int) {
	for {
		jobStr, err := w.Services.Redis.DequeueJob(ctx)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			wg.Done()
			return
		}
		telemetry.Logger.Info("Dequeued job", zap.Any("worker_ID", id))

		var job model.Job
		err = json.Unmarshal([]byte(jobStr), &job)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			wg.Done()
			return
		}

		output, err := w.doTranscode(job)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			wg.Done()
			return
		}
		telemetry.Logger.Info("Finished job", zap.Any("worker_ID", id))

		err = w.pushResult(ctx, job, output, err)
		if err != nil {
			w.errorChannel <- JobError{jobStr, err}
			wg.Done()
			return
		}
		telemetry.Logger.Info("Pushed job result", zap.Any("worker_ID", id))

		select {
		case <-ctx.Done():
			wg.Done()
			return
		default:
		}
	}

}

func (w *WorkerService) pushResult(ctx context.Context, completedJob model.Job, stdout string, err error) error {
	//panic("pushResult not implemented")

	resultBytes, err := json.Marshal(model.JobResult{Job: completedJob, Output: stdout, Error: err})
	if err != nil {
		return err
	}
	resultString := string(resultBytes)
	err = w.Services.Redis.EnqueueJobResult(ctx, resultString)
	if err != nil {
		return err
	}

	telemetry.Logger.Info("Pushed job result", zap.Any("job_string", completedJob))
	return nil
}

func (w *WorkerService) doTranscode(job model.Job) (string, error) {
	//panic("DoTranscode not implemented")
	//Temporarily just print stuff for testing

	args := job.GetFFmpegCommand()
	cmd := exec.Command("echo", args...)
	stdout, err := cmd.Output()
	if err != nil {
		return string(stdout), err
	}

	fmt.Println(string(stdout))
	return string(stdout), nil
}
