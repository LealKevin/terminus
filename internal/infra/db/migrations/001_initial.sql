-- Write your migrate up statements here
START TRANSACTION;
CREATE TABLE worlds (
  id UUID PRIMARY KEY,
  width INT NOT NULL,
  height INT NOT NULL,
  layout TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE players (
  id UUID PRIMARY KEY,
  world_id UUID REFERENCES worlds(id) ON DELETE CASCADE,
  x INT NOT NULL,
  y INT NOT NULL,
  health INT NOT NULL,
  attack INT NOT NULL,
  defense INT NOT NULL,
  range INT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mobs (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  world_id UUID REFERENCES worlds(id) ON DELETE CASCADE,
  x INT NOT NULL,
  y INT NOT NULL,
  type TEXT NOT NULL,
  health INT NOT NULL,
  attack INT NOT NULL,
  defense INT NOT NULL,
  attack_speed INT NOT NULL,
  symbol CHAR(1) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMIT;

---- create above / drop below ----
START TRANSACTION;
DROP TABLE IF EXISTS mobs;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS worlds;
COMMIT;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
