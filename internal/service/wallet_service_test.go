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

		assert.ErrorIs(t, err, util.ErrCurrencyMismatch)
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

// TestWithdraw tests the Withdraw method of WalletService.
func TestWithdraw(t *testing.T) {
	walletID := int64(1)
	amount := decimal.NewFromFloat(50.00)
	currency := "USD"

	// Test Case 1: Successful Withdrawal
	t.Run("SuccessfulWithdrawal", func(t *testing.T) {
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
		expectedNewBalance := initialWallet.Balance.Sub(amount)
		updatedWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  expectedNewBalance,
		}

		mockTxController.On("Commit").Return(nil).Once()
		mockTxController.On("Rollback").Return(nil).Maybe()

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount.Neg()).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(updatedWallet, nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.NoError(t, err)
		assert.NotNil(t, resWallet)
		assert.NotNil(t, resTx)
		assert.Equal(t, expectedNewBalance, resWallet.Balance)
		assert.Equal(t, domain.TransactionTypeWithdrawal, resTx.Type)
		assert.Equal(t, amount, resTx.Amount)

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 2: Invalid Amount
	t.Run("InvalidAmount", func(t *testing.T) {
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
		resWallet, resTx, err := service.Withdraw(ctx, walletID, invalidAmount, currency)

		assert.ErrorIs(t, err, util.ErrInvalidInput)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 3: Wallet Not Found
	t.Run("WalletNotFound", func(t *testing.T) {
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

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(nil, util.ErrNotFound).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 4: Currency Mismatch
	t.Run("CurrencyMismatch", func(t *testing.T) {
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
			Currency: "EUR", // Mismatch
			Balance:  decimal.NewFromFloat(500.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrCurrencyMismatch)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 5: Insufficient Funds
	t.Run("InsufficientFunds", func(t *testing.T) {
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
			Balance:  decimal.NewFromFloat(20.00), // Less than amount
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrInsufficientFunds)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockWalletRepo.AssertNotCalled(t, "UpdateWalletBalance")
		mockTransactionRepo.AssertNotCalled(t, "CreateTransaction")
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 6: Update Balance Error
	t.Run("UpdateBalanceError", func(t *testing.T) {
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

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount.Neg()).Return(errors.New("db error")).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update wallet balance")
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTransactionRepo.AssertNotCalled(t, "CreateTransaction")
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 7: Create Transaction Error
	t.Run("CreateTransactionError", func(t *testing.T) {
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

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount.Neg()).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(errors.New("db error")).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create transaction")
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})
}

// TestTransfer tests the Transfer method of WalletService.
func TestTransfer(t *testing.T) {
	fromWalletID := int64(1)
	toWalletID := int64(2)
	amount := decimal.NewFromFloat(50.00)
	currency := "USD"

	// Test Case 1: Successful Transfer
	t.Run("SuccessfulTransfer", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		initialToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: currency,
			Balance:  decimal.NewFromFloat(100.00),
		}
		expectedFromBalance := initialFromWallet.Balance.Sub(amount)
		expectedToBalance := initialToWallet.Balance.Add(amount)
		updatedFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  expectedFromBalance,
		}
		updatedToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: currency,
			Balance:  expectedToBalance,
		}

		mockTxController.On("Commit").Return(nil).Once()
		mockTxController.On("Rollback").Return(nil).Maybe()

		// First GetWalletByID for fromWallet, then for toWallet
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(initialToWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, fromWalletID, amount.Neg()).Return(nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, toWalletID, amount).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(updatedFromWallet, nil).Once() // Re-fetch
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(updatedToWallet, nil).Once()     // Re-fetch

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.NoError(t, err)
		assert.NotNil(t, resFromWallet)
		assert.NotNil(t, resToWallet)
		assert.NotNil(t, resTx)
		assert.Equal(t, expectedFromBalance, resFromWallet.Balance)
		assert.Equal(t, expectedToBalance, resToWallet.Balance)
		assert.Equal(t, domain.TransactionTypeTransfer, resTx.Type)
		assert.Equal(t, amount, resTx.Amount)

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 2: Invalid Amount
	t.Run("InvalidAmount", func(t *testing.T) {
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
		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, invalidAmount, currency)

		assert.ErrorIs(t, err, util.ErrInvalidInput)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 3: Same Wallet Transfer
	t.Run("SameWalletTransfer", func(t *testing.T) {
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

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, fromWalletID, amount, currency) // fromWalletID == toWalletID

		assert.ErrorIs(t, err, util.ErrSameWalletTransfer)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 4: From Wallet Not Found
	t.Run("FromWalletNotFound", func(t *testing.T) {
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

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(nil, util.ErrNotFound).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockWalletRepo.AssertNotCalled(t, "GetWalletByID", ctx, mock.Anything, toWalletID) // toWallet not fetched
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 5: To Wallet Not Found
	t.Run("ToWalletNotFound", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(nil, util.ErrNotFound).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 6: From Wallet Currency Mismatch
	t.Run("FromWalletCurrencyMismatch", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: "EUR", // Mismatch
			Balance:  decimal.NewFromFloat(500.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrCurrencyMismatch)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockWalletRepo.AssertNotCalled(t, "GetWalletByID", ctx, mock.Anything, toWalletID)
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 7: To Wallet Currency Mismatch
	t.Run("ToWalletCurrencyMismatch", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		initialToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: "EUR", // Mismatch
			Balance:  decimal.NewFromFloat(100.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(initialToWallet, nil).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrCurrencyMismatch)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 8: Insufficient Funds (From Wallet)
	t.Run("InsufficientFunds", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(20.00), // Less than amount
		}
		initialToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: currency,
			Balance:  decimal.NewFromFloat(100.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(initialToWallet, nil).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrInsufficientFunds)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockWalletRepo.AssertNotCalled(t, "UpdateWalletBalance")
		mockTransactionRepo.AssertNotCalled(t, "CreateTransaction")
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 9: Update From Wallet Balance Error
	t.Run("UpdateFromWalletBalanceError", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		initialToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: currency,
			Balance:  decimal.NewFromFloat(100.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(initialToWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, fromWalletID, amount.Neg()).Return(errors.New("db error")).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update source wallet balance")
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockWalletRepo.AssertNotCalled(t, "UpdateWalletBalance", ctx, mock.Anything, toWalletID, mock.Anything) // To wallet not updated
		mockTransactionRepo.AssertNotCalled(t, "CreateTransaction")
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 10: Update To Wallet Balance Error
	t.Run("UpdateToWalletBalanceError", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		initialToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: currency,
			Balance:  decimal.NewFromFloat(100.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(initialToWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, fromWalletID, amount.Neg()).Return(nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, toWalletID, amount).Return(errors.New("db error")).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update destination wallet balance")
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockTransactionRepo.AssertNotCalled(t, "CreateTransaction")
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 11: Create Transaction Error
	t.Run("CreateTransactionError", func(t *testing.T) {
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

		initialFromWallet := &domain.Wallet{
			ID:       fromWalletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		initialToWallet := &domain.Wallet{
			ID:       toWalletID,
			UserID:   2,
			Currency: currency,
			Balance:  decimal.NewFromFloat(100.00),
		}

		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(initialFromWallet, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(initialToWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, fromWalletID, amount.Neg()).Return(nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, toWalletID, amount).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(errors.New("db error")).Once()
		mockTxController.On("Rollback").Return(nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create transaction")
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})
}

// TestGetBalance tests the GetBalance method of WalletService.
func TestGetBalance(t *testing.T) {
	walletID := int64(1)
	currency := "USD"

	// Test Case 1: Successful GetBalance
	t.Run("SuccessfulGetBalance", func(t *testing.T) {
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor) // This is used for read-only operations
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor, // Pass mockDBExecutor here
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

		expectedWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(750.00),
		}

		// GetBalance uses s.dbExecutor directly, not a transaction
		mockWalletRepo.On("GetWalletByID", ctx, mockDBExecutor, walletID).Return(expectedWallet, nil).Once()

		resWallet, err := service.GetBalance(ctx, walletID)

		assert.NoError(t, err)
		assert.NotNil(t, resWallet)
		assert.Equal(t, expectedWallet, resWallet)

		// Assert that no transaction-related methods were called
		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 2: Wallet Not Found
	t.Run("WalletNotFound", func(t *testing.T) {
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

		mockWalletRepo.On("GetWalletByID", ctx, mockDBExecutor, walletID).Return(nil, util.ErrNotFound).Once()

		resWallet, err := service.GetBalance(ctx, walletID)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resWallet)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 3: Repository Error
	t.Run("RepositoryError", func(t *testing.T) {
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

		testError := errors.New("database connection lost")
		mockWalletRepo.On("GetWalletByID", ctx, mockDBExecutor, walletID).Return(nil, testError).Once()

		resWallet, err := service.GetBalance(ctx, walletID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), testError.Error())
		assert.Nil(t, resWallet)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})
}

// TestCreateUserAndWallet tests the CreateUserAndWallet method of WalletService.
func TestCreateUserAndWallet(t *testing.T) {
	username := "testuser"
	currency := "USD"

	// Test Case 1: Successful CreateUserAndWallet
	t.Run("SuccessfulCreateUserAndWallet", func(t *testing.T) {
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

		// Expect no user to be found initially
		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(nil, util.ErrNotFound).Once()

		// Expect user and wallet creation
		createdUser := &domain.User{ID: 1, Username: username}
		createdWallet := &domain.Wallet{ID: 101, UserID: createdUser.ID, Currency: currency, Balance: decimal.Zero}

		// Mock CreateUser and CreateWallet calls
		mockUserRepo.On("CreateUser", ctx, mock.Anything, mock.AnythingOfType("*domain.User")).Run(func(args mock.Arguments) {
			// Simulate setting ID on the passed user object
			userArg := args.Get(2).(*domain.User)
			userArg.ID = createdUser.ID
		}).Return(nil).Once()

		mockWalletRepo.On("CreateWallet", ctx, mock.Anything, mock.AnythingOfType("*domain.Wallet")).Run(func(args mock.Arguments) {
			// Simulate setting ID on the passed wallet object
			walletArg := args.Get(2).(*domain.Wallet)
			walletArg.ID = createdWallet.ID
		}).Return(nil).Once()

		// Expect transaction commit
		mockTxController.On("Commit").Return(nil).Once()
		mockTxController.On("Rollback").Return(nil).Maybe() // In case of unexpected rollback

		resUser, resWallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.NoError(t, err)
		assert.NotNil(t, resUser)
		assert.NotNil(t, resWallet)
		assert.Equal(t, createdUser.ID, resUser.ID)
		assert.Equal(t, createdUser.Username, resUser.Username)
		assert.Equal(t, createdWallet.ID, resWallet.ID)
		assert.Equal(t, createdWallet.UserID, resWallet.UserID)
		assert.Equal(t, createdWallet.Currency, resWallet.Currency)
		assert.True(t, createdWallet.Balance.Equal(decimal.Zero))

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 2: User Already Exists
	t.Run("UserAlreadyExists", func(t *testing.T) {
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

		existingUser := &domain.User{ID: 1, Username: username}
		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(existingUser, nil).Once() // User found
		mockTxController.On("Rollback").Return(nil).Once()                                                  // Expect rollback

		resUser, resWallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
		assert.Nil(t, resUser)
		assert.Nil(t, resWallet)

		mockUserRepo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything)
		mockWalletRepo.AssertNotCalled(t, "CreateWallet", mock.Anything, mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 3: Error Checking Existing User (not ErrNotFound)
	t.Run("ErrorCheckingExistingUser", func(t *testing.T) {
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

		testError := errors.New("db connection failed")
		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(nil, testError).Once() // Simulate a DB error
		mockTxController.On("Rollback").Return(nil).Once()                                               // Expect rollback

		resUser, resWallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check existing user")
		assert.Nil(t, resUser)
		assert.Nil(t, resWallet)

		mockUserRepo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything)
		mockWalletRepo.AssertNotCalled(t, "CreateWallet", mock.Anything, mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 4: Create User Error
	t.Run("CreateUserError", func(t *testing.T) {
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

		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(nil, util.ErrNotFound).Once()
		testError := errors.New("user repo save error")
		mockUserRepo.On("CreateUser", ctx, mock.Anything, mock.AnythingOfType("*domain.User")).Return(testError).Once()
		mockTxController.On("Rollback").Return(nil).Once() // Expect rollback

		resUser, resWallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user")
		assert.Nil(t, resUser)
		assert.Nil(t, resWallet)

		mockWalletRepo.AssertNotCalled(t, "CreateWallet", mock.Anything, mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 5: Create Wallet Error
	t.Run("CreateWalletError", func(t *testing.T) {
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

		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(nil, util.ErrNotFound).Once()
		mockUserRepo.On("CreateUser", ctx, mock.Anything, mock.AnythingOfType("*domain.User")).Run(func(args mock.Arguments) {
			userArg := args.Get(2).(*domain.User)
			userArg.ID = 1 // Simulate ID being set
		}).Return(nil).Once()
		testError := errors.New("wallet repo save error")
		mockWalletRepo.On("CreateWallet", ctx, mock.Anything, mock.AnythingOfType("*domain.Wallet")).Return(testError).Once()
		mockTxController.On("Rollback").Return(nil).Once() // Expect rollback

		resUser, resWallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create wallet")
		assert.Nil(t, resUser)
		assert.Nil(t, resWallet)

		mockTxController.AssertNotCalled(t, "Commit")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 6: Commit Error
	t.Run("CommitError", func(t *testing.T) {
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

		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(nil, util.ErrNotFound).Once()
		mockUserRepo.On("CreateUser", ctx, mock.Anything, mock.AnythingOfType("*domain.User")).Run(func(args mock.Arguments) {
			userArg := args.Get(2).(*domain.User)
			userArg.ID = 1 // Simulate ID being set
		}).Return(nil).Once()
		mockWalletRepo.On("CreateWallet", ctx, mock.Anything, mock.AnythingOfType("*domain.Wallet")).Run(func(args mock.Arguments) {
			walletArg := args.Get(2).(*domain.Wallet)
			walletArg.ID = 101 // Simulate ID being set
		}).Return(nil).Once()

		testError := errors.New("commit failed")
		mockTxController.On("Commit").Return(testError).Once()
		mockTxController.On("Rollback").Return(nil).Maybe() // Rollback might be called after commit fails

		resUser, resWallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
		assert.Nil(t, resUser)
		assert.Nil(t, resWallet)

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})
}

// TestGetTransactionHistory tests the GetTransactionHistory method of WalletService.
func TestGetTransactionHistory(t *testing.T) {
	walletID := int64(1)
	limit := 10
	offset := 0

	// Test Case 1: Successful GetTransactionHistory with results
	t.Run("SuccessfulGetTransactionHistoryWithResults", func(t *testing.T) {
		ctx := context.Background()
		mockUserRepo := new(MockUserRepository)
		mockWalletRepo := new(MockWalletRepository)
		mockTransactionRepo := new(MockTransactionRepository)
		mockDBBeginner := new(MockDBBeginner)
		mockDBExecutor := new(MockDBExecutor) // This is used for read-only operations
		mockTxController := new(MockTxController)

		service := NewWalletService(
			mockDBBeginner,
			mockDBExecutor, // Pass mockDBExecutor here
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

		// Corrected expectedTransactions definition
		expectedTransactions := []domain.Transaction{
			{
				ID:           1,
				FromWalletID: nil,       // Deposit has no from_wallet_id
				ToWalletID:   &walletID, // Deposit goes to wallet_id
				Type:         domain.TransactionTypeDeposit,
				Amount:       decimal.NewFromFloat(100),
				Currency:     "USD", // Assuming currency is "USD" for these transactions
			},
			{
				ID:           2,
				FromWalletID: &walletID, // Withdrawal comes from wallet_id
				ToWalletID:   nil,       // Withdrawal has no to_wallet_id
				Type:         domain.TransactionTypeWithdrawal,
				Amount:       decimal.NewFromFloat(50),
				Currency:     "USD", // Assuming currency is "USD" for these transactions
			},
		}

		// GetTransactionHistory uses s.dbExecutor directly, not a transaction
		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, limit, offset).Return(expectedTransactions, nil).Once()

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, limit, offset)

		assert.NoError(t, err)
		assert.NotNil(t, resTransactions)
		assert.Equal(t, expectedTransactions, resTransactions)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 2: Successful GetTransactionHistory with no results
	t.Run("SuccessfulGetTransactionHistoryNoResults", func(t *testing.T) {
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

		expectedTransactions := []domain.Transaction{} // Empty slice

		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, limit, offset).Return(expectedTransactions, nil).Once()

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, limit, offset)

		assert.NoError(t, err)
		assert.NotNil(t, resTransactions)
		assert.Empty(t, resTransactions)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 3: Repository Error
	t.Run("RepositoryError", func(t *testing.T) {
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

		testError := errors.New("network error")
		// FIX: Explicitly return a nil slice of the correct type
		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, limit, offset).Return([]domain.Transaction(nil), testError).Once()

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, limit, offset)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), testError.Error())
		assert.Nil(t, resTransactions) // resTransactions should be nil here if the repo returns nil slice and error

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})

	// Test Case 4: Invalid Limit/Offset (should use defaults)
	t.Run("InvalidLimitOffset", func(t *testing.T) {
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

		// Corrected expectedTransactions definition
		expectedTransactions := []domain.Transaction{
			{
				ID:           1,
				FromWalletID: nil,
				ToWalletID:   &walletID,
				Type:         domain.TransactionTypeDeposit,
				Amount:       decimal.NewFromFloat(100),
				Currency:     "USD",
			},
		}

		// Expect the default limit (10) and offset (0) to be used
		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, 10, 0).Return(expectedTransactions, nil).Once()

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, -5, -10) // Invalid limit/offset

		assert.NoError(t, err)
		assert.NotNil(t, resTransactions)
		assert.Equal(t, expectedTransactions, resTransactions)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
		mockTxController.AssertNotCalled(t, "Commit")
		mockTxController.AssertNotCalled(t, "Rollback")

		mock.AssertExpectationsForObjects(t, mockDBBeginner, mockDBExecutor, mockTxController, mockUserRepo, mockWalletRepo, mockTransactionRepo)
	})
}
