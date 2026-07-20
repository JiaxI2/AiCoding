// Package gateref defines immutable references to validationevidence receipts.
package gateref

// GateRef points at evidence owned by internal/validationevidence. It deliberately
// contains no receipt payload and has no dependency on that package.
type GateRef struct {
	Profile            string `json:"profile"`
	ValidationIdentity string `json:"validationIdentity"`
	ReceiptID          string `json:"receiptID"`
}
