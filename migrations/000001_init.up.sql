CREATE TABLE IF NOT EXISTS users (
    id              SERIAL           PRIMARY KEY,
    nickname        VARCHAR(100)     UNIQUE         NOT NULL,
    room_id         INTEGER                         NOT NULL,
    created_at      TIMESTAMPTZ                     NOT NULL  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rooms (
    id              SERIAL          PRIMARY KEY,
    name            VARCHAR(100)    UNIQUE          NOT NULL,
    created_at      TIMESTAMPTZ                     NOT NULL  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id              SERIAL          PRIMARY KEY,
    user_id         INTEGER                     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id         INTEGER                     NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    message         VARCHAR(1000)               NOT NULL,
    created_at      TIMESTAMPTZ                 NOT NULL
);