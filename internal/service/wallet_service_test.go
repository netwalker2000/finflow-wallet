// internal/service/wallet_service_test.go
package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"finflow-wallet/internal/domain"
	"finflow-wallet/internal/repository"
	"finflow-wallet/internal/util"
	"finflow-wallet/pkg/db" // Import pkg/db for interfaces and function types

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBExecutor is a mock implementation of repository.DBExecutor.
type MockDBExecutor struct {
	mock.Mock
}

func (m *MockDBExecutor) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	argsCalled := m.Called(ctx, dest, query, args)
	return argsCalled.Error(0)
}

func (m *MockDBExecutor) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	argsCalled := m.Called(ctx, dest, query, args)
	return argsCalled.Error(0)
}

func (m *MockDBExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	argsCalled := m.Called(ctx, query, args)
	return argsCalled.Get(0).(sql.Result), argsCalled.Error(1)
}

func (m *MockDBExecutor) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	m.Called(ctx, query, args)
	return &sql.Row{}
}

// MockUserRepository is a mock implementation of repository.UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, q repository.DBExecutor, user *domain.User) error {
	args := m.Called(ctx, q, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, q repository.DBExecutor, id int64) (*domain.User, error) {
	args := m.Called(ctx, q, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, q repository.DBExecutor, username string) (*domain.User, error) {
	args := m.Called(ctx, q, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockWalletRepository is a mock implementation of repository.WalletRepository.
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) CreateWallet(ctx context.Context, q repository.DBExecutor, wallet *domain.Wallet) error {
	args := m.Called(ctx, q, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetWalletByID(ctx context.Context, q repository.DBExecutor, id int64) (*domain.Wallet, error) {
	args := m.Called(ctx, q, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) GetWalletByUserIDAndCurrency(ctx context.Context, q repository.DBExecutor, userID int64, currency string) (*domain.Wallet, error) {
	args := m.Called(ctx, q, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateWalletBalance(ctx context.Context, q repository.DBExecutor, walletID int64, amount decimal.Decimal) error {
	args := m.Called(ctx, q, walletID, amount)
	return args.Error(0)
}

// MockTransactionRepository is a mock implementation of repository.TransactionRepository.
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) CreateTransaction(ctx context.Context, q repository.DBExecutor, transaction *domain.Transaction) error {
	args := m.Called(ctx, q, transaction)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTransactionsByWalletID(ctx context.Context, q repository.DBExecutor, walletID int64, limit, offset int) ([]domain.Transaction, error) {
	args := m.Called(ctx, q, walletID, limit, offset)
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

// MockDBBeginner is a mock implementation of db.DBTxBeginner.
type MockDBBeginner struct {
	mock.Mock
}

func (m *MockDBBeginner) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	args := m.Called(ctx, opts)
	return &sqlx.Tx{}, args.Error(1)
}

// MockTxController is a mock implementation of db.TxController.
// It also implicitly implements repository.DBExecutor for testing purposes
// by embedding MockDBExecutor.
type MockTxController struct {
	mock.Mock
	MockDBExecutor // Embed MockDBExecutor to satisfy repository.DBExecutor interface
}

func (m *MockTxController) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTxController) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

// TestDeposit tests the Deposit method of WalletService.
func TestDeposit(t *testing.T) {
	walletID := int64(1)
	amount := decimal.NewFromFloat(100.00)
	currency := "USD"

	// Test Case 1: Successful Deposit
	t.Run("SuccessfulDeposit", func(t *testing.T) {
		// Create mocks and service instance INSIDE the t.Run block
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor)
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor,
			mockUserRepo,
			mockWalletRepo,
			mockTransactionRepo,
			func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
				return mockTxController, nil
			},
			func(tx db.TxController) error {
				return mockTxController.Commit()
			},
			func(tx db.TxController) {
				_ = mockTxController.Rollback()
			},
		)

		initialWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		expectedNewBalance := initialWallet.Balance.Add(amount)
		updatedWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  expectedNewBalance,
		}

		// Set expectations for this specific test case
		mockTxController.On("Commit").Return(nil).Once()
		mockTxController.On("Rollback").Return(nil).Maybe() // Rollback might be called if Commit fails or defer runs after Commit.

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(updatedWallet, nil).Once() // Re-fetch updated wallet

		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.NoError(t, err)
		assert.NotNil(t, resWallet)
		assert.NotNil(t, resTx)
		assert.Equal(t, expectedNewBalance, resWallet.Balance)
		assert.Equal(t, domain.TransactionTypeDeposit, resTx.Type)
		assert.Equal(t, amount, resTx.Amount)

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 2: Invalid Amount
	t.Run("InvalidAmount", func(t *testing.T) {
		// Create mocks and service instance INSIDE the t.Run block
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor)
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor,
			mockUserRepo,
			mockWalletRepo,
			mockTransactionRepo,
			func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
				return mockTxController, nil
			},
			func(tx db.TxController) error {
				return mockTxController.Commit()
			},
			func(tx db.TxController) {
				_ = mockTxController.Rollback()
			},
		)

		invalidAmount := decimal.NewFromFloat(-10.00)
		resWallet, resTx, err := service.Deposit(ctx, walletID, invalidAmount, currency)

		assert.ErrorIs(t, err, util.ErrInvalidInput)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		// Ensure no transaction was begun (because it's an early return due to invalid input)
		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 3: Wallet Not Found
	t.Run("WalletNotFound", func(t *testing.T) {
		// Create mocks and service instance INSIDE the t.Run block
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor)
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor,
			mockUserRepo,
			mockWalletRepo,
			mockTransactionRepo,
			func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
				return mockTxController, nil // Simulates successful beginTx
			},
			func(tx db.TxController) error {
				return mockTxController.Commit()
			},
			func(tx db.TxController) {
				_ = mockTxController.Rollback()
			},
		)

		// Set expectations for this specific test case
		// A transaction begins, then GetWalletByID fails, so Rollback is called. Commit is NOT called.
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(nil, util.ErrNotFound).Once()
		mockTxController.On("Rollback").Return(nil).Once() // Expect rollback to return nil

		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit") // Ensure Commit was not called

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 4: Currency Mismatch
	t.Run("CurrencyMismatch", func(t *testing.T) {
		// Create mocks and service instance INSIDE the t.Run block
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor)
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor,
			mockUserRepo,
			mockWalletRepo,
			mockTransactionRepo,
			func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
				return mockTxController, nil // Simulates successful beginTx
			},
			func(tx db.TxController) error {
				return mockTxController.Commit()
			},
			func(tx db.TxController) {
				_ = mockTxController.Rollback()
			},
		)

		initialWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: "EUR", // Mismatch
			Balance:  decimal.NewFromFloat(500.00),
		}

		// Set expectations for this specific test case
		// A transaction begins, then currency mismatch occurs, so Rollback is called. Commit is NOT called.
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockTxController.On("Rollback").Return(nil).Once() // Expect rollback to return nil

		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency mismatch")
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit") // Ensure Commit was not called

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 5: Update Balance Error
	t.Run("UpdateBalanceError", func(t *testing.T) {
		// Create mocks and service instance INSIDE the t.Run block
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor)
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor,
			mockUserRepo,
			mockWalletRepo,
			mockTransactionRepo,
			func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
				return mockTxController, nil // Simulates successful beginTx
			},
			func(tx db.TxController) error {
				return mockTxController.Commit()
			},
			func(tx db.TxController) {
				_ = mockTxController.Rollback()
			},
		)

		initialWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}

		// Set expectations for this specific test case
		// A transaction begins, then UpdateWalletBalance fails, so Rollback is called. Commit is NOT called.
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount).Return(errors.New("db error")).Once()
		mockTxController.On("Rollback").Return(nil).Once() // Expect rollback to return nil

		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update wallet balance")
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit") // Ensure Commit was not called

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})
}
