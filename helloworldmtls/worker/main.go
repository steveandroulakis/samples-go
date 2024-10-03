package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/temporalio/samples-go/helloworldmtls"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/worker"
)

var TransferMoneyOperation = temporalnexus.NewWorkflowRunOperation(
	"transferMoney",
	helloworldmtls.MoneyTransferWorkflow,
	func(ctx context.Context, input helloworldmtls.TransferInput, soo nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
		// Generate a unique ID for each workflow run
		// A common approach is to combine input parameters with a UUID
		workflowID := fmt.Sprintf("transferMoney-nexus-%s", uuid.New())
		return client.StartWorkflowOptions{
			ID: workflowID,
		}, nil
	},
)

func main() {
	// The client and worker are heavyweight objects that should be created once per process.
	clientOptions, err := helloworldmtls.ParseClientOptionFlags(os.Args[1:])
	if err != nil {
		log.Fatalf("Invalid arguments: %v", err)
	}
	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "hello-world-mtls", worker.Options{})

	s := nexus.NewService("payments")
	s.Register(TransferMoneyOperation)
	w.RegisterNexusService(s)

	w.RegisterWorkflow(helloworldmtls.OrderFulfillWorkflow)
	w.RegisterWorkflow(helloworldmtls.MoneyTransferWorkflow)
	w.RegisterActivity(helloworldmtls.OrderFulfillmentActivities)
	w.RegisterActivity(helloworldmtls.MoneyTransferActivities)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
