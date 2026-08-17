-- Модель дома для X-Ray AR: геометрия + реперные точки (якоря) + зоны труб/кабелей.

CREATE TABLE IF NOT EXISTS home_mesh (
    owner_id   text PRIMARY KEY,
    mesh       jsonb NOT NULL DEFAULT '{}'::jsonb,   -- 3D-геометрия дома
    anchors    jsonb NOT NULL DEFAULT '[]'::jsonb,  -- реперные точки (розетки, рамы, решётки)
    zones      jsonb NOT NULL DEFAULT '[]'::jsonb,  -- сегменты труб/кабелей с device_id
    updated_at timestamptz NOT NULL DEFAULT now()
);
