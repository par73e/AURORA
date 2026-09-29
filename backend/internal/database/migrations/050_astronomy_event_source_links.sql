-- The old aurora.local address was a placeholder and cannot serve model documentation.
UPDATE data_sources
SET base_url = 'https://github.com/par73e/AURORA/blob/main/backend/internal/observatory/calendar.go'
WHERE code = 'aurora_astronomy_model'
  AND base_url = 'https://aurora.local/astronomy-model';

UPDATE astronomy_events
SET source_url = 'https://github.com/par73e/AURORA/blob/main/backend/internal/observatory/calendar.go'
WHERE source_code = 'aurora_astronomy_model'
  AND source_url = 'https://aurora.local/astronomy-model';

-- Horizons API needs a COMMAND parameter and is not a readable event source page.
UPDATE astronomy_events
SET source_url = 'https://ssd.jpl.nasa.gov/horizons/app.html'
WHERE source_code = 'jpl_horizons_events'
  AND source_url = 'https://ssd.jpl.nasa.gov/api/horizons.api';
