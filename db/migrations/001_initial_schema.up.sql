CREATE TABLE campaigns (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image_url TEXT,
    cta TEXT,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')) DEFAULT 'INACTIVE'
);

CREATE TABLE targeting_rules (
    id SERIAL PRIMARY KEY,
    campaign_id VARCHAR(50) REFERENCES campaigns(id) ON DELETE CASCADE,
    dimension TEXT NOT NULL CHECK (dimension IN ('Country', 'OS', 'AppID')),
    is_inclusion BOOLEAN NOT NULL DEFAULT TRUE,
    values TEXT[] NOT NULL,
    UNIQUE(campaign_id, dimension)
);

CREATE OR REPLACE FUNCTION notify_data_change()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('data_changed', TG_TABLE_NAME || ', id: ' || COALESCE(NEW.id, OLD.id)::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER campaigns_notify_change
AFTER INSERT OR UPDATE OR DELETE ON campaigns
FOR EACH ROW EXECUTE PROCEDURE notify_data_change();

CREATE TRIGGER targeting_rules_notify_change
AFTER INSERT OR UPDATE OR DELETE ON targeting_rules
FOR EACH ROW EXECUTE PROCEDURE notify_data_change();

INSERT INTO campaigns (id, name, image_url, cta, status) VALUES
    ('spotify', 'Spotify - Music for everyone', 'https://somelink', 'Download', 'ACTIVE'),
    ('duolingo', 'Duolingo: Best way to learn', 'https://somelink2', 'Install', 'ACTIVE'),
    ('subwaysurfer', 'Subway Surfer', 'https://somelink3', 'Play', 'ACTIVE');

INSERT INTO targeting_rules (campaign_id, dimension, is_inclusion, values) VALUES
    ('spotify', 'Country', true, ARRAY['US', 'Canada']),
    ('duolingo', 'OS', true, ARRAY['Android', 'iOS']),
    ('duolingo', 'Country', false, ARRAY['US']),
    ('subwaysurfer', 'OS', true, ARRAY['Android']),
    ('subwaysurfer', 'AppID', true, ARRAY['com.gametion.ludokinggame']);