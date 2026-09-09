-- +goose Up
-- 009: demonstration community data so every feature area shows content on a
-- fresh demo environment (the 005 gap: journeys/hikes/partner/notifications were
-- specced but never seeded).
--
-- DEV/STAGING DEMO ONLY — never apply to a production community database.
-- All 5 demo users share the 0012 admin password hash: "Admin123!" (rotate/remove
-- before any public deploy). Idempotent via fixed ids + ON CONFLICT DO NOTHING;
-- reversible via the Down section (deletes by the fixed-id list only).
--
-- Fixed-id register: deadbe09-0000-4000-8000-0000000000NN
--   users 01..05 · contacts 100,101 · posts 10..15 · hikes 20..25
--   notices 30..33 · requests 40..44 · notifications 50..56
-- Mountain FKs resolve by 0011 slug (semeru, prau, agung, lawu, sindoro).

-- ---------- demo users ----------
INSERT INTO users (id, email, password_hash, email_verified_at, display_name, bio, home_region, role, status, created_at, updated_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000001', 'rina@example.hikingfo.test',
   '$argon2id$v=19$m=65536,t=1,p=4$lXcK0XohYdD1k0Oj6Zvj+Q$Ptwhb/LSUOp36qcEFaoyBnEKlUKzlxONk1dilf7q17c',
   now(), 'Rina · contoh',
   '{"id":"Pendaki akhir pekan, suka jalur hutan.","en":"Weekend hiker who loves forest trails."}',
   'Jawa Timur', 'member', 'active', now(), now()),
  ('deadbe09-0000-4000-8000-000000000002', 'bayu@example.hikingfo.test',
   '$argon2id$v=19$m=65536,t=1,p=4$lXcK0XohYdD1k0Oj6Zvj+Q$Ptwhb/LSUOp36qcEFaoyBnEKlUKzlxONk1dilf7q17c',
   now(), 'Bayu · contoh',
   '{"id":"Pencari teman naik gunung.","en":"Looking for hiking partners."}',
   'Jawa Tengah', 'member', 'active', now(), now()),
  ('deadbe09-0000-4000-8000-000000000003', 'siti@example.hikingfo.test',
   '$argon2id$v=19$m=65536,t=1,p=4$lXcK0XohYdD1k0Oj6Zvj+Q$Ptwhb/LSUOp36qcEFaoyBnEKlUKzlxONk1dilf7q17c',
   now(), 'Siti · contoh',
   '{"id":"Suka fotografi lanskap.","en":"Landscape photography fan."}',
   'Jawa Barat', 'member', 'active', now(), now()),
  ('deadbe09-0000-4000-8000-000000000004', 'agus@example.hikingfo.test',
   '$argon2id$v=19$m=65536,t=1,p=4$lXcK0XohYdD1k0Oj6Zvj+Q$Ptwhb/LSUOp36qcEFaoyBnEKlUKzlxONk1dilf7q17c',
   now(), 'Agus W · contoh',
   '{"id":"Pendaki pemula, masih belajar.","en":"Beginner hiker, still learning."}',
   'DI Yogyakarta', 'member', 'active', now(), now()),
  ('deadbe09-0000-4000-8000-000000000005', 'dewi@example.hikingfo.test',
   '$argon2id$v=19$m=65536,t=1,p=4$lXcK0XohYdD1k0Oj6Zvj+Q$Ptwhb/LSUOp36qcEFaoyBnEKlUKzlxONk1dilf7q17c',
   now(), 'Dewi · contoh',
   '{"id":"Penggiat alam bebas.","en":"Outdoors enthusiast."}',
   'Jawa Timur', 'member', 'active', now(), now())
ON CONFLICT (id) DO NOTHING;

-- Private contacts on two demo accounts (Principle I negative-test fixture:
-- present in DB, must never surface in any public/search feed).
INSERT INTO user_contacts (user_id, phone, whatsapp, instagram)
VALUES
  ('deadbe09-0000-4000-8000-000000000001', '081234567890', '+6281234567890', 'demo_only'),
  ('deadbe09-0000-4000-8000-000000000002', '089876543210', '+6289876543210', 'demo_only')
ON CONFLICT (user_id) DO NOTHING;

-- ---------- precondition guard (research D2 / data-model) ----------
-- Fail loud, never half-load: if the catalogue is not seeded, name the slug.
-- +goose StatementBegin
DO $$
DECLARE s text;
BEGIN
  FOREACH s IN ARRAY ARRAY['semeru','prau','agung','lawu','sindoro'] LOOP
    IF NOT EXISTS (SELECT 1 FROM mountains WHERE slug = s) THEN
      RAISE EXCEPTION 'demo seed: missing mountain slug %', s;
    END IF;
  END LOOP;
END $$;
-- +goose StatementEnd

-- ---------- journey posts (6; 2 hidden to prove the feed filters) ----------
INSERT INTO journey_posts (id, user_id, mountain_id, route_id, title, summary, narrative, photo_keys, visibility, moderation_status, published_at, updated_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000010', 'deadbe09-0000-4000-8000-000000000001',
   (SELECT id FROM mountains WHERE slug='semeru'),
   (SELECT id FROM mountain_routes WHERE mountain_id=(SELECT id FROM mountains WHERE slug='semeru') ORDER BY id LIMIT 1),
   'Pendakian Semeru (contoh)',
   '{"id":"Mahameru di musim kemarau, dua hari satu malam.","en":"Mahameru in the dry season, two days one night."}',
   '{"id":"Jalur Ranu Kumbolo malam sangat dingin, tapi sunrise di Penanjakan sepadan. Catatan: bawa air lebih.","en":"The Ranu Kumbolo trail was freezing at night, but sunrise at Penanjakan was worth it. Note: carry extra water."}',
   '{demo/sunrise.jpg}', 'published', 'visible', now() - interval '1 day', now()),
  ('deadbe09-0000-4000-8000-000000000011', 'deadbe09-0000-4000-8000-000000000003',
   (SELECT id FROM mountains WHERE slug='prau'), NULL,
   'Prau dari Tambak (contoh)',
   '{"id":"Malam di Pos 3 ramai pendaki, sunrise dari Puncak Prau jelas.","en":"Night at Pos 3 busy with hikers, clear sunrise from Prau summit."}',
   '{"id":"Pendakian singkat tiga jam ke puncak, cocok untuk pemula yang ingin melihat sunrise.","en":"A short three-hour climb to the summit, good for beginners wanting a sunrise view."}',
   '{demo/ridge.jpg}', 'published', 'visible', now() - interval '3 days', now()),
  ('deadbe09-0000-4000-8000-000000000012', 'deadbe09-0000-4000-8000-000000000002',
   (SELECT id FROM mountains WHERE slug='agung'), NULL,
   'Agung via Pura (contoh)',
   '{"id":"Jalur pura sunyi, kabut tebal di subuh.","en":"Quiet temple route, thick fog at dawn."}',
   '{"id":"Naik lewat jalur Pura Pasar Agung, kabut menutup kawah sampai pukul enam. Buta arah di beberapa tikungan, GPS membantu.","en":"Climbed via the Pura Pasar Agung route; fog hid the crater until six. Some turns were disorienting, GPS helped."}',
   '{demo/crater.jpg,demo/sunrise.jpg}', 'published', 'visible', now() - interval '6 days', now()),
  ('deadbe09-0000-4000-8000-000000000013', 'deadbe09-0000-4000-8000-000000000001',
   (SELECT id FROM mountains WHERE slug='lawu'), NULL,
   'Lawu via Stren (contoh)',
   '{"id":"Tritik curug sepanjang jalur, tanah gambut basah.","en":"Waterfalls along the route, wet peat soil."}',
   '{"id":"Jalur Stren banyak air terjun kecil. Hati-hati akar licin setelah hujan semalam.","en":"The Stren route has many small waterfalls. Watch for slippery roots after last night''s rain."}',
   '{}', 'published', 'visible', now() - interval '10 days', now()),
  -- hidden from the feed (moderation filter): published but under_review
  ('deadbe09-0000-4000-8000-000000000014', 'deadbe09-0000-4000-8000-000000000003',
   (SELECT id FROM mountains WHERE slug='sindoro'), NULL,
   'Sindoro via Apitan (contoh)',
   '{"id":"Diblok untuk peninjauan moderasi (data contoh).","en":"Blocked for moderation review (sample data)."}',
   '{"id":"Laporan ini ada untuk menguji filter moderasi feed.","en":"This post exists to exercise the feed moderation filter."}',
   '{}', 'published', 'under_review', now() - interval '5 days', now()),
  -- hidden from the feed (visibility filter): draft by a demo author
  ('deadbe09-0000-4000-8000-000000000015', 'deadbe09-0000-4000-8000-000000000004',
   (SELECT id FROM mountains WHERE slug='semeru'), NULL,
   'Draf pribadi (contoh)',
   '{"id":"Masih draf, tidak boleh muncul di feed.","en":"Still a draft, must not appear in the feed."}',
   '{"id":"Catatan draft untuk pengujian visibilitas.","en":"Draft note for the visibility test."}',
   '{}', 'draft', 'visible', NULL, now())
ON CONFLICT (id) DO NOTHING;

-- ---------- hike records (6: 4 verified-with-evidence, 2 unverified) ----------
-- Principle III: verified status only where evidence_photo_keys is non-empty.
INSERT INTO hike_log_entries (id, user_id, mountain_id, route_id, climb_date, evidence_photo_keys, status, published_post_id, created_at, updated_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000020', 'deadbe09-0000-4000-8000-000000000001',
   (SELECT id FROM mountains WHERE slug='semeru'), NULL, (current_date - 2),
   '{demo/sunrise.jpg}', 'verified',
   'deadbe09-0000-4000-8000-000000000010', now() - interval '1 day', now()),
  ('deadbe09-0000-4000-8000-000000000021', 'deadbe09-0000-4000-8000-000000000001',
   (SELECT id FROM mountains WHERE slug='lawu'), NULL, (current_date - 12),
   '{demo/ridge.jpg}', 'verified', NULL, now() - interval '11 days', now()),
  ('deadbe09-0000-4000-8000-000000000022', 'deadbe09-0000-4000-8000-000000000001',
   (SELECT id FROM mountains WHERE slug='prau'), NULL, (current_date - 20),
   '{}', 'unverified', NULL, now() - interval '19 days', now()),
  ('deadbe09-0000-4000-8000-000000000023', 'deadbe09-0000-4000-8000-000000000003',
   (SELECT id FROM mountains WHERE slug='prau'), NULL, (current_date - 6),
   '{demo/ridge.jpg}', 'verified', NULL, now() - interval '5 days', now()),
  ('deadbe09-0000-4000-8000-000000000024', 'deadbe09-0000-4000-8000-000000000003',
   (SELECT id FROM mountains WHERE slug='sindoro'), NULL, (current_date - 15),
   '{}', 'unverified', NULL, now() - interval '14 days', now()),
  ('deadbe09-0000-4000-8000-000000000025', 'deadbe09-0000-4000-8000-000000000002',
   (SELECT id FROM mountains WHERE slug='agung'), NULL, (current_date - 8),
   '{demo/crater.jpg}', 'verified', NULL, now() - interval '7 days', now())
ON CONFLICT (id) DO NOTHING;

-- ---------- partner notices (4 open/matched; runtime-relative windows) ----------
INSERT INTO partner_notices (id, user_id, mountain_id, trip_start, trip_end, note, status, expires_at, created_at, updated_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000030', 'deadbe09-0000-4000-8000-000000000001',
   (SELECT id FROM mountains WHERE slug='semeru'),
   current_date + 7, current_date + 9, 'Cari partner naik, budget aman.', 'open',
   now() + interval '45 days', now(), now()),
  ('deadbe09-0000-4000-8000-000000000031', 'deadbe09-0000-4000-8000-000000000003',
   (SELECT id FROM mountains WHERE slug='prau'),
   current_date + 14, current_date + 16, 'Looking for one more for Prau.', 'open',
   now() + interval '45 days', now(), now()),
  ('deadbe09-0000-4000-8000-000000000032', 'deadbe09-0000-4000-8000-000000000002',
   (SELECT id FROM mountains WHERE slug='agung'),
   current_date + 21, current_date + 23, 'Rombongan Agung masih butuh 2 orang.', 'open',
   now() + interval '45 days', now(), now()),
  -- matched notice (paired with accepted request 41)
  ('deadbe09-0000-4000-8000-000000000033', 'deadbe09-0000-4000-8000-000000000004',
   (SELECT id FROM mountains WHERE slug='lawu'),
   current_date + 30, current_date + 32, 'Lawu santai, sudah ada partner.', 'matched',
   now() + interval '45 days', now(), now())
ON CONFLICT (id) DO NOTHING;

-- ---------- partner requests (5 across states) ----------
INSERT INTO partner_requests (id, from_user_id, to_user_id, notice_id, mountain_id, trip_start, trip_end, message, status, matched_at, expires_at, created_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000040', 'deadbe09-0000-4000-8000-000000000004',
   'deadbe09-0000-4000-8000-000000000001',
   'deadbe09-0000-4000-8000-000000000030',
   (SELECT id FROM mountains WHERE slug='semeru'),
   current_date + 7, current_date + 9, 'Halo, boleh gabung?', 'pending', NULL,
   now() + interval '45 days', now()),
  ('deadbe09-0000-4000-8000-000000000041', 'deadbe09-0000-4000-8000-000000000002',
   'deadbe09-0000-4000-8000-000000000004',
   'deadbe09-0000-4000-8000-000000000033',
   (SELECT id FROM mountains WHERE slug='lawu'),
   current_date + 30, current_date + 32, 'Serius ikut ke Lawu.', 'accepted',
   now() - interval '2 days', now() + interval '45 days', now() - interval '3 days'),
  ('deadbe09-0000-4000-8000-000000000042', 'deadbe09-0000-4000-8000-000000000003',
   'deadbe09-0000-4000-8000-000000000002',
   'deadbe09-0000-4000-8000-000000000032',
   (SELECT id FROM mountains WHERE slug='agung'),
   current_date + 21, current_date + 23, 'Sisa kursi buat Agung?', 'declined', NULL,
   now() + interval '45 days', now()),
  ('deadbe09-0000-4000-8000-000000000043', 'deadbe09-0000-4000-8000-000000000001',
   'deadbe09-0000-4000-8000-000000000003',
   'deadbe09-0000-4000-8000-000000000031',
   (SELECT id FROM mountains WHERE slug='prau'),
   current_date + 14, current_date + 16, 'Mau join tim Prau.', 'pending', NULL,
   now() + interval '45 days', now()),
  ('deadbe09-0000-4000-8000-000000000044', 'deadbe09-0000-4000-8000-000000000005',
   'deadbe09-0000-4000-8000-000000000003',
   'deadbe09-0000-4000-8000-000000000031',
   (SELECT id FROM mountains WHERE slug='prau'),
   current_date + 14, current_date + 16, 'Niat ikut, tapi mundur.', 'withdrawn', NULL,
   now() + interval '45 days', now())
ON CONFLICT (id) DO NOTHING;

-- ---------- notifications (7; rina has 3 incl. unread) ----------
INSERT INTO notifications (id, user_id, type, payload, read_at, created_at)
VALUES
  ('deadbe09-0000-4000-8000-000000000050', 'deadbe09-0000-4000-8000-000000000001',
   'partner_request_received',
   '{"id":"Agus minta gabung ke Semeru.","en":"Agus asked to join you on Semeru.","request_id":"deadbe09-0000-4000-8000-000000000040"}',
   NULL, now() - interval '4 hours'),
  ('deadbe09-0000-4000-8000-000000000051', 'deadbe09-0000-4000-8000-000000000001',
   'partner_request_accepted',
   '{"id":"Siti menerima permintaan Prau kamu.","en":"Siti accepted your Prau request.","request_id":"deadbe09-0000-4000-8000-000000000043"}',
   now() - interval '2 days', now() - interval '3 days'),
  ('deadbe09-0000-4000-8000-000000000052', 'deadbe09-0000-4000-8000-000000000001',
   'badge_earned',
   '{"id":"Badge baru: Lawu terverifikasi.","en":"New badge: Lawu verified.","mountain_slug":"lawu"}',
   NULL, now() - interval '11 days'),
  ('deadbe09-0000-4000-8000-000000000053', 'deadbe09-0000-4000-8000-000000000003',
   'partner_request_received',
   '{"id":"Bayu kirim permintaan ke Prau.","en":"Bayu sent a Prau request.","request_id":"deadbe09-0000-4000-8000-000000000042"}',
   NULL, now() - interval '6 hours'),
  ('deadbe09-0000-4000-8000-000000000054', 'deadbe09-0000-4000-8000-000000000002',
   'badge_earned',
   '{"id":"Badge baru: Agung terverifikasi.","en":"New badge: Agung verified.","mountain_slug":"agung"}',
   now() - interval '5 days', now() - interval '7 days'),
  ('deadbe09-0000-4000-8000-000000000055', 'deadbe09-0000-4000-8000-000000000004',
   'partner_request_accepted',
   '{"id":"Bayu menerima permintaan Lawu.","en":"Bayu accepted the Lawu request.","request_id":"deadbe09-0000-4000-8000-000000000041"}',
   NULL, now() - interval '2 days'),
  ('deadbe09-0000-4000-8000-000000000056', 'deadbe09-0000-4000-8000-000000000005',
   'moderation_outcome',
   '{"id":"Laporan sindoro ditinjau moderasi.","en":"Sindoro report is under moderation review.","post_id":"deadbe09-0000-4000-8000-000000000014"}',
   NULL, now() - interval '5 days')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
-- Surgical removal by fixed ids only (never the email predicate, so a real user
-- could not theoretically share one). FK-safe child-first order.
DELETE FROM notifications
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000050','deadbe09-0000-4000-8000-000000000051',
   'deadbe09-0000-4000-8000-000000000052','deadbe09-0000-4000-8000-000000000053',
   'deadbe09-0000-4000-8000-000000000054','deadbe09-0000-4000-8000-000000000055',
   'deadbe09-0000-4000-8000-000000000056');
DELETE FROM partner_requests
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000040','deadbe09-0000-4000-8000-000000000041',
   'deadbe09-0000-4000-8000-000000000042','deadbe09-0000-4000-8000-000000000043',
   'deadbe09-0000-4000-8000-000000000044');
DELETE FROM partner_notices
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000030','deadbe09-0000-4000-8000-000000000031',
   'deadbe09-0000-4000-8000-000000000032','deadbe09-0000-4000-8000-000000000033');
DELETE FROM journey_posts
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000010','deadbe09-0000-4000-8000-000000000011',
   'deadbe09-0000-4000-8000-000000000012','deadbe09-0000-4000-8000-000000000013',
   'deadbe09-0000-4000-8000-000000000014','deadbe09-0000-4000-8000-000000000015');
DELETE FROM hike_log_entries
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000020','deadbe09-0000-4000-8000-000000000021',
   'deadbe09-0000-4000-8000-000000000022','deadbe09-0000-4000-8000-000000000023',
   'deadbe09-0000-4000-8000-000000000024','deadbe09-0000-4000-8000-000000000025');
DELETE FROM user_contacts
 WHERE user_id IN (
   'deadbe09-0000-4000-8000-000000000001','deadbe09-0000-4000-8000-000000000002');
DELETE FROM users
 WHERE id IN (
   'deadbe09-0000-4000-8000-000000000001','deadbe09-0000-4000-8000-000000000002',
   'deadbe09-0000-4000-8000-000000000003','deadbe09-0000-4000-8000-000000000004',
   'deadbe09-0000-4000-8000-000000000005');
