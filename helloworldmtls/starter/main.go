package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/temporalio/samples-go/helloworldmtls"
	"go.temporal.io/sdk/client"
)

func main() {
	// The client is a heavyweight object that should be created once per process.
	clientOptions, err := helloworldmtls.ParseClientOptionFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("Invalid arguments: %v", err)
	}
	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("orderFulfill-nexus-%s", uuid.New()),
		TaskQueue: "order-fulfill-nexus",
	}

	order := helloworldmtls.Order{
		ID:          "ABC123",
		TotalAmount: 999.99,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, helloworldmtls.OrderFulfillWorkflow, order)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// Synchronously wait for the workflow completion.
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable get workflow result", err)
	}
	log.Println("Workflow result:", result)
}
