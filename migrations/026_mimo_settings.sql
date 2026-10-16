CREATE TABLE mimo_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    preferred_billing_mode TEXT NOT NULL CHECK (preferred_billing_mode IN ('token_plan', 'payg'))
);

INSERT INTO mimo_settings(id, preferred_billing_mode) VALUES (1, 'token_plan');
