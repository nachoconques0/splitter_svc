-- A Person outlives every Bill they appear on, so people is its own table rather
-- than a name copied onto each Share.
CREATE TABLE people (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    -- A name of spaces is the same thing as no name.
    CONSTRAINT people_name_is_not_blank CHECK (btrim(name) <> '')
);

CREATE TABLE bills (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    description text NOT NULL,

    -- Never a float and never a NUMERIC: exactness has to survive the hop into
    -- Go, and an integer is the only representation exact at every hop.
    total       bigint NOT NULL,
    currency    char(3) NOT NULL,

    -- Guards the replace against a stale client.
    version     bigint NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT bills_total_is_not_negative CHECK (total >= 0),
    CONSTRAINT bills_currency_is_iso_4217 CHECK (currency ~ '^[A-Z]{3}$')
);

-- A Share has no identity outside its Bill, so the Bill and Person it joins are
-- its key — which is also what stops the same Person appearing on one twice.
CREATE TABLE shares (
    bill_id    uuid NOT NULL REFERENCES bills (id) ON DELETE CASCADE,

    -- RESTRICT by default: a Person on a Bill cannot be deleted from under it.
    person_id  uuid NOT NULL REFERENCES people (id),

    -- Hundredths of a percent: 100.00% is 10000.
    percentage integer NOT NULL,

    PRIMARY KEY (bill_id, person_id),

    -- 0% is valid: it records that someone was there without charging them.
    CONSTRAINT shares_percentage_is_within_range CHECK (percentage BETWEEN 0 AND 10000)
);

-- Both columns hold an integer standing for a decimal, so the unit travels with
-- the schema and shows up beside the column in any client.
COMMENT ON COLUMN bills.total IS
    'The whole amount of the bill, in the smallest unit of its currency. 8500 with currency EUR is 85.00 EUR.';
COMMENT ON COLUMN shares.percentage IS
    'How much of the bill this share claims, in hundredths of a percent. 3333 is 33.33%; a bill''s shares sum to exactly 10000.';

-- One Bill, so the app is usable the first time it is opened. The identifiers
-- are fixed because the page shows one Bill and has to name it.
INSERT INTO people (id, name) VALUES
    ('a0000000-0000-4000-8000-000000000001', 'Ada Lovelace'),
    ('a0000000-0000-4000-8000-000000000002', 'Grace Hopper'),
    ('a0000000-0000-4000-8000-000000000003', 'Alan Turing'),
    -- On no Bill at all: a Person is not created by a Bill.
    ('a0000000-0000-4000-8000-000000000004', 'Katherine Johnson');

-- 85.00 EUR three ways. Deliberately does not divide evenly, so the seeded Bill
-- shows a cent being handed out rather than hiding it.
INSERT INTO bills (id, description, total, currency) VALUES
    ('b1110000-0000-4000-8000-000000000001', 'Dinner at Trattoria da Enzo', 8500, 'EUR');

-- 33.33 + 33.33 + 33.34 = exactly 100.00.
INSERT INTO shares (bill_id, person_id, percentage) VALUES
    ('b1110000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000001', 3333),
    ('b1110000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000002', 3333),
    ('b1110000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000003', 3334);
