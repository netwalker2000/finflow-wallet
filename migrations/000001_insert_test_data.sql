-- Insert test users
INSERT INTO users (username) VALUES ('alice');
INSERT INTO users (username) VALUES ('bob');
INSERT INTO users (username) VALUES ('charlie');

-- Insert wallets for users
-- Alice's USD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'alice'),
    'USD',
    10.00
);

-- Bob's USD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'bob'),
    'USD',
    10.00
);

-- Charlie's USD wallet (for potential future transfers/deposits)
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'charlie'),
    'USD',
    10.00
);

-- Alice's HKD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'alice'),
    'HKD',
    100.00
);

-- Bob's HKD wallet
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'bob'),
    'HKD',
    200.00
);

-- Charlie's USD wallet (for potential future transfers/deposits)
INSERT INTO wallets (user_id, currency, balance)
VALUES (
    (SELECT id FROM users WHERE username = 'charlie'),
    'HKD',
    300.00
);

-- Simulate a deposit for Alice
DO $$
DECLARE
    alice_wallet_id BIGINT;
BEGIN
    SELECT id INTO alice_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'USD';

    -- Update Alice's balance
    UPDATE wallets
    SET balance = balance + 1000.00
    WHERE id = alice_wallet_id;

    -- Record the deposit transaction
    INSERT INTO transactions (to_wallet_id, amount, currency, type, description)
    VALUES (
        alice_wallet_id,
        1000.00,
        'USD',
        'DEPOSIT',
        'Initial deposit to Alice''s wallet'
    );
END $$;


-- Simulate a withdrawal for Alice
DO $$
DECLARE
    alice_wallet_id BIGINT;
BEGIN
    SELECT id INTO alice_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'USD';

    -- Update Alice's balance
    UPDATE wallets
    SET balance = balance - 200.00
    WHERE id = alice_wallet_id;

    -- Record the withdrawal transaction
    INSERT INTO transactions (from_wallet_id, amount, currency, type, description)
    VALUES (
        alice_wallet_id,
        200.00,
        'USD',
        'WITHDRAWAL',
        'Withdrawal from Alice''s wallet'
    );
END $$;


-- Simulate a transfer from Alice to Bob
DO $$
DECLARE
    alice_wallet_id BIGINT;
    bob_wallet_id BIGINT;
    transfer_amount NUMERIC(24, 8) := 150.00; -- Use NUMERIC(24,8) for consistency in declaration
BEGIN
    SELECT id INTO alice_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'alice') AND currency = 'USD';
    SELECT id INTO bob_wallet_id FROM wallets WHERE user_id = (SELECT id FROM users WHERE username = 'bob') AND currency = 'USD';

    -- Deduct from Alice's balance
    UPDATE wallets
    SET balance = balance - transfer_amount
    WHERE id = alice_wallet_id;

    -- Add to Bob's balance
    UPDATE wallets
    SET balance = balance + transfer_amount
    WHERE id = bob_wallet_id;

    -- Record the transfer transaction
    INSERT INTO transactions (from_wallet_id, to_wallet_id, amount, currency, type, description)
    VALUES (
        alice_wallet_id,
        bob_wallet_id,
        transfer_amount,
        'USD',
        'TRANSFER',
        'Transfer from Alice to Bob'
    );
END $$;