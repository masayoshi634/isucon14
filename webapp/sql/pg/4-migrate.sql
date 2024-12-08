INSERT INTO vacant_chair (chair_id)
  SELECT chairs.id FROM chairs WHERE is_active = 1 ON CONFLICT DO NOTHING;
DELETE FROM vacant_chair WHERE chair_id IN (
  SELECT chairs.id FROM chairs INNER JOIN rides
  ON chairs.id = rides.chair_id
  INNER JOIN ride_statuses ON rides.id = ride_statuses.ride_id
  WHERE chair_sent_at IS NULL
);

INSERT INTO chair_locations_summary (chair_id, total_distance, total_distance_updated_at)
  SELECT id AS chair_id,
         COALESCE(total_distance, 0) AS total_distance,
         COALESCE(total_distance_updated_at, CURRENT_TIMESTAMP(6)) AS total_distance_updated_at
  FROM chairs
         LEFT JOIN (SELECT chair_id,
                            SUM(COALESCE(distance, 0)) AS total_distance,
                            MAX(created_at)          AS total_distance_updated_at
                     FROM (SELECT chair_id,
                                  created_at,
                                  ABS(latitude - LAG(latitude) OVER (PARTITION BY chair_id ORDER BY created_at)) +
                                 ABS(longitude - LAG(longitude) OVER (PARTITION BY chair_id ORDER BY created_at)) AS distance
                           FROM chair_locations) tmp
                     GROUP BY chair_id) distance_table ON distance_table.chair_id = chairs.id;
