INSERT INTO vacant_chair (chair_id)
  SELECT chairs.id FROM chairs WHERE is_active = 1 ON CONFLICT DO NOTHING;
DELETE FROM vacant_chair WHERE chair_id IN (
  SELECT chairs.id FROM chairs INNER JOIN rides
  ON chairs.id = rides.chair_id
  INNER JOIN ride_statuses ON rides.id = ride_statuses.ride_id
  WHERE chair_sent_at IS NULL
);
