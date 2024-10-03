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
		client := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("transferMoney-%s", uuid.New()),
			TaskQueue: "order-fulfill-nexus",
		}
		return client, nil
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

	w := worker.New(c, "order-fulfill-nexus", worker.Options{})
	s := nexus.NewService("payment-service")
	s.Register(TransferMoneyOperation)
	w.RegisterNexusService(s)

	// Register workflows
	w.RegisterWorkflow(helloworldmtls.OrderFulfillWorkflow)
	w.RegisterWorkflow(helloworldmtls.MoneyTransferWorkflow)

	// Register PaymentActivities
	paymentActivities := &helloworldmtls.PaymentActivities{}
	w.RegisterActivity(paymentActivities)

	// Register OrderActivities
	orderActivities := &helloworldmtls.OrderActivities{}
	w.RegisterActivity(orderActivities)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
