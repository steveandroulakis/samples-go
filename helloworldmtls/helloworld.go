package helloworldmtls

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Define the Order type
type Order struct {
	ID          string
	TotalAmount float64
}

// OrderFulfillWorkflow orchestrates the order fulfillment process.
func OrderFulfillWorkflow(ctx workflow.Context, order Order) (string, error) {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Nexus Client - Connect to the Nexus endpoint
	paymentClient := workflow.NewNexusClient("stevea-nexus-endpoint", "payment-service")
	transferInput := TransferInput{
		Amount: order.TotalAmount,
	}

	// Execute MoneyTransferWorkflow using Nexus
	handle := paymentClient.ExecuteOperation(ctx, "transferMoney", transferInput, workflow.NexusOperationOptions{})

	var transferResult TransferOutput
	if err := handle.Get(ctx, &transferResult); err != nil {
		return "", err
	}

	// Reserve Inventory
	err := workflow.ExecuteActivity(ctx, (*OrderActivities).ReserveInventory, order).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	// Deliver Order
	err = workflow.ExecuteActivity(ctx, (*OrderActivities).DeliverOrder, order).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	return "Order fulfilled", nil
}

// Input for MoneyTransferWorkflow
type TransferInput struct {
	Amount float64
}

type TransferOutput struct {
	Status  string
	Message string
}

func MoneyTransferWorkflow(ctx workflow.Context, input TransferInput) (*TransferOutput, error) {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Validate
	err := workflow.ExecuteActivity(ctx, (*PaymentActivities).Validate, input).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	// Withdraw
	var idempotencyKey string
	_ = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return uuid.New().String()
	}).Get(&idempotencyKey)

	err = workflow.ExecuteActivity(ctx, (*PaymentActivities).Withdraw, idempotencyKey, input.Amount, "source-account-id").Get(ctx, nil) // Replace with actual account details
	if err != nil {
		return nil, err
	}

	// Deposit
	depositResponse := ""
	err = workflow.ExecuteActivity(ctx, (*PaymentActivities).Deposit, idempotencyKey, input.Amount, "destination-account-id").Get(ctx, &depositResponse) // Replace with actual account details
	if err != nil {
		return nil, err
	}

	// Send Notification
	err = workflow.ExecuteActivity(ctx, (*PaymentActivities).SendNotification, input).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	output := &TransferOutput{
		Status:  "Success",
		Message: "Transfer completed successfully",
	}
	return output, nil
}

// ParseClientOptionFlags parses the given arguments into client options. In
// some cases a failure will be returned as an error, in others the process may
// exit with help info.
func ParseClientOptionFlags(args []string) (client.Options, error) {
	// Parse args
	set := flag.NewFlagSet("order-fulfill-nexus", flag.ExitOnError)
	targetHost := set.String("target-host", "localhost:7233", "Host:port for the server")
	namespace := set.String("namespace", "default", "Namespace for the server")
	serverRootCACert := set.String("server-root-ca-cert", "", "Optional path to root server CA cert")
	clientCert := set.String("client-cert", "", "Required path to client cert")
	clientKey := set.String("client-key", "", "Required path to client key")
	serverName := set.String("server-name", "", "Server name to use for verifying the server's certificate")
	insecureSkipVerify := set.Bool("insecure-skip-verify", false, "Skip verification of the server's certificate and host name")
	if err := set.Parse(args); err != nil {
		return client.Options{}, fmt.Errorf("failed parsing args: %w", err)
	} else if *clientCert == "" || *clientKey == "" {
		return client.Options{}, fmt.Errorf("-client-cert and -client-key are required")
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(*clientCert, *clientKey)
	if err != nil {
		return client.Options{}, fmt.Errorf("failed loading client cert and key: %w", err)
	}

	// Load server CA if given
	var serverCAPool *x509.CertPool
	if *serverRootCACert != "" {
		serverCAPool = x509.NewCertPool()
		b, err := os.ReadFile(*serverRootCACert)
		if err != nil {
			return client.Options{}, fmt.Errorf("failed reading server CA: %w", err)
		} else if !serverCAPool.AppendCertsFromPEM(b) {
			return client.Options{}, fmt.Errorf("server CA PEM file invalid")
		}
	}

	return client.Options{
		HostPort:  *targetHost,
		Namespace: *namespace,
		ConnectionOptions: client.ConnectionOptions{
			TLS: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				RootCAs:            serverCAPool,
				ServerName:         *serverName,
				InsecureSkipVerify: *insecureSkipVerify,
			},
		},
	}, nil
}
