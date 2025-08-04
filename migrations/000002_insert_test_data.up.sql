-- 000002_insert_test_data.up.sql

-- Insert test users
INSERT INTO users (username) VALUES ('alice');
INSERT INTO users (username) VALUES ('bob');
INSERT INTO users (username) VALUES ('charlie');

-- Insert wallets for users with initial balance 0.00
-- Alice's USD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'alice'),
    'USD',
    0.00
);

-- Alice's HKD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'alice'),
    'HKD',
    0.00
);

-- Bob's USD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'bob'),
    'USD',
    0.00
);

-- Bob's HKD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'bob'),
    'HKD',
    0.00
);

-- Charlie's USD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'charlie'),
    'USD',
    0.00
);

-- Charlie's HKD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'charlie'),
    'HKD',
    0.00
);

-- --- Simulate initial deposits for all 6 wallets ---

-- Deposit for Alice's USD wallet
DO $$
DECLARE
    wallet_id BIGINT;
    deposit_amount NUMERIC(20, 4) := 1000.00; -- Alice USD 初始存款
BEGIN
    SELECT id INTO wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'USD';
    UPDATE wallets SET balance = balance + deposit_amount WHERE id = wallet_id;
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, deposit_amount, 'USD', 'DEPOSIT', 'Initial deposit to Alice''s USD wallet');
END $$;

-- Deposit for Alice's HKD wallet
DO $$
DECLARE
    wallet_id BIGINT;
    deposit_amount NUMERIC(20, 4) := 500.00; -- Alice HKD 初始存款
BEGIN
    SELECT id INTO wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'HKD';
    UPDATE wallets SET balance = balance + deposit_amount WHERE id = wallet_id;
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, deposit_amount, 'HKD', 'DEPOSIT', 'Initial deposit to Alice''s HKD wallet');
END $$;

-- Deposit for Bob's USD wallet (Non-zero deposit, followed by immediate withdrawal to keep balance at 0 before transfer)
DO $$
DECLARE
    wallet_id BIGINT;
    deposit_amount NUMERIC(20, 4) := 100.00; -- Bob USD 初始存款 (必须大于0)
    withdrawal_amount NUMERIC(20, 4) := 100.00; -- 等额取款
BEGIN
    SELECT id INTO wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'bob') AND currency = 'USD';
    
    -- Deposit
    UPDATE wallets SET balance = balance + deposit_amount WHERE id = wallet_id;
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, deposit_amount, 'USD', 'DEPOSIT', 'Initial deposit to Bob''s USD wallet');

    -- Withdrawal to reset balance to 0
    UPDATE wallets SET balance = balance - withdrawal_amount WHERE id = wallet_id;
    INSERT INTO transactions (from_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, withdrawal_amount, 'USD', 'WITHDRAWAL', 'Reset Bob''s USD wallet balance to 0');
END $$;

-- Deposit for Bob's HKD wallet
DO $$
DECLARE
    wallet_id BIGINT;
    deposit_amount NUMERIC(20, 4) := 800.00; -- Bob HKD 初始存款
BEGIN
    SELECT id INTO wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'bob') AND currency = 'HKD';
    UPDATE wallets SET balance = balance + deposit_amount WHERE id = wallet_id;
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, deposit_amount, 'HKD', 'DEPOSIT', 'Initial deposit to Bob''s HKD wallet');
END $$;

-- Deposit for Charlie's USD wallet
DO $$
DECLARE
    wallet_id BIGINT;
    deposit_amount NUMERIC(20, 4) := 300.00; -- Charlie USD 初始存款
BEGIN
    SELECT id INTO wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'charlie') AND currency = 'USD';
    UPDATE wallets SET balance = balance + deposit_amount WHERE id = wallet_id;
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, deposit_amount, 'USD', 'DEPOSIT', 'Initial deposit to Charlie''s USD wallet');
END $$;

-- Deposit for Charlie's HKD wallet
DO $$
DECLARE
    wallet_id BIGINT;
    deposit_amount NUMERIC(20, 4) := 700.00; -- Charlie HKD 初始存款
BEGIN
    SELECT id INTO wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'charlie') AND currency = 'HKD';
    UPDATE wallets SET balance = balance + deposit_amount WHERE id = wallet_id;
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (wallet_id, deposit_amount, 'HKD', 'DEPOSIT', 'Initial deposit to Charlie''s HKD wallet');
END $$;


-- --- Existing withdrawal and transfer operations ---

-- Simulate a withdrawal for Alice's USD wallet
DO $$
DECLARE
    alice_usd_wallet_id BIGINT;
BEGIN
    SELECT id INTO alice_usd_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'USD';

    -- Update Alice's balance
    UPDATE wallets
    SET balance = balance - 200.00
    WHERE id = alice_usd_wallet_id;

    -- Record the withdrawal transaction
    INSERT INTO transactions (from_wallet_id, amount, currency, type, description)
    VALUES (
        alice_usd_wallet_id,
        200.00,
        'USD',
        'WITHDRAWAL',
        'Withdrawal from Alice''s USD wallet'
    );
END $$;


-- Simulate a transfer from Alice's USD to Bob's USD
DO $$
DECLARE
    alice_usd_wallet_id BIGINT;
    bob_usd_wallet_id BIGINT;
    transfer_amount NUMERIC(20, 4) := 150.00;
BEGIN
    SELECT id INTO alice_usd_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'USD';
    SELECT id INTO bob_usd_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'bob') AND currency = 'USD';

    -- Deduct from Alice's balance
    UPDATE wallets
    SET balance = balance - transfer_amount
    WHERE id = alice_usd_wallet_id;

    -- Add to Bob's balance
    UPDATE wallets
    SET balance = balance + transfer_amount
    WHERE id = bob_usd_wallet_id;

    -- Record the transfer transaction
    INSERT INTO transactions (from_wallet_id, to_wallet_id, amount, currency, type, description)
    VALUES (
        alice_usd_wallet_id,
        bob_usd_wallet_id,
        transfer_amount,
        'USD',
        'TRANSFER',
        'Transfer from Alice USD to Bob USD'
    );
END $$;