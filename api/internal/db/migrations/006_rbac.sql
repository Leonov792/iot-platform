-- RBAC: роли, привязка к «дому» и расписание доступа персонала.

-- role: owner | family | staff
ALTER TABLE users ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'owner';

-- home_id — id владельца «дома». у owner совпадает с собственным id;
-- family/staff указывают на владельца, к чьим устройствам имеют доступ.
ALTER TABLE users ADD COLUMN IF NOT EXISTS home_id text;

-- schedule — окна доступа персонала (jsonb), напр.:
--   [{"zone":"pool","days":[1,2,3,4,5],"start":"08:00","end":"20:00"}]
ALTER TABLE users ADD COLUMN IF NOT EXISTS schedule jsonb NOT NULL DEFAULT '[]'::jsonb;

-- у уже созданных владельцев home_id = собственный id
UPDATE users SET home_id = id WHERE home_id IS NULL;
