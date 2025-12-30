package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// LightningInvoice represents a Lightning payment invoice
type LightningInvoice struct {
	ChargeID   string `json:"charge_id"`   // OpenNode charge ID (primary key)
	Username   string `json:"username"`    // User who created this invoice
	AmountSats int64  `json:"amount_sats"` // Amount in satoshis
	Credits    int64  `json:"credits"`     // Credits to grant when paid
	Status     string `json:"status"`      // pending, paid, expired
	Invoice    string `json:"invoice"`     // Lightning invoice (lnbc...)
	ExpiresAt  int64  `json:"expires_at"`  // Unix timestamp
	PaidAt     int64  `json:"paid_at"`     // Unix timestamp (0 if not paid)
	CreatedAt  int64  `json:"created_at"`  // Unix timestamp
	Processed  bool   `json:"processed"`   // Set to true after credits added
}

// CreditTransaction represents a credit transaction (purchase or usage)
type CreditTransaction struct {
	ID        int64  `json:"id"`         // Auto-increment ID
	Username  string `json:"username"`   // User
	Amount    int64  `json:"amount"`     // +N for purchase, -1 for usage
	Type      string `json:"type"`       // purchase, usage, admin_grant
	ChargeID  string `json:"charge_id"`  // OpenNode charge ID (if purchase)
	Timestamp int64  `json:"timestamp"`  // Unix timestamp
}

// LightningDB handles Lightning-related persistence
type LightningDB struct {
	mu           sync.RWMutex
	invoices     map[string]*LightningInvoice // chargeID -> invoice
	transactions []CreditTransaction
	invoicePath  string
	txPath       string
	nextTxID     int64
}

// NewLightningDB creates a new Lightning database
func NewLightningDB(dataDir string) (*LightningDB, error) {
	lightningDir := filepath.Join(dataDir, "lightning")
	if err := os.MkdirAll(lightningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lightning directory: %w", err)
	}

	db := &LightningDB{
		invoices:     make(map[string]*LightningInvoice),
		transactions: make([]CreditTransaction, 0),
		invoicePath:  filepath.Join(lightningDir, "invoices.json"),
		txPath:       filepath.Join(lightningDir, "transactions.json"),
		nextTxID:     1,
	}

	// Load existing data
	if err := db.LoadInvoices(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load invoices: %w", err)
	}
	if err := db.LoadTransactions(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load transactions: %w", err)
	}

	// Set nextTxID to max ID + 1
	for _, tx := range db.transactions {
		if tx.ID >= db.nextTxID {
			db.nextTxID = tx.ID + 1
		}
	}

	return db, nil
}

// LoadInvoices loads invoices from disk
func (db *LightningDB) LoadInvoices() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := ioutil.ReadFile(db.invoicePath)
	if err != nil {
		return err
	}

	var invoices []LightningInvoice
	if err := json.Unmarshal(data, &invoices); err != nil {
		return fmt.Errorf("failed to parse invoices: %w", err)
	}

	db.invoices = make(map[string]*LightningInvoice)
	for i := range invoices {
		db.invoices[invoices[i].ChargeID] = &invoices[i]
	}

	return nil
}

// SaveInvoices saves invoices to disk
func (db *LightningDB) SaveInvoices() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	invoices := make([]LightningInvoice, 0, len(db.invoices))
	for _, inv := range db.invoices {
		invoices = append(invoices, *inv)
	}

	data, err := json.MarshalIndent(invoices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal invoices: %w", err)
	}

	if err := ioutil.WriteFile(db.invoicePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write invoices: %w", err)
	}

	return nil
}

// LoadTransactions loads transactions from disk
func (db *LightningDB) LoadTransactions() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := ioutil.ReadFile(db.txPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &db.transactions); err != nil {
		return fmt.Errorf("failed to parse transactions: %w", err)
	}

	return nil
}

// SaveTransactions saves transactions to disk
func (db *LightningDB) SaveTransactions() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	data, err := json.MarshalIndent(db.transactions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transactions: %w", err)
	}

	if err := ioutil.WriteFile(db.txPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write transactions: %w", err)
	}

	return nil
}

// CreateInvoice stores a new invoice
func (db *LightningDB) CreateInvoice(inv *LightningInvoice) error {
	db.mu.Lock()
	db.invoices[inv.ChargeID] = inv
	db.mu.Unlock()

	return db.SaveInvoices()
}

// GetInvoice retrieves an invoice by charge ID
func (db *LightningDB) GetInvoice(chargeID string) (*LightningInvoice, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	inv, exists := db.invoices[chargeID]
	if !exists {
		return nil, fmt.Errorf("invoice not found: %s", chargeID)
	}

	return inv, nil
}

// UpdateInvoiceStatus updates invoice status and paid time
func (db *LightningDB) UpdateInvoiceStatus(chargeID string, status string, paidAt int64) error {
	db.mu.Lock()
	inv, exists := db.invoices[chargeID]
	if !exists {
		db.mu.Unlock()
		return fmt.Errorf("invoice not found: %s", chargeID)
	}

	inv.Status = status
	if paidAt > 0 {
		inv.PaidAt = paidAt
	}
	db.mu.Unlock()

	return db.SaveInvoices()
}

// MarkInvoiceProcessed marks invoice as processed (credits added)
func (db *LightningDB) MarkInvoiceProcessed(chargeID string) error {
	db.mu.Lock()
	inv, exists := db.invoices[chargeID]
	if !exists {
		db.mu.Unlock()
		return fmt.Errorf("invoice not found: %s", chargeID)
	}

	inv.Processed = true
	db.mu.Unlock()

	return db.SaveInvoices()
}

// AddTransaction records a credit transaction
func (db *LightningDB) AddTransaction(username string, amount int64, txType string, chargeID string, timestamp int64) error {
	db.mu.Lock()
	tx := CreditTransaction{
		ID:        db.nextTxID,
		Username:  username,
		Amount:    amount,
		Type:      txType,
		ChargeID:  chargeID,
		Timestamp: timestamp,
	}
	db.nextTxID++
	db.transactions = append(db.transactions, tx)
	db.mu.Unlock()

	return db.SaveTransactions()
}

// GetUserTransactions returns all transactions for a user
func (db *LightningDB) GetUserTransactions(username string) []CreditTransaction {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var userTx []CreditTransaction
	for _, tx := range db.transactions {
		if tx.Username == username {
			userTx = append(userTx, tx)
		}
	}

	return userTx
}

// GetUnprocessedPaidInvoices returns all paid but unprocessed invoices
func (db *LightningDB) GetUnprocessedPaidInvoices() []*LightningInvoice {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var unprocessed []*LightningInvoice
	for _, inv := range db.invoices {
		if inv.Status == "paid" && !inv.Processed {
			unprocessed = append(unprocessed, inv)
		}
	}

	return unprocessed
}
