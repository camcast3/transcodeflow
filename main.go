package main

import (
	"context"
	"log"
	"os/exec"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func main() {
	// Create Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Spawn multiple transcoding workers.
	for i := 0; i < 3; i++ {
		go transcodeWorker(rdb)
	}

	// Similarly start healthcheck and file replacement workers.
	// ...

	// Block forever.
	select {}
}

// transcodeWorker continuously polls for new transcoding jobs.
func transcodeWorker(rdb *redis.Client) {
	for {
		// BLPOP blocks until a job is available.
		result, err := rdb.BLPop(ctx, 0*time.Second, "transcode:jobs").Result()
		if err != nil {
			log.Printf("Error popping job: %v", err)
			continue
		}
		// result[0] is the key, result[1] is the payload.
		jobPayload := result[1]
		log.Printf("Received job: %s", jobPayload)

		// Parse the jobPayload (likely as JSON) to extract job details.
		// For demonstration, assume jobPayload has input, output, container, dryRun flag.

		// Call ffmpeg via exec.Command, or use any proper wrapper.
		cmd := exec.Command("ffmpeg", "-i", "input.mp4", "output_file.mkv")
		// Properly set up our command with parameters from the job.
		err = cmd.Run()
		if err != nil {
			log.Printf("Error transcoding file: %v", err)
			// Optionally, send the job to a retry queue.
			continue
		}
		log.Println("Transcoding completed")

		// Publish a new job for the healthcheck phase.
		err = rdb.RPush(ctx, "transcode:healthcheck", jobPayload).Err()
		if err != nil {
			log.Printf("Error publishing healthcheck job: %v", err)
		}
	}
}
