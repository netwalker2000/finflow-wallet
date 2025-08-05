// internal/service/wallet_service.go
package service

import (
	"context"
	"errors"
	"fmt"

	"finflow-wallet/internal/domain"
	"finflow-wallet/internal/repository"
	"finflow-wallet/internal/util"
	"finflow-wallet/pkg/db"

	"github.com/shopspring/decimal"
)

// WalletService defines the interface for wallet-related business logic.
type WalletService interface {
	Deposit(ctx context.Context, walletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Transaction, error)
	Withdraw(ctx context.Context, walletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Transaction, error)
	Transfer(ctx context.Context, fromWalletID, toWalletID int64, amount decimal.Decimal, currency string) (*domain.Wallet, *domain.Wallet, *domain.Transaction, error)
	GetBalance(ctx context.Context, walletID int64) (*domain.Wallet, error)
	GetTransactionHistory(ctx context.Context, walletID int64, limit, offset int) ([]domain.Transaction, int64, error)
	CreateUserAndWallet(ctx context.Context, username, currency string) (*domain.User, *domain.Wallet, error)
}

// walletService implements the WalletService interface.
type walletService struct {
	dbBeginner      db.DBTxBeginner       // For starting transactions (e.g., *sqlx.DB)
	dbExecutor      repository.DBExecutor // For non-transactional reads (e.g., *sqlx.DB)
	userRepo        repository.UserRepository
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	beginTx         db.BeginTxFunc    // Injected dependency for beginning transactions
	commitTx        db.CommitTxFunc   // Injected dependency for committing transactions
	rollbackTx      db.RollbackTxFunc // Injected dependency for rolling back transactions
}

// NewWalletService creates a new instance of WalletService.
func NewWalletService(
	dbBeginner db.DBTxBeginner,
	dbExecutor repository.DBExecutor,
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	beginTx db.BeginTxFunc,
	commitTx db.CommitTxFunc,
	rollbackTx db.RollbackTxFunc,
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

	txController, err := s.beginTx(ctx, s.dbBeginner) // Use injected function
	if err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to begin transaction: %w", err)
	}
	defer s.rollbackTx(txController) // Use injected function

	txExecutor, ok := txController.(repository.DBExecutor)
	if !ok {
		return nil, nil, fmt.Errorf("deposit: transaction controller does not implement DBExecutor")
	}

	wallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to get wallet %d: %w", walletID, err)
	}
	if wallet.Currency != currency {
		return nil, nil, util.ErrCurrencyMismatch
	}

	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, walletID, amount); err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to update wallet balance: %w", err)
	}

	transaction := domain.NewTransaction(nil, &walletID, amount, currency, domain.TransactionTypeDeposit, nil)
	if err := s.transactionRepo.CreateTransaction(ctx, txExecutor, transaction); err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to create transaction: %w", err)
	}

	updatedWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, walletID)
	if err != nil {
		return nil, nil, fmt.Errorf("deposit: failed to re-fetch updated wallet %d: %w", walletID, err)
	}

	if err := s.commitTx(txController); err != nil { // Use injected function
		return nil, nil, fmt.Errorf("deposit: failed to commit transaction: %w", err)
	}

	return updatedWallet, transaction, nil
}

// Withdraw, Transfer, GetBalance, GetTransactionHistory, CreateUserAndWallet methods
// (Adjust these similarly to Deposit, using s.beginTx, s.commitTx, s.rollbackTx, and passing s.dbBeginner or txExecutor to repos.
// For GetBalance and GetTransactionHistory, use s.dbExecutor for queries.)

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
		return nil, nil, util.ErrCurrencyMismatch
	}

	if wallet.Balance.LessThan(amount) {
		return nil, nil, util.ErrInsufficientFunds
	}

	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, walletID, amount.Neg()); err != nil {
		return nil, nil, fmt.Errorf("withdraw: failed to update wallet balance: %w", err)
	}

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
		return nil, nil, nil, util.ErrCurrencyMismatch
	}

	toWallet, err := s.walletRepo.GetWalletByID(ctx, txExecutor, toWalletID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to get destination wallet %d: %w", toWalletID, err)
	}
	if toWallet.Currency != currency {
		return nil, nil, nil, util.ErrCurrencyMismatch
	}

	if fromWallet.Balance.LessThan(amount) {
		return nil, nil, nil, util.ErrInsufficientFunds
	}

	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, fromWalletID, amount.Neg()); err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to update source wallet balance: %w", err)
	}

	if err := s.walletRepo.UpdateWalletBalance(ctx, txExecutor, toWalletID, amount); err != nil {
		return nil, nil, nil, fmt.Errorf("transfer: failed to update destination wallet balance: %w", err)
	}

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

func (s *walletService) GetBalance(ctx context.Context, walletID int64) (*domain.Wallet, error) {
	// For read-only operations outside a transaction, use s.dbExecutor
	wallet, err := s.walletRepo.GetWalletByID(ctx, s.dbExecutor, walletID)
	if err != nil {
		return nil, fmt.Errorf("get balance: failed to get wallet %d: %w", walletID, err)
	}
	return wallet, nil
}

// GetTransactionHistory retrieves a paginated list of transactions for a specific wallet.
func (s *walletService) GetTransactionHistory(ctx context.Context, walletID int64, limit, offset int) ([]domain.Transaction, int64, error) {
	// First, check if the wallet exists
	_, err := s.walletRepo.GetWalletByID(ctx, s.dbExecutor, walletID)
	if err != nil {
		if util.IsError(err, util.ErrNotFound) {
			return nil, 0, util.ErrWalletNotFound
		}
		return nil, 0, fmt.Errorf("failed to check wallet existence: %w", err)
	}

	// Call repository to get transactions and total count
	transactions, totalCount, err := s.transactionRepo.GetTransactionsByWalletID(ctx, s.dbExecutor, walletID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve transaction history: %w", err)
	}

	return transactions, totalCount, nil
}

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

	_, err = s.userRepo.GetUserByUsername(ctx, txExecutor, username)
	if err == nil {
		return nil, nil, fmt.Errorf("create user and wallet: user with username '%s' already exists", username)
	}
	if !errors.Is(err, util.ErrNotFound) {
		return nil, nil, fmt.Errorf("create user and wallet: failed to check existing user: %w", err)
	}

	user := domain.NewUser(username)
	if err := s.userRepo.CreateUser(ctx, txExecutor, user); err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to create user: %w", err)
	}

	wallet := domain.NewWallet(user.ID, currency)
	if err := s.walletRepo.CreateWallet(ctx, txExecutor, wallet); err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to create wallet: %w", err)
	}

	if err := s.commitTx(txController); err != nil {
		return nil, nil, fmt.Errorf("create user and wallet: failed to commit transaction: %w", err)
	}

	return user, wallet, nil
}
