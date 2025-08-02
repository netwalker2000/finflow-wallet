// internal/service/wallet_service.go
package service

import (
	"context"
	"errors"
	"fmt"

	"finflow-wallet/internal/domain"
	"finflow-wallet/internal/repository" // Import repository package
	"finflow-wallet/internal/util"
	"finflow-wallet/pkg/db" // For DBTxBeginner and TxController

	"github.com/shopspring/decimal"
)

// Define function types for transaction management using the new interfaces
type BeginTxFunc func(context.Context, db.DBTxBeginner) (db.TxController, error)
type CommitTxFunc func(db.TxController) error
type RollbackTxFunc func(db.TxController)

// WalletService defines the interface for wallet-related business logic.
type WalletService interface {
	// Deposit adds money to a user's wallet.
	Deposit(ctx context.Context, walletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Transaction, error)
	// Withdraw removes money from a user's wallet.
	Withdraw(ctx context.Context, walletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Transaction, error)
	// Transfer sends money from one user's wallet to another.
	Transfer(ctx context.Context, fromWalletID, toWalletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Wallet, *domain.Transaction, error)
	// GetBalance retrieves the current balance of a wallet.
	GetBalance(ctx context.Context, walletID int64) (*domain.Wallet, error)
	// GetTransactionHistory retrieves the transaction history for a wallet.
	GetTransactionHistory(ctx context.Context, walletID int64, limit, offset int) ([]domain.Transaction, error)
	// CreateUserAndWallet creates a new user and an associated wallet.
	CreateUserAndWallet(ctx context.Context, username, currency string) (*domain.User, *domain.Wallet, error)
}

// walletService implements the WalletService interface.
type walletService struct {
	dbBeginner      db.DBTxBeginner       // For starting transactions (e.g., *sqlx.DB)
	dbExecutor      repository.DBExecutor // For non-transactional reads (e.g., *sqlx.DB)
	userRepo        repository.UserRepository
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	beginTx         BeginTxFunc    // Injected dependency for beginning transactions
	commitTx        CommitTxFunc   // Injected dependency for committing transactions
	rollbackTx      RollbackTxFunc // Injected dependency for rolling back transactions
}

// NewWalletService creates a new instance of WalletService.
func NewWalletService(
	dbBeginner db.DBTxBeginner,
	dbExecutor repository.DBExecutor,
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	beginTx BeginTxFunc,
	commitTx CommitTxFunc,
	rollbackTx RollbackTxFunc,
) WalletService {
	return &walletService{
		dbBeginner:      dbBeginner,
		dbExecutor:      dbExecutor,
		userRepo:        userRepo,
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		beginTx:         beginTx,
		commitTx:        commitTx,
		rollbackTx:      rollbackTx,
	}
}

// Deposit adds money to a user's wallet.
func (s *walletService) Deposit(ctx context.Context, walletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Transaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, util.ErrInvalidInput
	}

	// Use the injected beginTx function and s.dbBeginner
	txController, err := s.beginTx(ctx, s.dbBeginner)
	if err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to begin transaction: %w", err)
	}
	defer s.rollbackTx(txController) // Use the injected rollbackTx function

	// Cast txController to repository.DBExecutor for repository methods
	txExecutor, ok := txController.(repository.DBExecutor)
	if !ok {
		return nil, nil, fmt.Errorf("deposit: transaction controller does not implement DBExecutor")
	}

	wallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to get wallet %d: %w", walletID, err)
	}
	if wallet.Currency != currency {
		return nil, nil, fmt.Errorf("deposit: currency mismatch, expected %s but got %s", wallet.Currency, currency)
	}

	// Update wallet balance
	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, walletID, amount); err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to update wallet balance: %w", err)
	}

	// Create transaction record
	transaction := domain.NewTransaction(nil, &walletID, amount, currency, domain.TransactionTypeDeposit, nil)
	if err := s.transactionRepo.CreateTransaction(ctx, txExecutor, transaction); err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to create transaction: %w", err)
	}

	// Re-fetch wallet to get updated balance (optional, can also calculate locally)
	updatedWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to re-fetch updated wallet %d: %w", walletID, err)
	}

	if err := s.commitTx(txController); err != nil { // Use the injected commitTx function
		return nil, nil, fmt.Errorf("deposit: failed to commit transaction: %w", err)
	}

	return updatedWallet, transaction, nil
}

// Withdraw removes money from a user's wallet.
func (s *walletService) Withdraw(ctx context.Context, walletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Transaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, util.ErrInvalidInput
	}

	txController, err := s.beginTx(ctx, s.dbBeginner)
	if err != nil {
		return nil, nil, fmt.Errorf("withdraw: failed to begin transaction: %w", err)
	}
	defer s.rollbackTx(txController)

	txExecutor, ok := txController.(repository.DBExecutor)
	if !ok {
		return nil, nil, fmt.Errorf("withdraw: transaction controller does not implement DBExecutor")
	}

	wallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("withdraw: failed to get wallet %d: %w", walletID, err)
	}
	if wallet.Currency != currency {
		return nil, nil, fmt.Errorf("withdraw: currency mismatch, expected %s but got %s", wallet.Currency, currency)
	}

	// Check for sufficient funds
	if wallet.Balance.LessThan(amount) {
		return nil, nil, util.ErrInsufficientFunds
	}

	// Update wallet balance (subtract amount)
	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, walletID, amount.Neg()); err != nil { // Use .Neg() to subtract
		return nil, nil, fmt.Errorf("withdraw: failed to update wallet balance: %w", err)
	}

	// Create transaction record
	transaction := domain.NewTransaction(&walletID, nil, amount, currency, domain.TransactionTypeWithdrawal, nil)
	if err := s.transactionRepo.CreateTransaction(ctx, txExecutor, transaction); err != nil {
		return nil, nil, fmt.Errorf("withdraw: failed to create transaction: %w", err)
	}

	updatedWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("withdraw: failed to re-fetch updated wallet %d: %w", walletID, err)
	}

	if err := s.commitTx(txController); err != nil {
		return nil, nil, fmt.Errorf("withdraw: failed to commit transaction: %w", err)
	}

	return updatedWallet, transaction, nil
}

// Transfer sends money from one user's wallet to another.
func (s *walletService) Transfer(ctx context.Context, fromWalletID, toWalletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Wallet, *domain.Transaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, nil, util.ErrInvalidInput
	}
	if fromWalletID == toWalletID {
		return nil, nil, nil, util.ErrSameWalletTransfer
	}

	txController, err := s.beginTx(ctx, s.dbBeginner)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to begin transaction: %w", err)
	}
	defer s.rollbackTx(txController)

	txExecutor, ok := txController.(repository.DBExecutor)
	if !ok {
		return nil, nil, nil, fmt.Errorf("transfer: transaction controller does not implement DBExecutor")
	}

	fromWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, fromWalletID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to get source wallet %d: %w", fromWalletID, err)
	}
	if fromWallet.Currency != currency {
		return nil, nil, nil, fmt.Errorf("transfer: source wallet currency mismatch, expected %s but got %s", fromWallet.Currency, currency)
	}

	toWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, toWalletID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to get destination wallet %d: %w", toWalletID, err)
	}
	if toWallet.Currency != currency {
		return nil, nil, nil, fmt.Errorf("transfer: destination wallet currency mismatch, expected %s but got %s", toWallet.Currency, currency)
	}

	// Check for sufficient funds in source wallet
	if fromWallet.Balance.LessThan(amount) {
		return nil, nil, nil, util.ErrInsufficientFunds
	}

	// Update source wallet balance (subtract amount)
	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, fromWalletID, amount.Neg()); err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to update source wallet balance: %w", err)
	}

	// Update destination wallet balance (add amount)
	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, toWalletID, amount); err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to update destination wallet balance: %w", err)
	}

	// Create transaction record
	transaction := domain.NewTransaction(&fromWalletID, &toWalletID, amount, currency, domain.TransactionTypeTransfer, nil)
	if err := s.transactionRepo.CreateTransaction(ctx, txExecutor, transaction); err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to create transaction: %w", err)
	}

	updatedFromWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, fromWalletID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to re-fetch updated source wallet %d: %w", fromWalletID, err)
	}
	updatedToWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, toWalletID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to re-fetch updated destination wallet %d: %w", toWalletID, err)
	}

	if err := s.commitTx(txController); err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to commit transaction: %w", err)
	}

	return updatedFromWallet, updatedToWallet, transaction, nil
}

// GetBalance retrieves the current balance of a wallet.
func (s *walletService) GetBalance(ctx context.Context, walletID int64) (*domain.Wallet, error) {
	// For read-only operations outside a transaction, use s.dbExecutor
	wallet, err := s.walletRepo.GetWalletByID(ctx, s.dbExecutor, walletID)
	if err != nil {
		return nil, fmt.Errorf("get balance: failed to get wallet %d: %w", walletID, err)
	}
	return wallet, nil
}

// GetTransactionHistory retrieves the transaction history for a wallet.
func (s *walletService) GetTransactionHistory(ctx context.Context, walletID int64, limit, offset int) ([]domain.Transaction, error) {
	if limit <= 0 || limit > 100 { // Enforce reasonable limits
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	// For read-only operations outside a transaction, use s.dbExecutor
	transactions, err := s.transactionRepo.GetTransactionsByWalletID(ctx, s.dbExecutor, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get transaction history: failed to get transactions for wallet %d: %w", walletID, err)
	}
	return transactions, nil
}

// CreateUserAndWallet creates a new user and an associated wallet.
func (s *walletService) CreateUserAndWallet(ctx context.Context, username, currency string) (*domain.User, *domain.Wallet, error) {
	txController, err := s.beginTx(ctx, s.dbBeginner)
	if err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to begin transaction: %w", err)
	}
	defer s.rollbackTx(txController)

	txExecutor, ok := txController.(repository.DBExecutor)
	if !ok {
		return nil, nil, fmt.Errorf("create user and wallet: transaction controller does not implement DBExecutor")
	}

	// Check if user already exists
	_, err = s.userRepo.GetUserByUsername(ctx, txExecutor, username)
	if err == nil {
		return nil, nil, fmt.Errorf("create user and wallet: user with username '%s' already exists", username)
	}
	if !errors.Is(err, util.ErrNotFound) {
		return nil, nil, fmt.Errorf("create user and wallet: failed to check existing user: %w", err)
	}

	// Create user
	user := domain.NewUser(username)
	if err := s.userRepo.CreateUser(ctx, txExecutor, user); err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to create user: %w", err)
	}

	// Create wallet for the new user
	wallet := domain.NewWallet(user.ID, currency)
	if err := s.walletRepo.CreateWallet(ctx, txExecutor, wallet); err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to create wallet: %w", err)
	}

	if err := s.commitTx(txController); err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to commit transaction: %w", err)
	}

	return user, wallet, nil
}
