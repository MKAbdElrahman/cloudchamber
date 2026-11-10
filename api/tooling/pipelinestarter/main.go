package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	pipelinewf "github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/workflow"
	temporalclient "github.com/MKAbdElrahman/cloudchamber/app/sdk/temporal"
)

func main() {
	hostPort := flag.String("host", "localhost:7233", "Temporal frontend host:port")
	namespace := flag.String("namespace", "default", "Temporal namespace")
	jobID := flag.String("job", fmt.Sprintf("job-%d", time.Now().Unix()), "Pipeline job ID")
	source := flag.String("source", "https://www.youtube.com/watch?v=ciRTdOIhNCc", "Source reference URL/path")
	flag.Parse()

	c, err := temporalclient.NewClient(temporalclient.Config{
		HostPort:  *hostPort,
		Namespace: *namespace,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial temporal: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	workflowID := fmt.Sprintf("cloudchamber-%s", *jobID)
	run, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: pipelinewf.TaskQueueWorkflow,
	}, pipelinewf.ProcessSourceWorkflow, pipelinewf.ProcessSourceInput{
		JobID:     *jobID,
		SourceRef: *source,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start workflow: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("started workflowID=%s runID=%s jobID=%s\n", run.GetID(), run.GetRunID(), *jobID)

	var result pipelinewf.ProcessSourceResult
	if err := run.Get(context.Background(), &result); err != nil {
		fmt.Fprintf(os.Stderr, "workflow result: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("completed jobID=%s audio=%s transcript=%s frames=%s document=%s\n",
		result.JobID, result.AudioRef, result.Transcript, result.KeyFramesRef, result.DocumentRef)
}
