-- Create keys table
CREATE TABLE IF NOT EXISTS keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_code VARCHAR(50) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_keys_reservation_id ON keys(reservation_id);
CREATE INDEX IF NOT EXISTS idx_keys_user_id ON keys(user_id);
CREATE INDEX IF NOT EXISTS idx_keys_valid_from ON keys(valid_from);
CREATE INDEX IF NOT EXISTS idx_keys_valid_until ON keys(valid_until);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_keys_updated_at BEFORE UPDATE ON keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

