# Finflow Wallet Application Backend

This project implements a centralized wallet application backend with RESTful API endpoints, as a technical assignment for Crypto.com's Bank Services - Senior Software Engineer position.
For the name "Finflow", "Fin" stands for financial technology (which is also my major of MSc in HKUST), and "flow" stands for the dynamics of financial entities, like deposit/withdraw/transfer of wallets.

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

# Run migrations
migrate -path migrations -database "postgres://user:password@localhost:5432/walletdb?sslmode=disable" up

# Insert test data
docker exec -i finflow_postgres psql -U user -d walletdb < migrations/000001_insert_test_data.sql

# query test data
docker exec -it finflow_postgres psql -U user -d walletdb
input sql and query then \q to quit
```

### Run the Application

```bash
make all
go run cmd/api/main.go
```

## API Endpoints

The application exposes the following RESTful API endpoints:

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
    *   **Error Response (402 Payment Required):** If insufficient funds.

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
    *   **Error Response (404 Not Found):** If wallet does not exist.

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
            "offset": 0
        }
        ```
    *   **Error Response (404 Not Found):** If wallet does not exist.

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
            "from_wallet_new_balance": "525.00",
            "to_wallet_new_balance": "25.00"
        }
        ```
    *   **Error Response (400 Bad Request):** For invalid input, same wallet transfer, or currency mismatch.
    *   **Error Response (402 Payment Required):** If insufficient funds in the source wallet.
    *   **Error Response (404 Not Found):** If either wallet does not exist.

---

## Testing

The project includes comprehensive unit tests for the core business logic and repository layers, ensuring correctness and robustness.

### How to Run Tests

To run all unit tests:

```bash
go test ./...
```

To run the integtion test
```bash
curl http://localhost:8080/wallets/1/balance

curl http://localhost:8080/wallets/1/transactions

curl -X POST -H "Content-Type: application/json" -d '{"amount": "100.00", "currency": "USD"}' http://localhost:8080/wallets/1/deposit

curl -X POST -H "Content-Type: application/json" -d '{"amount": "50.00", "currency": "USD"}' http://localhost:8080/wallets/1/withdraw


curl -X POST -H "Content-Type: application/json" -d '{"from_wallet_id": 1, "to_wallet_id": 2, "amount": "25.00", "currency": "USD"}' http://localhost:8080/transfers
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
