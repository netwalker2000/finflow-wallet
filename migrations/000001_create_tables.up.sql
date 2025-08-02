-- Table: users
-- Stores user information.
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY, -- Using BIGSERIAL for auto-incrementing, large integer primary key
    username VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- TIMESTAMPTZ for unambiguous, UTC-based timestamps
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for faster lookups by username
CREATE UNIQUE INDEX idx_users_username ON users (username);

-- Table: wallets
-- Stores wallet information for each user, including balance and currency.
CREATE TABLE wallets (
    id BIGSERIAL PRIMARY KEY, -- Using BIGSERIAL for auto-incrementing, large integer primary key
    user_id BIGINT NOT NULL REFERENCES users(id), -- Foreign key to users table
    currency VARCHAR(10) NOT NULL, -- e.g., 'USD', 'FIAT', 'BTC', 'ETH'
    balance NUMERIC(20, 4) NOT NULL DEFAULT 0.00, -- Adjusted to NUMERIC(20, 4) for fiat money
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure a user has only one wallet per currency
    UNIQUE(user_id, currency)
);

-- Index for faster lookups by user_id
CREATE INDEX idx_wallets_user_id ON wallets (user_id);

-- Table: transactions
-- Records all financial movements (deposits, withdrawals, transfers).
CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY, -- Using BIGSERIAL for auto-incrementing, large integer primary key
    from_wallet_id BIGINT REFERENCES wallets(id), -- Source wallet for withdrawals/transfers (nullable for deposits)
    to_wallet_id BIGINT REFERENCES wallets(id),   -- Destination wallet for deposits/transfers (nullable for withdrawals)
    amount NUMERIC(20, 4) NOT NULL CHECK (amount > 0), -- Adjusted to NUMERIC(20, 4), and must be positive
    currency VARCHAR(10) NOT NULL, -- Currency of the transaction
    type VARCHAR(50) NOT NULL,     -- e.g., 'DEPOSIT', 'WITHDRAWAL', 'TRANSFER'
    status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED', -- e.g., 'COMPLETED', 'PENDING', 'FAILED'
    transaction_time TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- The actual time the transaction occurred
    description TEXT,               -- Optional description of the transaction
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure that at least one wallet is involved in a transaction
    CHECK (from_wallet_id IS NOT NULL OR to_wallet_id IS NOT NULL),
    -- For transfers, ensure source and destination wallets are different
    CHECK (from_wallet_id IS NULL OR to_wallet_id IS NULL OR from_wallet_id <> to_wallet_id)
);

-- Indexes for faster transaction history lookups
CREATE INDEX idx_transactions_from_wallet_id ON transactions (from_wallet_id);
CREATE INDEX idx_transactions_to_wallet_id ON transactions (to_wallet_id);
CREATE INDEX idx_transactions_transaction_time ON transactions (transaction_time DESC); -- For chronological history