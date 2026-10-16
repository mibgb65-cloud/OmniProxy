CREATE TABLE stepfun_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    preferred_billing_mode TEXT NOT NULL CHECK (preferred_billing_mode IN ('step_plan', 'payg'))
);

INSERT INTO stepfun_settings(id, preferred_billing_mode) VALUES (1, 'step_plan');
