# Wallet Application Backend

This project implements a centralized wallet application backend with RESTful API endpoints, as a technical assignment for Crypto.com's Bank Services - Senior Software Engineer position.

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
    *   `id` (UUID, PK)
    *   `username` (VARCHAR, UNIQUE)
    *   `created_at`, `updated_at` (TIMESTAMPTZ)
*   **Wallet:** Represents a user's wallet.
    *   `id` (UUID, PK)
    *   `user_id` (UUID, FK to `users.id`)
    *   `currency` (VARCHAR, e.g., 'USD', 'FIAT')
    *   `balance` (NUMERIC(20, 8), for high precision)
    *   `created_at`, `updated_at` (TIMESTAMPTZ)
*   **Transaction:** Records all financial movements.
    *   `id` (UUID, PK)
    *   `from_wallet_id` (UUID, FK to `wallets.id`, NULLABLE)
    *   `to_wallet_id` (UUID, FK to `wallets.id`, NULLABLE)
    *   `amount` (NUMERIC(20, 8))
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
    git clone <your-github-repo-link>
    cd wallet-app
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