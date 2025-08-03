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
*   **View Transaction History:** Provides a list of all transactions for a specified wallet.

## Technology Stack

*   **Language:** Go (version 1.23.0)
*   **Database:** PostgreSQL
*   **HTTP Framework:** `net/http` (standard library) 
*   **Database ORM/Client:** `github.com/jmoiron/sqlx` (for simplified SQL operations and struct scanning)
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
    *   `username` (VARCHAR, UNIQUE)
    *   `created_at`, `updated_at` (TIMESTAMPTZ)
*   **Wallet:** Represents a user's wallet.
    *   `id` (PK，auto-increase integer)
    *   `user_id` (FK to `users.id`)
    *   `currency` (VARCHAR, e.g., 'USD', 'FIAT')
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
