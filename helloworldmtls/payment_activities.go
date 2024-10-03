package helloworldmtls

import (
	"context"
	"errors"
)

type PaymentActivities struct{}

func (a *PaymentActivities) Validate(ctx context.Context, input TransferInput) (bool, error) {
	if input.Amount <= 0 {
		return false, errors.New("amount must be greater than zero")
	}
	return true, nil
}

func (a *PaymentActivities) Withdraw(ctx context.Context, idempotencyKey string, amount float64, accountID string) (string, error) {
	if amount <= 0 {
		return "", errors.New("amount must be greater than zero")
	}
	return "withdrawal successful", nil
}

func (a *PaymentActivities) Deposit(ctx context.Context, idempotencyKey string, amount float64, accountID string) (string, error) {
	if amount <= 0 {
		return "", errors.New("amount must be greater than zero")
	}
	return "deposit successful", nil
}

func (a *PaymentActivities) SendNotification(ctx context.Context, input TransferInput) (bool, error) {
	// Implement notification logic
	return true, nil
}
