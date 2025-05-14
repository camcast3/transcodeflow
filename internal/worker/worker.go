package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
	"transcodeflow/internal/model"
	"transcodeflow/internal/service"
	"transcodeflow/internal/telemetry"

	"go.uber.org/zap"
)

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
	resultChannel      chan JobResult
	MaxParallelization int
	WorkFunc           JobTask
	InternalErrorHandler
}

// placeholder until we're sure how we want to report the outcome
// for a specific job which may be malformed
type JobResult struct {
	JobString string
	Err       error
}

func NewWorkerService(svc *service.Services, maxParallelization int, workFunc JobTask, handler InternalErrorHandler) *WorkerService {
	if handler == nil {
		handler = &DefaultErrorHandler{}
	}

	if workFunc == nil {
		workFunc = DoTranscode
	}

	return &WorkerService{
		svc,
		make(chan JobResult, maxParallelization),
		maxParallelization,
		workFunc,
		handler,
	}
}

func (w *WorkerService) Start(ctx context.Context) error {
	currentWorkers := 0
	workerId := 0 //just increment an int for now; better solution later if necessary
	for {
		select {
		case <-ctx.Done():
			//add more graceful cleanup here
			return ctx.Err()
		case result := <-w.resultChannel:
			currentWorkers--
			if result.Err != nil {
				w.HandleError(result.Err) //todo: handle decrementing maxParallel if we see the 'too many parallel' error
			}
		default:
			if currentWorkers < w.MaxParallelization {
				currentWorkers++
				//start new job
				go w.getJobs(ctx, workerId) //give distinct contexts later if necessary
				workerId++
			}
			time.Sleep(time.Second * 1) //make sleeptime configurable for test
		}
	}
}

func (w *WorkerService) getJobs(ctx context.Context, id int) {
	jobStr, err := w.Services.Redis.DequeueJob(ctx)
	if err != nil {
		w.resultChannel <- JobResult{jobStr, err}
		return
	}
	telemetry.Logger.Info("Dequeued job", zap.Any("worker_ID", id))

	var job model.Job
	err = json.Unmarshal([]byte(jobStr), &job)
	if err != nil {
		w.resultChannel <- JobResult{jobStr, err}
		return
	}

	output, err := w.WorkFunc(job)
	telemetry.Logger.Info("Finished job", zap.Any("worker_ID", id))

	err = w.pushResult(ctx, job, output, err)
	if err != nil {
		w.resultChannel <- JobResult{jobStr, err}
		return
	}
	telemetry.Logger.Info("Pushed job result", zap.Any("worker_ID", id))
	w.resultChannel <- JobResult{jobStr, nil}
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

func DoTranscode(job model.Job) (string, error) {
	args := job.GetFFmpegCommand()
	cmd := exec.Command("ffmpeg", args...)
	stdout, err := cmd.Output()
	if err != nil {
		return string(stdout), err
	}

	fmt.Println(string(stdout))
	return string(stdout), nil
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
