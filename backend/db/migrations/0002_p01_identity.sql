-- P01-001..P01-006: identity challenges, one-time email verification and credentials.
CREATE TABLE IF NOT EXISTS auth_challenges (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL,
  purpose TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  turnstile_verified BOOLEAN NOT NULL DEFAULT FALSE,
  otp_verified BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS auth_challenges_email_idx ON auth_challenges (email, created_at DESC);

CREATE TABLE IF NOT EXISTS email_otps (
  id UUID PRIMARY KEY,
  challenge_id UUID NOT NULL REFERENCES auth_challenges(id),
  email TEXT NOT NULL,
  code_digest TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS password_credentials (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  password_hash TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  access_token_digest TEXT NOT NULL,
  refresh_token_digest TEXT NOT NULL UNIQUE,
  access_expires_at TIMESTAMPTZ NOT NULL,
  refresh_expires_at TIMESTAMPTZ NOT NULL,
  refresh_consumed_at TIMESTAMPTZ
);
