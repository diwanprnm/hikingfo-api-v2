-- +goose Up
-- 009: demo gallery media rows referencing objects under the `demo/` key
-- namespace in the `photos` bucket (uploaded by the quickstart mc step).
-- Objects may legitimately be absent (honest placeholder tiles) — the rows are
-- still correct demo state, see research D5.
-- Fixed ids: deadbe09-0000-4000-8000-0000000000NN (60..65).

-- Guard first: referenced slugs must exist (mirrors 0013; never half-loads).
-- +goose StatementBegin
DO $$
DECLARE s text;
BEGIN
  FOREACH s IN ARRAY ARRAY['semeru','prau','agung'] LOOP
    IF NOT EXISTS (SELECT 1 FROM mountains WHERE slug = s) THEN
      RAISE EXCEPTION 'demo media seed: missing mountain slug %', s;
    END IF;
  END LOOP;
END $$;
-- +goose StatementEnd

INSERT INTO mountain_photos (id, mountain_id, photo_key, created_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000060', (SELECT id FROM mountains WHERE slug='semeru'), 'demo/ridge.jpg',   now() - interval '2 days'),
  ('deadbe09-0000-4000-8000-000000000061', (SELECT id FROM mountains WHERE slug='semeru'), 'demo/sunrise.jpg', now() - interval '2 days'),
  ('deadbe09-0000-4000-8000-000000000062', (SELECT id FROM mountains WHERE slug='semeru'), 'demo/crater.jpg',  now() - interval '2 days'),
  ('deadbe09-0000-4000-8000-000000000063', (SELECT id FROM mountains WHERE slug='prau'),   'demo/sunrise.jpg', now() - interval '5 days'),
  ('deadbe09-0000-4000-8000-000000000064', (SELECT id FROM mountains WHERE slug='prau'),   'demo/crater.jpg',  now() - interval '5 days'),
  ('deadbe09-0000-4000-8000-000000000065', (SELECT id FROM mountains WHERE slug='agung'),  'demo/crater.jpg',  now() - interval '7 days')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM mountain_photos
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000060','deadbe09-0000-4000-8000-000000000061',
   'deadbe09-0000-4000-8000-000000000062','deadbe09-0000-4000-8000-000000000063',
   'deadbe09-0000-4000-8000-000000000064','deadbe09-0000-4000-8000-000000000065');
