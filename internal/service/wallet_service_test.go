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
	"finflow-wallet/pkg/db" // Import pkg/db for DBTxBeginner and TxController

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBExecutor is a mock implementation of repository.DBExecutor.
// This mock will be embedded in MockTxController and MockDBBeginner.
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
	return sql.Result(nil), argsCalled.Error(1)
}

func (m *MockDBExecutor) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// For QueryRowContext, you usually need to mock the Scan method of the returned *sql.Row.
	// This is complex. For unit tests, it's often simpler to mock the *repository method*
	// that uses QueryRowContext to return the expected domain object or error directly.
	// For now, we'll return nil.
	return nil
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

// MockTxController is a mock that implements db.TxController AND repository.DBExecutor.
// This is the mock for the *transaction* object itself.
type MockTxController struct {
	MockDBExecutor // Embed MockDBExecutor for query methods (GetContext, SelectContext, ExecContext, QueryRowContext)
	mock.Mock      // For Commit/Rollback specific mocks
}

// Commit implements db.TxController.
func (m *MockTxController) Commit() error {
	args := m.Called()
	return args.Error(0)
}

// Rollback implements db.TxController.
func (m *MockTxController) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

// MockDBBeginner is a mock that implements db.DBTxBeginner AND repository.DBExecutor.
// This is the mock for the main DB connection that starts transactions.
type MockDBBeginner struct {
	MockDBExecutor // Embed MockDBExecutor for non-transactional queries
	mock.Mock      // For BeginTxx specific mocks
}

// BeginTxx implements db.DBTxBeginner.
func (m *MockDBBeginner) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	args := m.Called(ctx, opts)
	// We must return a *sqlx.Tx, but it won't be used for Commit/Rollback directly by the service.
	// The service uses the injected BeginTxFunc which returns db.TxController.
	// This *sqlx.Tx is just a placeholder to satisfy the signature.
	return &sqlx.Tx{}, args.Error(1)
}

// TestDeposit tests the Deposit method of WalletService.
func TestDeposit(t *testing.T) {
	ctx := context.Background()
	mockDBBeginner := new(MockDBBeginner) // Mock for db.DBTxBeginner
	mockDBExecutor := new(MockDBExecutor) // Mock for repository.DBExecutor (for non-tx reads)
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTransactionRepo := new(MockTransactionRepository)

	// Mock functions for transaction management
	mockBeginTx := func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
		args := mockDBBeginner.Called(ctx, dbConn) // Call the mockDBBeginner's BeginTxx
		if err := args.Error(1); err != nil {
			return nil, err
		}
		// Return a MockTxController which implements db.TxController and repository.DBExecutor
		mockTx := new(MockTxController)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Once()
		return mockTx, nil
	}

	mockCommitTx := func(tx db.TxController) error {
		// We expect the Commit method on the mockTxController to be called
		return tx.Commit() // This will call the mocked Commit on MockTxController
	}

	mockRollbackTx := func(tx db.TxController) {
		// We expect the Rollback method on the mockTxController to be called
		tx.Rollback() // This will call the mocked Rollback on MockTxController
	}

	service := NewWalletService(
		mockDBBeginner,
		mockDBExecutor, // Pass a separate mock for non-transactional DBExecutor
		mockUserRepo,
		mockWalletRepo,
		mockTransactionRepo,
		mockBeginTx,
		mockCommitTx,
		mockRollbackTx,
	)

	walletID := int64(1)
	amount := decimal.NewFromFloat(100.00)
	currency := "USD"

	// Test Case 1: Successful Deposit
	t.Run("SuccessfulDeposit", func(t *testing.T) {
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

		// Mock BeginTxx on mockDBBeginner
		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()

		// Mock repository calls. The `mock.Anything` for the `DBExecutor` parameter
		// will match the `MockTxController` returned by `mockBeginTx`.
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(updatedWallet, nil).Once()

		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.NoError(t, err)
		assert.NotNil(t, resWallet)
		assert.NotNil(t, resTx)
		assert.Equal(t, expectedNewBalance, resWallet.Balance)
		assert.Equal(t, domain.TransactionTypeDeposit, resTx.Type)
		assert.Equal(t, amount, resTx.Amount)

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{} // Reset mocks for next test
		mockWalletRepo.Calls = []mock.Call{}
		mockTransactionRepo.Calls = []mock.Call{}
	})

	// Test Case 2: Invalid Amount
	t.Run("InvalidAmount", func(t *testing.T) {
		invalidAmount := decimal.NewFromFloat(-10.00)
		resWallet, resTx, err := service.Deposit(ctx, walletID, invalidAmount, currency)

		assert.ErrorIs(t, err, util.ErrInvalidInput)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything) // Should not begin transaction
	})

	// Test Case 3: Wallet Not Found
	t.Run("WalletNotFound", func(t *testing.T) {
		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(nil, util.ErrNotFound).Once()

		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockWalletRepo.Calls = []mock.Call{}
	})

	// Test Case 4: Currency Mismatch
	t.Run("CurrencyMismatch", func(t *testing.T) {
		initialWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: "EUR", // Mismatch
			Balance:  decimal.NewFromFloat(500.00),
		}
		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.Error(t, err) // Specific error message will be checked by the service layer
		assert.Contains(t, err.Error(), "currency mismatch")
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockWalletRepo.Calls = []mock.Call{}
	})

	// Test Case 5: Update Balance Error
	t.Run("UpdateBalanceError", func(t *testing.T) {
		initialWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(500.00),
		}
		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, walletID, amount).Return(errors.New("db error")).Once()
		resWallet, resTx, err := service.Deposit(ctx, walletID, amount, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update wallet balance")
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockWalletRepo.Calls = []mock.Call{}
	})
}

// TestWithdraw tests the Withdraw method of WalletService.
func TestWithdraw(t *testing.T) {
	ctx := context.Background()
	mockDBBeginner := new(MockDBBeginner)
	mockDBExecutor := new(MockDBExecutor)
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTransactionRepo := new(MockTransactionRepository)

	mockBeginTx := func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
		args := mockDBBeginner.Called(ctx, dbConn)
		if err := args.Error(1); err != nil {
			return nil, err
		}
		mockTx := new(MockTxController)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Once()
		return mockTx, nil
	}
	mockCommitTx := func(tx db.TxController) error { return tx.Commit() }
	mockRollbackTx := func(tx db.TxController) { tx.Rollback() }

	service := NewWalletService(
		mockDBBeginner,
		mockDBExecutor,
		mockUserRepo,
		mockWalletRepo,
		mockTransactionRepo,
		mockBeginTx,
		mockCommitTx,
		mockRollbackTx,
	)

	walletID := int64(1)
	amount := decimal.NewFromFloat(100.00)
	currency := "USD"

	// Test Case 1: Successful Withdrawal
	t.Run("SuccessfulWithdrawal", func(t *testing.T) {
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

		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
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

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockWalletRepo.Calls = []mock.Call{}
		mockTransactionRepo.Calls = []mock.Call{}
	})

	// Test Case 2: Insufficient Funds
	t.Run("InsufficientFunds", func(t *testing.T) {
		initialWallet := &domain.Wallet{
			ID:       walletID,
			UserID:   1,
			Currency: currency,
			Balance:  decimal.NewFromFloat(50.00), // Less than amount
		}

		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, walletID).Return(initialWallet, nil).Once()

		resWallet, resTx, err := service.Withdraw(ctx, walletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrInsufficientFunds)
		assert.Nil(t, resWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockWalletRepo.Calls = []mock.Call{}
	})

	// Add other test cases for Withdraw: Invalid Amount, Wallet Not Found, Currency Mismatch, Update Balance Error, etc.
}

// TestTransfer tests the Transfer method of WalletService.
func TestTransfer(t *testing.T) {
	ctx := context.Background()
	mockDBBeginner := new(MockDBBeginner)
	mockDBExecutor := new(MockDBExecutor)
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTransactionRepo := new(MockTransactionRepository)

	mockBeginTx := func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
		args := mockDBBeginner.Called(ctx, dbConn)
		if err := args.Error(1); err != nil {
			return nil, err
		}
		mockTx := new(MockTxController)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Once()
		return mockTx, nil
	}
	mockCommitTx := func(tx db.TxController) error { return tx.Commit() }
	mockRollbackTx := func(tx db.TxController) { tx.Rollback() }

	service := NewWalletService(
		mockDBBeginner,
		mockDBExecutor,
		mockUserRepo,
		mockWalletRepo,
		mockTransactionRepo,
		mockBeginTx,
		mockCommitTx,
		mockRollbackTx,
	)

	fromWalletID := int64(1)
	toWalletID := int64(2)
	amount := decimal.NewFromFloat(50.00)
	currency := "USD"

	// Test Case 1: Successful Transfer
	t.Run("SuccessfulTransfer", func(t *testing.T) {
		fromWalletInitial := &domain.Wallet{ID: fromWalletID, UserID: 1, Currency: currency, Balance: decimal.NewFromFloat(500.00)}
		toWalletInitial := &domain.Wallet{ID: toWalletID, UserID: 2, Currency: currency, Balance: decimal.NewFromFloat(100.00)}

		fromWalletExpectedBalance := fromWalletInitial.Balance.Sub(amount)
		toWalletExpectedBalance := toWalletInitial.Balance.Add(amount)

		fromWalletUpdated := &domain.Wallet{ID: fromWalletID, UserID: 1, Currency: currency, Balance: fromWalletExpectedBalance}
		toWalletUpdated := &domain.Wallet{ID: toWalletID, UserID: 2, Currency: currency, Balance: toWalletExpectedBalance}

		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(fromWalletInitial, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(toWalletInitial, nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, fromWalletID, amount.Neg()).Return(nil).Once()
		mockWalletRepo.On("UpdateWalletBalance", ctx, mock.Anything, toWalletID, amount).Return(nil).Once()
		mockTransactionRepo.On("CreateTransaction", ctx, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, fromWalletID).Return(fromWalletUpdated, nil).Once()
		mockWalletRepo.On("GetWalletByID", ctx, mock.Anything, toWalletID).Return(toWalletUpdated, nil).Once()

		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, toWalletID, amount, currency)

		assert.NoError(t, err)
		assert.NotNil(t, resFromWallet)
		assert.NotNil(t, resToWallet)
		assert.NotNil(t, resTx)
		assert.Equal(t, fromWalletExpectedBalance, resFromWallet.Balance)
		assert.Equal(t, toWalletExpectedBalance, resToWallet.Balance)
		assert.Equal(t, domain.TransactionTypeTransfer, resTx.Type)
		assert.Equal(t, amount, resTx.Amount)

		mockDBBeginner.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockTransactionRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{} // Reset mocks for next test
		mockWalletRepo.Calls = []mock.Call{}
		mockTransactionRepo.Calls = []mock.Call{}
	})

	// Test Case 2: Same Wallet Transfer
	t.Run("SameWalletTransfer", func(t *testing.T) {
		resFromWallet, resToWallet, resTx, err := service.Transfer(ctx, fromWalletID, fromWalletID, amount, currency)

		assert.ErrorIs(t, err, util.ErrSameWalletTransfer)
		assert.Nil(t, resFromWallet)
		assert.Nil(t, resToWallet)
		assert.Nil(t, resTx)

		mockDBBeginner.AssertNotCalled(t, "BeginTxx", mock.Anything, mock.Anything)
	})

	// Add other test cases for Transfer: Invalid Amount, Insufficient Funds, Source/Dest Wallet Not Found, Currency Mismatch, Update Balance Error, etc.
}

// TestGetBalance tests the GetBalance method of WalletService.
func TestGetBalance(t *testing.T) {
	ctx := context.Background()
	mockDBBeginner := new(MockDBBeginner)
	mockDBExecutor := new(MockDBExecutor) // This mock will be used for non-transactional reads
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTransactionRepo := new(MockTransactionRepository)

	mockBeginTx := func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
		args := mockDBBeginner.Called(ctx, dbConn)
		if err := args.Error(1); err != nil {
			return nil, err
		}
		mockTx := new(MockTxController)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Once()
		return mockTx, nil
	}
	mockCommitTx := func(tx db.TxController) error { return tx.Commit() }
	mockRollbackTx := func(tx db.TxController) { tx.Rollback() }

	service := NewWalletService(
		mockDBBeginner,
		mockDBExecutor, // Pass this mock for non-transactional calls
		mockUserRepo,
		mockWalletRepo,
		mockTransactionRepo,
		mockBeginTx,
		mockCommitTx,
		mockRollbackTx,
	)

	walletID := int64(1)
	expectedWallet := &domain.Wallet{
		ID:       walletID,
		UserID:   1,
		Currency: "USD",
		Balance:  decimal.NewFromFloat(750.00),
	}

	// Test Case 1: Successful Get Balance
	t.Run("SuccessfulGetBalance", func(t *testing.T) {
		// GetBalance uses s.dbExecutor. So we mock GetWalletByID on mockWalletRepo
		// with mockDBExecutor as the DBExecutor parameter.
		mockWalletRepo.On("GetWalletByID", ctx, mockDBExecutor, walletID).Return(expectedWallet, nil).Once()

		resWallet, err := service.GetBalance(ctx, walletID)

		assert.NoError(t, err)
		assert.NotNil(t, resWallet)
		assert.Equal(t, expectedWallet.Balance, resWallet.Balance)

		mockWalletRepo.AssertExpectations(t)
		mockDBExecutor.AssertExpectations(t) // Assert calls on mockDBExecutor
		mockWalletRepo.Calls = []mock.Call{}
		mockDBExecutor.Calls = []mock.Call{}
	})

	// Test Case 2: Wallet Not Found
	t.Run("WalletNotFound", func(t *testing.T) {
		mockWalletRepo.On("GetWalletByID", ctx, mockDBExecutor, walletID).Return(nil, util.ErrNotFound).Once()

		resWallet, err := service.GetBalance(ctx, walletID)

		assert.ErrorIs(t, err, util.ErrNotFound)
		assert.Nil(t, resWallet)

		mockWalletRepo.AssertExpectations(t)
		mockDBExecutor.AssertExpectations(t)
		mockWalletRepo.Calls = []mock.Call{}
		mockDBExecutor.Calls = []mock.Call{}
	})
}

// TestGetTransactionHistory tests the GetTransactionHistory method of WalletService.
func TestGetTransactionHistory(t *testing.T) {
	ctx := context.Background()
	mockDBBeginner := new(MockDBBeginner)
	mockDBExecutor := new(MockDBExecutor) // This mock will be used for non-transactional reads
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTransactionRepo := new(MockTransactionRepository)

	mockBeginTx := func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
		args := mockDBBeginner.Called(ctx, dbConn)
		if err := args.Error(1); err != nil {
			return nil, err
		}
		mockTx := new(MockTxController)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Once()
		return mockTx, nil
	}
	mockCommitTx := func(tx db.TxController) error { return tx.Commit() }
	mockRollbackTx := func(tx db.TxController) { tx.Rollback() }

	service := NewWalletService(
		mockDBBeginner,
		mockDBExecutor, // Pass this mock for non-transactional calls
		mockUserRepo,
		mockWalletRepo,
		mockTransactionRepo,
		mockBeginTx,
		mockCommitTx,
		mockRollbackTx,
	)

	walletID := int64(1)
	limit := 10
	offset := 0
	expectedTransactions := []domain.Transaction{
		{ID: 1, Amount: decimal.NewFromFloat(100), Type: domain.TransactionTypeDeposit},
		{ID: 2, Amount: decimal.NewFromFloat(50), Type: domain.TransactionTypeWithdrawal},
	}

	// Test Case 1: Successful Get Transaction History
	t.Run("SuccessfulGetTransactionHistory", func(t *testing.T) {
		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, limit, offset).Return(expectedTransactions, nil).Once()

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, limit, offset)

		assert.NoError(t, err)
		assert.NotNil(t, resTransactions)
		assert.Len(t, resTransactions, len(expectedTransactions))
		assert.Equal(t, expectedTransactions[0].ID, resTransactions[0].ID) // Corrected assertion

		mockTransactionRepo.AssertExpectations(t)
		mockDBExecutor.AssertExpectations(t)
		mockTransactionRepo.Calls = []mock.Call{}
		mockDBExecutor.Calls = []mock.Call{}
	})

	// Test Case 2: Repository Error
	t.Run("RepositoryError", func(t *testing.T) {
		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, limit, offset).Return([]domain.Transaction{}, errors.New("db error")).Once()

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, limit, offset)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get transactions")
		assert.Empty(t, resTransactions)

		mockTransactionRepo.AssertExpectations(t)
		mockDBExecutor.AssertExpectations(t)
		mockTransactionRepo.Calls = []mock.Call{}
		mockDBExecutor.Calls = []mock.Call{}
	})

	// Test Case 3: Default Limit/Offset
	t.Run("DefaultLimitOffset", func(t *testing.T) {
		mockTransactionRepo.On("GetTransactionsByWalletID", ctx, mockDBExecutor, walletID, 10, 0).Return(expectedTransactions, nil).Once() // Expect default values

		resTransactions, err := service.GetTransactionHistory(ctx, walletID, 0, -5) // Pass invalid limit/offset

		assert.NoError(t, err)
		assert.NotNil(t, resTransactions)

		mockTransactionRepo.AssertExpectations(t)
		mockDBExecutor.AssertExpectations(t)
		mockTransactionRepo.Calls = []mock.Call{}
		mockDBExecutor.Calls = []mock.Call{}
	})
}

// TestCreateUserAndWallet tests the CreateUserAndWallet method of WalletService.
func TestCreateUserAndWallet(t *testing.T) {
	ctx := context.Background()
	mockDBBeginner := new(MockDBBeginner)
	mockDBExecutor := new(MockDBExecutor)
	mockUserRepo := new(MockUserRepository)
	mockWalletRepo := new(MockWalletRepository)
	mockTransactionRepo := new(MockTransactionRepository)

	mockBeginTx := func(ctx context.Context, dbConn db.DBTxBeginner) (db.TxController, error) {
		args := mockDBBeginner.Called(ctx, dbConn)
		if err := args.Error(1); err != nil {
			return nil, err
		}
		mockTx := new(MockTxController)
		mockTx.On("Commit").Return(nil).Once()
		mockTx.On("Rollback").Return(nil).Once()
		return mockTx, nil
	}
	mockCommitTx := func(tx db.TxController) error { return tx.Commit() }
	mockRollbackTx := func(tx db.TxController) { tx.Rollback() }

	service := NewWalletService(
		mockDBBeginner,
		mockDBExecutor,
		mockUserRepo,
		mockWalletRepo,
		mockTransactionRepo,
		mockBeginTx,
		mockCommitTx,
		mockRollbackTx,
	)

	username := "testuser"
	currency := "USD"
	userID := int64(100)
	walletID := int64(200)

	// Test Case 1: Successful creation
	t.Run("SuccessfulCreation", func(t *testing.T) {
		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(&domain.User{}, util.ErrNotFound).Once() // User not found initially
		mockUserRepo.On("CreateUser", ctx, mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Run(func(args mock.Arguments) {
			user := args.Get(2).(*domain.User)
			user.ID = userID // Simulate DB assigning ID
		}).Once()
		mockWalletRepo.On("CreateWallet", ctx, mock.Anything, mock.AnythingOfType("*domain.Wallet")).Return(nil).Run(func(args mock.Arguments) {
			wallet := args.Get(2).(*domain.Wallet)
			wallet.ID = walletID // Simulate DB assigning ID
		}).Once()

		user, wallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, wallet)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, userID, wallet.UserID)
		assert.Equal(t, currency, wallet.Currency)
		assert.Equal(t, walletID, wallet.ID)

		mockDBBeginner.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockWalletRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockUserRepo.Calls = []mock.Call{}
		mockWalletRepo.Calls = []mock.Call{}
	})

	// Test Case 2: User already exists
	t.Run("UserAlreadyExists", func(t *testing.T) {
		existingUser := &domain.User{ID: userID, Username: username}
		mockDBBeginner.On("BeginTxx", ctx, mock.Anything).Return(&sqlx.Tx{}, nil).Once()
		mockUserRepo.On("GetUserByUsername", ctx, mock.Anything, username).Return(existingUser, nil).Once()

		user, wallet, err := service.CreateUserAndWallet(ctx, username, currency)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user with username 'testuser' already exists")
		assert.Nil(t, user)
		assert.Nil(t, wallet)

		mockDBBeginner.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockDBBeginner.Calls = []mock.Call{}
		mockUserRepo.Calls = []mock.Call{}
	})

	// Add other test cases for CreateUserAndWallet: CreateUser error, CreateWallet error, etc.
}
