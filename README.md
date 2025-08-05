# Finflow Wallet Application Backend

*   This project implements a centralized wallet application backend with RESTful API endpoints, as a technical assignment for Crypto.com's Bank Services - Senior Software Engineer position.
*   For the name "Finflow", "Fin" stands for financial technology (which is also my major of MSc in HKUST), and "flow" stands for the dynamics of financial entities, like deposit/withdraw/transfer of wallets.

## Table of Contents

*   [Features](#features)
*   [Technology Stack](#technology-stack)
*   [Architecture](#architecture)
*   [Data Model](#data-model)
*   [Getting Started](#getting-started)
    *   [Prerequisites](#prerequisites)
    *   [Setup Environment](#setup-environment)
    *   [Run Database Migrations](#run-database-migrations)
    *   [Run the Application](#run-the-application)
*   [API Endpoints](#api-endpoints)
*   [Testing](#testing)
*   [Design Decisions & Rationale](#design-decisions--rationale)
*   [Areas for Improvement](#areas-for-improvement)
*   [Time Spent](#time-spent)
*   [Features Not Implemented](#features-not-implemented)
*   [How to Review the Code](#how-to-review-the-code)

## Features

The application provides the following core functionalities via RESTful APIs:

*   **Deposit Money:** Allows users to deposit funds into their wallets.
*   **Withdraw Money:** Allows users to withdraw funds from their wallets, with balance checks.
*   **Transfer Money:** Enables users to send money to other users, ensuring atomicity and sufficient funds.
*   **Check Wallet Balance:** Retrieves the current balance of a specified wallet.
*   **View Transaction History:** Provides a list of all transactions for a specified wallet with pagination.

## Technology Stack

*   **Language:** Go (version 1.23.0)
*   **Database:** PostgreSQL
*   **HTTP Framework:** `net/http` (standard library) 
*   **Database Client:** `github.com/jmoiron/sqlx` (for simplified SQL operations and struct scanning)
*   **Containerization:** Docker, Docker Compose

## Architecture

The application follows a layered architecture to ensure separation of concerns, maintainability, and testability:

1.  **API/Handler Layer (`internal/api/handler`):** Handles incoming HTTP requests, parses parameters, performs basic input validation, and marshals responses. It orchestrates calls to the Service layer.
2.  **Service Layer (`internal/service`):** Contains the core business logic. It orchestrates operations across multiple repositories, manages database transactions, and enforces business rules (e.g., balance checks, transfer atomicity).
3.  **Repository/Data Access Layer (`internal/repository`):** Abstracts database interactions. It provides interfaces for CRUD operations on `User`, `Wallet`, and `Transaction` entities, with concrete implementations for PostgreSQL (`internal/repository/postgres`).

## Data Model

The core entities and their relationships are as follows:

*   **User:** Represents a user of the wallet system.
    *   `id` (PK，auto-increase integer)
    *   `username` (VARCHAR)
    *   `created_at`, `updated_at` (TIMESTAMPTZ)
*   **Wallet:** Represents a user's wallet.
    *   `id` (PK，auto-increase integer)
    *   `user_id` (FK to `users.id`)
    *   `currency` (VARCHAR, e.g., 'USD')
    *   `balance` (NUMERIC(20, 4), for high precision)
    *   `created_at`, `updated_at` (TIMESTAMPTZ)
*   **Transaction:** Records all financial movements.
    *   `id` (PK，auto-increase integer)
    *   `from_wallet_id` (FK to `wallets.id`, NULLABLE)
    *   `to_wallet_id` (FK to `wallets.id`, NULLABLE)
    *   `amount` (NUMERIC(20, 4))
    *   `currency` (VARCHAR)
    *   `type` (VARCHAR, e.g., 'DEPOSIT', 'WITHDRAWAL', 'TRANSFER')
    *   `status` (VARCHAR, e.g., 'COMPLETED')
    *   `transaction_time` (TIMESTAMPTZ)
    *   `description` (TEXT, OPTIONAL)
    *   `created_at` (TIMESTAMPTZ)

## Getting Started

### Prerequisites

*   Go (1.23.0 or later)
*   Docker & Docker Compose (for running PostgreSQL)
*   Make (optional, for convenience scripts)

### Setup Environment

1.  **Clone the repository:**
    ```bash
    git clone git@github.com:netwalker2000/finflow-wallet.git
    cd finflow-wallet
    ```
2.  **Start PostgreSQL using Docker Compose:**
    ```bash
    docker-compose up -d postgres
    ```
    This will start a PostgreSQL container. Check `docker-compose.yml` for port and credentials.

### Run Database Migrations

Ensure the PostgreSQL container is running.

```bash
# Install migrate CLI if you haven't already
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations, including create table and insert data
migrate -path migrations -database "postgres://user:password@localhost:5432/walletdb?sslmode=disable" up

# query test data
docker exec -it finflow_postgres psql -U user -d walletdb
input sql and query then \q to quit

# clean data and tables when you want to reset the test
migrate -path migrations -database "postgres://user:password@localhost:5432/walletdb?sslmode=disable" down
```

### Run the Application

```bash
make all
go run cmd/api/main.go
```

## API Endpoints

*   The application exposes the following `RESTful API` endpoints:
*   **Note:**
    * The walletID field should be an integer
    * The currency symbol field is case sensitive

### Wallet Operations

*   **Deposit Money**
    *   **Endpoint:** `POST /wallets/{walletID}/deposit`
    *   **Description:** Deposits a specified amount into the given wallet.
    *   **Path Parameters:**
        *   `walletID` (integer): The ID of the wallet to deposit into.
    *   **Request Body (JSON):**
        ```json
        {
            "amount": "100.00",
            "currency": "USD"
        }
        ```
    *   **Successful Response (200 OK):**
        ```json
        {
            "message": "Deposit successful",
            "wallet_id": 1,
            "new_balance": "600.00",
            "transaction_id": 101
        }
        ```
    *   **Error Response:** 
        * If wallet does not exist - "Resource not found"
        * If ID input format error - "invalid input provided"
        * If currency mismatch - "wallet currency mismatch"
        * If amount is not bigger than 0 - "invalid input provided"


*   **Withdraw Money**
    *   **Endpoint:** `POST /wallets/{walletID}/withdraw`
    *   **Description:** Withdraws a specified amount from the given wallet, subject to balance checks.
    *   **Path Parameters:**
        *   `walletID` (integer): The ID of the wallet to withdraw from.
    *   **Request Body (JSON):**
        ```json
        {
            "amount": "50.00",
            "currency": "USD"
        }
        ```
    *   **Successful Response (200 OK):**
        ```json
        {
            "message": "Withdrawal successful",
            "wallet_id": 1,
            "new_balance": "550.00",
            "transaction_id": 102
        }
        ```
    *   **Error Response:** 
        * If wallet does not exist - "Resource not found"
        * If ID input format error - "invalid input provided"
        * If currency mismatch - "wallet currency mismatch"
        * If amount is not bigger than 0 - "invalid input provided"
        * If insufficient funds - "Insufficient funds"

*   **Get Wallet Balance**
    *   **Endpoint:** `GET /wallets/{walletID}/balance`
    *   **Description:** Retrieves the current balance of a specific wallet.
    *   **Path Parameters:**
        *   `walletID` (integer): The ID of the wallet.
    *   **Successful Response (200 OK):**
        ```json
        {
            "wallet_id": 1,
            "balance": "550.00",
            "currency": "USD"
        }
        ```
    *   **Error Response:** 
        * If wallet does not exist - "Resource not found"
        * If ID input format error - "invalid input provided"

*   **Get Transaction History**
    *   **Endpoint:** `GET /wallets/{walletID}/transactions`
    *   **Description:** Retrieves a paginated list of transactions for a specific wallet.
    *   **Path Parameters:**
        *   `walletID` (integer): The ID of the wallet.
    *   **Query Parameters:**
        *   `limit` (integer, optional): Maximum number of transactions to return (default: 10).
        *   `offset` (integer, optional): Number of transactions to skip (default: 0).
    *   **Successful Response (200 OK):**
        ```json
        {
            "data": [
                {
                    "id": 102,
                    "from_wallet_id": 1,
                    "to_wallet_id": null,
                    "amount": "50.00",
                    "currency": "USD",
                    "type": "WITHDRAWAL",
                    "status": "COMPLETED",
                    "transaction_time": "2025-08-03T10:00:00Z",
                    "description": null,
                    "created_at": "2025-08-03T10:00:00Z"
                },
                {
                    "id": 101,
                    "from_wallet_id": null,
                    "to_wallet_id": 1,
                    "amount": "100.00",
                    "currency": "USD",
                    "type": "DEPOSIT",
                    "status": "COMPLETED",
                    "transaction_time": "2025-08-03T09:00:00Z",
                    "description": null,
                    "created_at": "2025-08-03T09:00:00Z"
                }
            ],
            "limit": 10,
            "offset": 0,
            "total_count": 25 
        }
        ```
    *   **Error Response:** 
        * If wallet does not exist - "Resource not found"
        * If ID input format error - "invalid input provided"
    *   **How to implement pagination on the frontend:**
        * `data`: The array of transaction objects for the current page.
        * `limit`: The maximum number of items requested per page.
        * `offset`: The number of items skipped from the beginning, indicating the starting point of the current page.
        * `total_count`: The total number of available transactions for the given wallet, across all pages.
        * Frontend applications can use `total_count` along with `limit` to calculate the total number of pages **(ceil(total_count / limit))**. Users can then navigate between pages by adjusting the `offset` query parameter (e.g., offset = page_number * limit)

### Transfer Operations

*   **Transfer Money**
    *   **Endpoint:** `POST /transfers`
    *   **Description:** Transfers a specified amount from one wallet to another. This is an atomic operation.
    *   **Request Body (JSON):**
        ```json
        {
            "from_wallet_id": 1,
            "to_wallet_id": 2,
            "amount": "25.00",
            "currency": "USD"
        }
        ```
    *   **Successful Response (200 OK):**
        ```json
        {
            "message": "Transfer successful",
            "transaction_id": 103,
            "from_wallet_new_balance": "525.00"
        }
        ```
    *   **Note:** 
        * we ignore to_wallet_new_balance for security reasons, you don't want to expose the balance passively   
    *   **Error Response:** 
        * If wallet does not exist - "Resource not found"
        * If ID input format error - "invalid input provided"
        * If same wallet transfer - "Cannot transfer to the same wallet"
        * If currency mismatch - "wallet currency mismatch"
        * If insufficient funds in the source wallet - "Insufficient funds"
        * If amount is not bigger than 0 - "invalid input provided"

---

## Testing

This project prioritizes code quality and reliability through a robust testing strategy, encompassing both unit and integration tests. This ensures the core business logic functions correctly and the entire application stack behaves as expected in an end-to-end scenario.

### Unit Tests

Unit tests are designed to validate individual components (e.g., service methods, repository functions) in isolation. They use mocking to simulate external dependencies, ensuring focused testing of specific logic without external interference.

*   **Framework:** Go's built-in `testing` package, complemented by `stretchr/testify/mock` for dependency mocking.
*   **Scope:** Core business logic within `internal/service` and `internal/repository` layers.
*   **How to Run:**
    ```bash
    go test ./internal/...
    ```
    (This command runs all unit tests within the `internal` directory and its subdirectories.)

### Integration Tests

Integration tests verify the interaction between different components and external services (like the database) as a complete system. They simulate real API requests, providing confidence in the application's end-to-end functionality.

#### Automated Integration Tests

These tests spin up an in-memory HTTP server and interact with a dedicated test database, providing a fast and reliable way to validate the full application stack.

*   **Framework:** Go's built-in `testing` package, `net/http/httptest` for the in-memory server, and `stretchr/testify/assert` for comprehensive assertions.
*   **Scope:** HTTP API endpoints, service layer, repository layer, and database interactions.
*   **Environment Setup (Crucial Pre-requisites):**
    Before running automated integration tests, ensure a dedicated PostgreSQL test database is set up and migrated. This guarantees a clean and predictable state for each test run.
    1.  **Start PostgreSQL:** Ensure your PostgreSQL container (e.g., via `docker-compose up -d postgres`) is running.
    2.  **Create Test Database:** Connect to your PostgreSQL instance and create a separate database for testing. Replace `user` and `password` with your actual credentials.
        ```bash
        docker exec -it finflow_postgres psql -U user -d postgres -c "CREATE DATABASE walletdb_test;"
        ```
    3.  **Run Migrations:** Apply database migrations to the newly created test database.
        ```bash
        migrate -path migrations -database "postgres://user:password@localhost:5432/walletdb_test?sslmode=disable" up
        ```
    4.  **Set Environment Variables:** Ensure the following environment variables are set to point to your test database.
        ```bash
        export DB_HOST=localhost
        export DB_PORT=5432
        export DB_USER=user
        export DB_PASSWORD=password
        export DB_NAME=walletdb_test
        export DB_SSLMODE=disable
        ```
*   **How to Run:**
    ```bash
    go test ./internal/api -v -run Integration
    ```
    (The `-run Integration` flag specifically targets the automated integration tests.)

#### Manual/Ad-hoc Integration Tests

For quick functional checks or debugging, you can interact with the running application directly using `curl` or similar tools.

*   **Tool:** `curl` (command-line tool).
*   **Scope:** Basic API functionality verification against a live running instance.
*   **Examples (assuming the application is running on `http://localhost:8080`):**
    ```bash
    # Get wallet balance
    curl http://localhost:8080/wallets/1/balance

    # Deposit money
    curl -X POST -H "Content-Type: application/json" -d '{"amount": "100.00", "currency": "USD"}' http://localhost:8080/wallets/1/deposit

    # Withdraw money
    curl -X POST -H "Content-Type: application/json" -d '{"amount": "50.00", "currency": "USD"}' http://localhost:8080/wallets/1/withdraw

    # Transfer money
    curl -X POST -H "Content-Type: application/json" -d '{"from_wallet_id": 1, "to_wallet_id": 3, "amount": "25.00", "currency": "USD"}' http://localhost:8080/transfers

    # Get transaction history
    curl http://localhost:8080/wallets/1/transactions
    ```


## Design Decisions & Rationale

*   **Layered Architecture:** Adopted a Handler-Service-Repository pattern to promote modularity, separation of concerns, and testability.
    *   **Handler:** Focuses on HTTP concerns (request/response).
    *   **Service:** Encapsulates business logic and transaction management.
    *   **Repository:** Handles data persistence logic.
*   **Concurrency Control:**
    *   Database transactions (`sql.Tx`) are used for all money-altering operations (deposit, withdraw, transfer) to guarantee atomicity.
*   **Error Handling:** Custom error types (`util.ErrInsufficientFunds`, `util.ErrNotFound`, etc.) are defined to provide specific business context. Errors are wrapped using `fmt.Errorf("%w", err)` to maintain a clear error chain, aiding debugging. A centralized error handling middleware or function in the API layer translates these internal errors into appropriate HTTP responses.
*   **Go Generics (Go 1.23.0):**
    *   Generics were utilized for `PaginatedResponse[T any]` to provide a reusable structure for API responses that include lists of items with pagination metadata. This avoids code duplication for different list types.
*   **`BIGSERIAL` for Primary Keys:** Chosen over UUIDs for primary keys (`id` columns) to optimize database performance, especially for insertions and indexing. `BIGSERIAL` provides auto-incrementing, large integer IDs, which offer better spatial locality and smaller index sizes compared to random UUIDs, crucial for high-volume financial data.
*   **`TIMESTAMPTZ` for Timestamps:** Used `TIMESTAMPTZ` (timestamp with time zone) for all time-related columns (`created_at`, `updated_at`, `transaction_time`). This ensures that all timestamps are stored internally in UTC, providing an unambiguous and precise record of events regardless of server location or time zone settings, which is critical for auditability and consistency in financial applications.
*   **`NUMERIC(20, 4)` for Monetary Values:**
    *   Crucial for financial applications to avoid floating-point inaccuracies. PostgreSQL's `NUMERIC` type provides arbitrary precision arithmetic.
    *   The `(20, 4)` precision was chosen based on the understanding that the "money" in this context primarily refers to **fiat currencies**, which typically require up to 4 decimal places for precision (e.g., in foreign exchange markets).
    *   This configuration provides 16 digits before the decimal point (up to `9,999,999,999,999,999.9999`), offering ample scale for large fiat currency balances and transaction amounts, while being efficient in storage compared to higher precision that might be needed for cryptocurrencies.

## Areas for Improvement

While this project provides a solid foundation, there are several areas where it could be further improved and extended:

*   **Comprehensive Input Validation:** Implement more robust and centralized input validation (e.g., using a dedicated validation library like `go-playground/validator`) for all API requests to ensure data integrity and security.
*   **Authentication and Authorization:** Integrate a proper authentication mechanism (e.g., JWT, OAuth2) and authorization (e.g., RBAC) to secure API endpoints and manage user permissions.
*   **Rate Limiting:** Implement rate limiting to protect the API from abuse and denial-of-service attacks.
*   **Observability:** Add metrics (e.g., Prometheus), tracing (e.g., OpenTelemetry), and more detailed structured logging to enhance monitoring and debugging capabilities in production.
*   **Idempotency:** Implement idempotency keys for write operations (Deposit, Withdraw, Transfer) to prevent duplicate processing of requests due to network retries.
*   **More Granular Error Handling:** Define more specific custom error types and map them to HTTP status codes for a richer API error response.
*   **Configuration Management:** Explore more advanced configuration management solutions (e.g., Viper) to support different environments (development, staging, production) and external configuration sources.
*   **Concurrency Control:** For extremely high-volume scenarios, consider more advanced concurrency control mechanisms beyond simple database transactions, such as optimistic locking or distributed locks, though for typical loads, database transactions are sufficient.
*   **User Management API:** Implement API endpoints for user creation, retrieval, and management, rather than assuming manual user creation.

## Time Spent

The development of this project involved the following time allocation:

*   **August 1, 2025:**
    *   20:30 - 22:30 (2 hours): Initial project research and understanding the assignment scope.
*   **August 2, 2025:**
    *   11:00 - 11:30 (0.5 hours): Communication regarding assignment scope.
    *   14:00 - 16:00 (2 hours): Database schema design and migration scripts.
    *   16:00 - 18:30 (2.5 hours): Initial service layer development (query operations functional, transactional operations pending).
*   **August 3, 2025:**
    *   10:00 - 12:30 (2.5 hours): Debugging and resolving transaction-related issues in service layer tests.
    *   13:00 - 14:00 (1 hour): Ensuring service layer build stability and passing all unit tests.
    *   14:00 - 15:00 (1 hour): Completion of all core code (config, app init, API handlers, main) and successful integration testing.
    *   15:00 - 16:30 (1.5 hours): README documentation, including API endpoints and testing sections.
*   **August 4, 2025:**
    *   15:30 - 20:00 (4.5 hours): Research and development for project-related enhancements; implemented integration test automation and performed both automated and manual testing.
*   **August 5, 2025:**
    *   10:00 - 13:00 (3 hours): Continued project refinement, specifically optimizing pagination design.

**Total Estimated Time: about 20 hours**

## Features Not Implemented

Due to the scope and time constraints, the following features, which are common in a production-grade wallet application, were not implemented:

*   **User Management API:** No API endpoints for creating, updating, or deleting users. Users are assumed to be pre-existing or managed externally.
*   **Advanced Currency Management:** No support for multiple currencies within a single wallet, currency conversion, or exchange rates. Each wallet is tied to a single currency.
*   **Transaction Fees:** The current implementation does not account for any transaction fees for deposits, withdrawals, or transfers.
*   **Soft Deletion:** Entities are not soft-deleted; they are assumed to be hard-deleted or remain in the database.
*   **Wallet Freezing/Blocking:** No functionality to freeze or block wallets (e.g., for suspicious activity).
*   **Audit Trails:** While transactions serve as a basic audit, a more comprehensive audit trail for all system changes (e.g., user updates, configuration changes) is not in place.
*   **Webhooks/Notifications:** No real-time notifications (e.g., webhooks, email, SMS) for transaction events.
*   **Scheduled Transactions:** No support for future-dated or recurring transactions.
*   **Admin Panel/API:** No separate interfaces or APIs for administrative tasks.

## How to Review the Code

To effectively review the codebase, it's recommended to follow the application's layered architecture and focus on key design principles:

1.  **Start with `cmd/api/main.go`:** Understand the application's entry point, how it initializes, and its graceful shutdown mechanism.
2.  **Review `internal/app.go`:** This file orchestrates the initialization of all components (config, logger, DB, repositories, services, handlers, router). Pay attention to dependency injection.
3.  **Examine `internal/config/config.go` and `internal/util/logger.go`:** Understand how configuration is loaded and how structured logging is set up.
4.  **Dive into `internal/api/router.go` and `internal/api/handler/wallet.go`:**
    *   `router.go`: See how HTTP routes are defined and mapped to handlers.
    *   `wallet.go`: Review how requests are handled, parameters are parsed, input is validated, and service methods are called. Observe the `respondWithJSON` and `respondWithError` helpers and the error mapping logic.
5.  **Analyze `internal/service/wallet_service.go`:** This is the core business logic layer.
    *   Pay close attention to the transaction management (`beginTxFn`, `commitTxFn`, `rollbackTxFn`) and how it's used to ensure atomicity for deposit, withdraw, and transfer operations.
    *   Verify the business rules (e.g., insufficient funds checks, currency matching, same wallet transfer prevention).
    *   Review how different repository methods are orchestrated.
6.  **Inspect `internal/repository/` and `internal/repository/postgres/`:**
    *   `internal/repository/*.go`: Understand the repository interfaces (contracts) that define data access operations.
    *   `internal/repository/postgres/*.go`: Review the concrete PostgreSQL implementations. Note how `sqlx.ExtContext` is used to allow repository methods to operate within an existing transaction.
7.  **Check `pkg/db/`:** Understand the abstraction for database connection and transaction management. The `TxController` and `DBTxBeginner` interfaces are crucial for testability.
8.  **Review `internal/util/errors.go`:** Understand the custom error types used for business logic errors.
9.  **Examine Tests:**
    *   **Unit Tests (`internal/service/*_test.go`):** See how `testify/mock` is used to mock dependencies, especially for repositories and transaction controllers. Verify that each test case (`t.Run`) is isolated and sets up its own mocks. Confirm that expected method calls and their arguments are asserted. Understand how different error scenarios are tested.
    *   **Integration Tests (`internal/api/api_integration_test.go`):** Review how the entire API is tested end-to-end, interacting with a real database. Observe how HTTP requests are simulated and responses are validated, ensuring the system behaves correctly as a whole. Pay attention to the setup and teardown of the test environment (e.g., database clearing).

## Issue fixed:

1.  **API Response Amount Formatting:** Implemented consistent balance or amount formatting in API responses (e.g., `650.00` instead of `650`). This required adjustments to integration tests to validate the new string format, while unit tests remain unaffected.
2.  **Enhanced Currency Mismatch Error Handling:** Improved error handling for `deposit`, `withdraw`, and `transfer` operations. Previously, currency mismatches resulted in a generic "Internal server error"; now, these are explicitly caught with a more informative "Wallet currency mismatch" message.
3.  **Variable Name Refactoring:** Reviewed and updated several program variable names for improved clarity and readability.    