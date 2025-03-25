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

// var wg sync.WaitGroup
type JobTask func(model.Job) (string, error)

type InternalErrorHandler interface {
	HandleError(err error)
}

type DefaultErrorHandler struct{}

func (eh *DefaultErrorHandler) HandleError(err error) {
	//For now just log errors we can't push a result for
	telemetry.Logger.Error(fmt.Sprintf("Error: %e", err))
}

type WorkerService struct {
	*service.Services
	errorChannel       chan error
	MaxParallelization int
	WorkFunc           JobTask
	sync.WaitGroup
	InternalErrorHandler
}

// placeholder until we're sure how we want to report the error
// for a specific job which may be malformed
type JobError struct {
	JobString string
	error
}

func NewWorkerService(svc *service.Services, maxParallelization int, workFunc JobTask, handler InternalErrorHandler) *WorkerService {
	if handler == nil {
		handler = &DefaultErrorHandler{}
	}

	return &WorkerService{
		svc,
		make(chan error, maxParallelization),
		maxParallelization,
		workFunc,
		sync.WaitGroup{},
		handler,
	}
}

func (w *WorkerService) Start(ctx context.Context) error {
	for i := range w.MaxParallelization {
		w.Add(1)
		go w.getJobs(ctx, i)
	}

	go func() {
		for err := range w.errorChannel {
			w.HandleError(err)
		}
	}()

	go func() {
		w.Wait()
		close(w.errorChannel)
	}()

	w.Wait()
	return nil
}

func (w *WorkerService) getJobs(ctx context.Context, id int) {
	//TODO restart failed worker goroutines?
	defer w.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			jobStr, err := w.Services.Redis.DequeueJob(ctx)
			if err != nil {
				w.errorChannel <- JobError{jobStr, err}
				return
			}
			telemetry.Logger.Info("Dequeued job", zap.Any("worker_ID", id))

			var job model.Job
			err = json.Unmarshal([]byte(jobStr), &job)
			if err != nil {
				w.errorChannel <- JobError{jobStr, err}
				return
			}

			output, err := w.WorkFunc(job)
			telemetry.Logger.Info("Finished job", zap.Any("worker_ID", id))

			err = w.pushResult(ctx, job, output, err)
			if err != nil {
				w.errorChannel <- JobError{jobStr, err}
				return
			}
			telemetry.Logger.Info("Pushed job result", zap.Any("worker_ID", id))
		}
	}
}

func (w *WorkerService) pushResult(ctx context.Context, completedJob model.Job, stdout string, err error) error {
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

func FakeDoTranscode(job model.Job) (string, error) {
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
