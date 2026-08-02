ALTER TABLE launch_events
    ADD COLUMN name_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN status_name_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN pad_name_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN location_name_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN mission_name_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN mission_type_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN mission_description_zh TEXT NOT NULL DEFAULT '',
    ADD COLUMN has_original BOOLEAN NOT NULL DEFAULT FALSE;
