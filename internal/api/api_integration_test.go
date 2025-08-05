// internal/api/api_integration_test.go
package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	app "finflow-wallet/internal" // Corrected import path and alias
	"finflow-wallet/internal/domain"
	// Import util for error checking
)

// testApp is the global application instance for testing.
var testApp *app.Application

// testServer is the httptest server.
var testServer *httptest.Server

// TestMain is the special entry point for Go tests, executed once before all tests.
func TestMain(m *testing.M) {
	// 1. Set up environment variables (ensure DB_NAME points to the test database).
	// In a real CI/CD environment, these variables would be provided by the CI system.
	setupEnvVars()

	// 2. Initialize the application.
	testApp = app.NewApplication()
	if err := testApp.Initialize(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize test application: %v\n", err)
		os.Exit(1) // Exit tests if initialization fails
	}

	// 3. Start an httptest server to test the HTTP handling layer.
	testServer = httptest.NewServer(testApp.HTTPHandler)
	// Ensure the server is closed after all tests are run.
	defer testServer.Close()

	// 4. Run all tests.
	code := m.Run()

	// 5. Shut down application resources after tests (e.g., database connections).
	if err := testApp.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to shutdown test application: %v\n", err)
		os.Exit(1)
	}

	os.Exit(code)
}

// setupEnvVars helper function: sets or checks database environment variables required for testing.
func setupEnvVars() {
	// Ensure these environment variables point to your test database
	if os.Getenv("SERVER_PORT") == "" {
		os.Setenv("SERVER_PORT", "8080")
	}
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", "localhost")
	}
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", "5432")
	}
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", "user") // Replace with your PostgreSQL username
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "password") // Replace with your PostgreSQL password
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", "walletdb_test") // Ensure this is your test database name
	}
	if os.Getenv("DB_SSLMODE") == "" {
		os.Setenv("DB_SSLMODE", "disable")
	}
}

// clearDatabase helper function: truncates all relevant tables to ensure a clean database state for each test case.
func clearDatabase(t *testing.T) {
	// Order is important due to foreign key dependencies.
	tables := []string{"transactions", "wallets", "users"}
	for _, table := range tables {
		// TRUNCATE TABLE ... RESTART IDENTITY CASCADE clears the table, resets sequences, and handles foreign key dependencies.
		_, err := testApp.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", table))
		require.NoError(t, err, "Failed to truncate table %s", table)
	}
}

// createTestUserAndWallet helper function: quickly creates a user and wallet for testing.
// It now only returns the walletID as userID is not directly used by the API tests.
func createTestUserAndWallet(t *testing.T, username, currency string, initialBalance decimal.Decimal) int64 {
	user := domain.NewUser(username)
	// Pass testApp.DB as the DBExecutor
	err := testApp.UserRepository.CreateUser(context.Background(), testApp.DB, user)
	require.NoError(t, err) // If user creation fails, stop the test immediately

	wallet := domain.NewWallet(user.ID, currency)
	// Set initial balance directly here for test setup simplicity, not via API deposit.
	wallet.Balance = initialBalance
	// Pass testApp.DB as the DBExecutor
	err = testApp.WalletRepository.CreateWallet(context.Background(), testApp.DB, wallet)
	require.NoError(t, err)

	// Since NewWallet defaults balance to 0, directly update the database to reflect initialBalance.
	// This is a test setup trick to avoid calling the API during setup.
	_, err = testApp.DB.ExecContext(context.Background(), "UPDATE wallets SET balance = $1 WHERE id = $2", initialBalance, wallet.ID)
	require.NoError(t, err)

	return wallet.ID
}

// makeRequest helper function: sends an HTTP request to the test server.
func makeRequest(t *testing.T, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, testServer.URL+path, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// Do NOT defer resp.Body.Close() here. The caller is responsible for closing the body
	// because they might need to read it or check headers after this function returns.
	return resp, string(respBody)
}

// TestDepositIntegration tests the Deposit API endpoint.
func TestDepositIntegration(t *testing.T) {
	// Clear the database before each test run to ensure test independence.
	clearDatabase(t)
	// Create a test user and wallet with an initial balance of 0.
	walletID := createTestUserAndWallet(t, "deposit_user", "USD", decimal.NewFromInt(0))

	t.Run("SuccessfulDeposit", func(t *testing.T) {
		depositAmount := decimal.NewFromFloat(100.00)
		requestBody := fmt.Sprintf(`{"amount": "%s", "currency": "USD"}`, depositAmount.String())
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", walletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var responseMap map[string]interface{}
		err := json.Unmarshal([]byte(body), &responseMap)
		require.NoError(t, err)

		assert.Equal(t, "Deposit successful", responseMap["message"])
		assert.Equal(t, float64(walletID), responseMap["wallet_id"])
		// Verify new balance
		newBalance, err := decimal.NewFromString(responseMap["new_balance"].(string))
		require.NoError(t, err)
		assert.True(t, depositAmount.Equal(newBalance), "New balance should match deposit amount") // <-- 修改这里

		// Additional verification: confirm balance again via GET /balance endpoint.
		respGet, bodyGet := makeRequest(t, "GET", fmt.Sprintf("/wallets/%d/balance", walletID), nil)
		defer respGet.Body.Close()
		assert.Equal(t, http.StatusOK, respGet.StatusCode)
		var balanceMap map[string]interface{}
		err = json.Unmarshal([]byte(bodyGet), &balanceMap)
		require.NoError(t, err)
		retrievedBalance, err := decimal.NewFromString(balanceMap["balance"].(string))
		require.NoError(t, err)
		assert.True(t, depositAmount.Equal(retrievedBalance), "Retrieved balance should match deposit amount") // <-- 修改这里
	})

	t.Run("InvalidAmount", func(t *testing.T) {
		requestBody := `{"amount": "-10.00", "currency": "USD"}`
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", walletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "invalid input provided")
	})

	t.Run("WalletNotFound", func(t *testing.T) {
		nonExistentWalletID := int64(9999)
		requestBody := `{"amount": "50.00", "currency": "USD"}`
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", nonExistentWalletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Contains(t, body, "Resource not found")
	})

	t.Run("CurrencyMismatch", func(t *testing.T) {
		requestBody := `{"amount": "50.00", "currency": "HKD"}`
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", walletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode) // <-- 期望 400
		assert.Contains(t, body, "wallet currency mismatch")    // <-- 期望特定消息
	})

	t.Run("SuccessfulDeposit_EUR", func(t *testing.T) {
		eurWalletID := createTestUserAndWallet(t, "deposit_user_eur", "EUR", decimal.NewFromInt(0))
		depositAmount := decimal.NewFromFloat(200.00)
		requestBody := fmt.Sprintf(`{"amount": "%s", "currency": "EUR"}`, depositAmount.String())
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", eurWalletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var responseMap map[string]interface{}
		err := json.Unmarshal([]byte(body), &responseMap)
		require.NoError(t, err)

		assert.Equal(t, "Deposit successful", responseMap["message"])
		assert.Equal(t, float64(eurWalletID), responseMap["wallet_id"])
		newBalance, err := decimal.NewFromString(responseMap["new_balance"].(string))
		require.NoError(t, err)
		assert.True(t, depositAmount.Equal(newBalance), "New balance for EUR wallet should match deposit amount") // <-- 修改这里
	})
}

// TestWithdrawIntegration tests the Withdraw API endpoint.
func TestWithdrawIntegration(t *testing.T) {
	clearDatabase(t)
	walletID := createTestUserAndWallet(t, "withdraw_user", "USD", decimal.NewFromFloat(500.00))

	t.Run("SuccessfulWithdrawal", func(t *testing.T) {
		withdrawAmount := decimal.NewFromFloat(100.00)
		requestBody := fmt.Sprintf(`{"amount": "%s", "currency": "USD"}`, withdrawAmount.String())
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/withdraw", walletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var responseMap map[string]interface{}
		err := json.Unmarshal([]byte(body), &responseMap)
		require.NoError(t, err)

		assert.Equal(t, "Withdrawal successful", responseMap["message"])
		newBalance, err := decimal.NewFromString(responseMap["new_balance"].(string))
		require.NoError(t, err)
		expectedBalance := decimal.NewFromFloat(400.00)
		assert.True(t, expectedBalance.Equal(newBalance), "New balance should be 400.00") // <-- 修改这里
	})

	t.Run("InsufficientFunds", func(t *testing.T) {
		withdrawAmount := decimal.NewFromFloat(1000.00)
		requestBody := fmt.Sprintf(`{"amount": "%s", "currency": "USD"}`, withdrawAmount.String())
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/withdraw", walletID), strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusPaymentRequired, resp.StatusCode)
		assert.Contains(t, body, "Insufficient funds")
	})
}

// TestTransferIntegration tests the Transfer API endpoint.
func TestTransferIntegration(t *testing.T) {
	clearDatabase(t)
	walletID1 := createTestUserAndWallet(t, "transfer_user1", "USD", decimal.NewFromFloat(500.00))
	walletID2 := createTestUserAndWallet(t, "transfer_user2", "USD", decimal.NewFromFloat(100.00))

	t.Run("SuccessfulTransfer", func(t *testing.T) {
		transferAmount := decimal.NewFromFloat(50.00)
		requestBody := fmt.Sprintf(`{"from_wallet_id": %d, "to_wallet_id": %d, "amount": "%s", "currency": "USD"}`, walletID1, walletID2, transferAmount.String())
		resp, body := makeRequest(t, "POST", "/transfers", strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var responseMap map[string]interface{}
		err := json.Unmarshal([]byte(body), &responseMap)
		require.NoError(t, err)

		assert.Equal(t, "Transfer successful", responseMap["message"])
		fromWalletNewBalance, err := decimal.NewFromString(responseMap["from_wallet_new_balance"].(string))
		require.NoError(t, err)

		expectedFromBalance := decimal.NewFromFloat(450.00)

		assert.True(t, expectedFromBalance.Equal(fromWalletNewBalance), "From wallet new balance should be 450.00")
	})

	t.Run("SameWalletTransfer", func(t *testing.T) {
		transferAmount := decimal.NewFromFloat(10.00)
		requestBody := fmt.Sprintf(`{"from_wallet_id": %d, "to_wallet_id": %d, "amount": "%s", "currency": "USD"}`, walletID1, walletID1, transferAmount.String())
		resp, body := makeRequest(t, "POST", "/transfers", strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "Cannot transfer to the same wallet")
	})

	t.Run("InsufficientFundsInSourceWallet", func(t *testing.T) {
		transferAmount := decimal.NewFromFloat(200.00)
		requestBody := fmt.Sprintf(`{"from_wallet_id": %d, "to_wallet_id": %d, "amount": "%s", "currency": "USD"}`, walletID2, walletID1, transferAmount.String())
		resp, body := makeRequest(t, "POST", "/transfers", strings.NewReader(requestBody))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusPaymentRequired, resp.StatusCode)
		assert.Contains(t, body, "Insufficient funds")
	})
}

// TestTransactionHistoryAndBalanceConsistency tests transaction history and balance consistency.
func TestTransactionHistoryAndBalanceConsistency(t *testing.T) {
	clearDatabase(t)
	walletID := createTestUserAndWallet(t, "consistency_user", "USD", decimal.NewFromInt(0))

	// Perform a series of operations.
	depositAmount1 := decimal.NewFromFloat(500.00)
	resp1, _ := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", walletID), strings.NewReader(fmt.Sprintf(`{"amount": "%s", "currency": "USD"}`, depositAmount1.String())))
	defer resp1.Body.Close()
	time.Sleep(10 * time.Millisecond)

	withdrawAmount := decimal.NewFromFloat(150.00)
	resp2, _ := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/withdraw", walletID), strings.NewReader(fmt.Sprintf(`{"amount": "%s", "currency": "USD"}`, withdrawAmount.String())))
	defer resp2.Body.Close()
	time.Sleep(10 * time.Millisecond)

	depositAmount2 := decimal.NewFromFloat(200.00)
	resp3, _ := makeRequest(t, "POST", fmt.Sprintf("/wallets/%d/deposit", walletID), strings.NewReader(fmt.Sprintf(`{"amount": "%s", "currency": "USD"}`, depositAmount2.String())))
	defer resp3.Body.Close()
	time.Sleep(10 * time.Millisecond)

	// Expected final balance: 0 + 500 - 150 + 200 = 550
	expectedFinalBalance := decimal.NewFromFloat(550.00)

	// 1. Get current balance.
	respBalance, bodyBalance := makeRequest(t, "GET", fmt.Sprintf("/wallets/%d/balance", walletID), nil)
	defer respBalance.Body.Close()
	assert.Equal(t, http.StatusOK, respBalance.StatusCode)
	var balanceMap map[string]interface{}
	err := json.Unmarshal([]byte(bodyBalance), &balanceMap)
	require.NoError(t, err)
	currentBalance, err := decimal.NewFromString(balanceMap["balance"].(string))
	require.NoError(t, err)
	assert.True(t, expectedFinalBalance.Equal(currentBalance), "Current balance should match expected final balance") // <-- 修改这里

	// 2. Get transaction history.
	respHistory, bodyHistory := makeRequest(t, "GET", fmt.Sprintf("/wallets/%d/transactions?limit=10&offset=0", walletID), nil)
	defer respHistory.Body.Close()
	assert.Equal(t, http.StatusOK, respHistory.StatusCode)
	var historyMap map[string]interface{}
	err = json.Unmarshal([]byte(bodyHistory), &historyMap)
	require.NoError(t, err)

	transactionsData := historyMap["data"].([]interface{})
	assert.Len(t, transactionsData, 3, "Should have 3 transactions")

	// 3. Calculate balance from transaction history.
	calculatedBalanceFromHistory := decimal.NewFromInt(0) // Start calculation from 0
	for _, txInterface := range transactionsData {
		txMap := txInterface.(map[string]interface{})
		amountStr := txMap["amount"].(string)
		txType := txMap["type"].(string)

		amount, err := decimal.NewFromString(amountStr)
		require.NoError(t, err)

		switch domain.TransactionType(txType) {
		case domain.TransactionTypeDeposit:
			calculatedBalanceFromHistory = calculatedBalanceFromHistory.Add(amount)
		case domain.TransactionTypeWithdrawal:
			calculatedBalanceFromHistory = calculatedBalanceFromHistory.Sub(amount)
		case domain.TransactionTypeTransfer:
			if txMap["from_wallet_id"] != nil && int64(txMap["from_wallet_id"].(float64)) == walletID {
				calculatedBalanceFromHistory = calculatedBalanceFromHistory.Sub(amount)
			} else if txMap["to_wallet_id"] != nil && int64(txMap["to_wallet_id"].(float64)) == walletID {
				calculatedBalanceFromHistory = calculatedBalanceFromHistory.Add(amount)
			}
		}
	}

	// 4. Compare the two balances for consistency.
	assert.True(t, currentBalance.Equal(calculatedBalanceFromHistory), "Balance derived from history should match current balance") // <-- 修改这里
}
