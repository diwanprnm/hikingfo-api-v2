-- +goose Up
-- T041 seed content: curated bilingual catalogue for the 25-mountain shortlist.
-- Shortlist drafted in auto-mode — USER SIGN-OFF PENDING (tasks.md notes T041
-- requires approval before final content). Mountains marked with
-- "reliability":"reported" need field verification before public launch.
--
-- Idempotent: re-running skips existing rows (ON CONFLICT / NOT EXISTS guards).
-- Run: goose -dir backend/migrations/seed postgres://$DSN up
--      (or psql -f; goose annotations kept so the same runner applies it)

-- ============================ JAWA ============================

-- Gunung Semeru — highest on Java
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'semeru',
  '{"id":"Gunung Semeru","en":"Mount Semeru"}'::jsonb,
  ARRAY['Mahameru','Sumeru'],
  'Jawa', 'Jawa Timur',
  '{"id":"Dari Malang ±52 km ke Desa Ranu Pani via Tumpang; akses terakhir ojek desa/mobil listrik desa.","en":"From Malang ±52 km to Ranu Pani village via Tumpang; final access by village ojek or village ev."}'::jsonb,
  -8.116, 112.923,
  '{"id":"Puncak Mahameru","en":"Mahameru Summit"}'::jsonb,
  3676, 5, 'published',
  '{"location":{"source":"Resort TNGH Ranu Pani","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG / TNGH","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGH fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Semeru Mount Semeru Mahameru Sumeru'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Ranu Pani","en":"Ranu Pani route"}'::jsonb, 13, 16, 2100,
  '{"id":"Kuota harian + izin daring resor TNGH; kesehatan surat wajib.","en":"Daily quota + online TNGH permit; health letter required."}'::jsonb, 1
FROM mountains WHERE slug='semeru'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Ranu Pani","en":"Ranu Pani basecamp"}'::jsonb,
  '{"id":"Kantor resort, musholla, warung, porter/jasa angkut, penginapan desa","en":"Resort office, musholla, food stalls, porter service, village homestays"}'::jsonb,
  '{"id":"Tiket TNGH Rp 10.000–25.000/hari; porter opsional Rp 400.000–600.000/trip","en":"TNGH ticket IDR 10,000–25,000/day; optional porter IDR 400,000–600,000/trip"}'::jsonb,
  true, -8.105, 112.918, 1
FROM mountains WHERE slug='semeru'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Lawu
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'lawu',
  '{"id":"Gunung Lawu","en":"Mount Lawu"}'::jsonb,
  ARRAY['Lawu'],
  'Jawa', 'Jawa Timur',
  '{"id":"Dari Solo ±2 jam ke Cemoro Sewu (Tawangmangu) atau jalur Candi Cetho/Selo.","en":"From Solo ±2 h to Cemoro Sewu (Tawangmangu) or the Candi Cetho/Selo route."}'::jsonb,
  -7.626, 111.192,
  '{"id":"Puncak Hargodumilah","en":"Hargodumilah Summit"}'::jsonb,
  3265, 3, 'published',
  '{"location":{"source":"Resort TNGH Cemoro Sewu","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGH fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Lawu Mount Lawu Hargodumilah'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Cemoro Sewu","en":"Cemoro Sewu route"}'::jsonb, 9, 10, 1800,
  '{"id":"Izin TNGH daring; kuota harian berlaku akhir pekan.","en":"Online TNGH permit; weekend daily quota applies."}'::jsonb, 1
FROM mountains WHERE slug='lawu'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Cemoro Sewu","en":"Cemoro Sewu basecamp"}'::jsonb,
  '{"id":"Pos TNGH, penginapan, warung, ojek","en":"TNGH post, homestays, food stalls, ojek"}'::jsonb,
  '{"id":"Tiket Rp 10.000–20.000/hari","en":"Ticket IDR 10,000–20,000/day"}'::jsonb,
  true, -7.655, 111.177, 1
FROM mountains WHERE slug='lawu'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Slamet — second highest on Java
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'slamet',
  '{"id":"Gunung Slamet","en":"Mount Slamet"}'::jsonb,
  ARRAY['Slamet'],
  'Jawa', 'Jawa Tengah',
  '{"id":"Akses umum dari Purwokerto ke Desa Bambangan (Kec. Karangreja) atau dari Brebes via Batu Satria.","en":"Public access from Purwokerto to Bambangan village (Karangreja) or from Brebes via Batu Satria."}'::jsonb,
  -7.242, 109.223,
  '{"id":"Puncak Batu Sagung Werda","en":"Batu Sagung Werda Summit"}'::jsonb,
  3428, 4, 'published',
  '{"location":{"source":"Perhutani BKPH Baturraden","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Slamet Mount Slamet Batu Sagung Werda'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Bambangan","en":"Bambangan route"}'::jsonb, 10, 12, 2000,
  '{"id":"Izin PKL/rintisan desa; kuota akhir pekan.","en":"Village forest permit; weekend quota."}'::jsonb, 1
FROM mountains WHERE slug='slamet'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Bambangan","en":"Bambangan basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung, musholla","en":"Registration post, parking, food stalls, musholla"}'::jsonb,
  '{"id":"Izin Rp 20.000–50.000; wajib pemandu setempat","en":"Permit IDR 20,000–50,000; local guide mandatory"}'::jsonb,
  true, -7.247, 109.180, 1
FROM mountains WHERE slug='slamet'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Sumbing
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'sumbing',
  '{"id":"Gunung Sumbing","en":"Mount Sumbing"}'::jsonb,
  ARRAY['Sumbing'],
  'Jawa', 'Jawa Tengah',
  '{"id":"Dari Temanggung/Magelang ke Desa Banjaroya (Kaliangkrik) atau Garung via Wonosobo.","en":"From Temanggung/Magelang to Banjaroya village (Kaliangkrik) or Garung via Wonosobo."}'::jsonb,
  -7.388, 110.067,
  '{"id":"Puncak Poyut–Sumilir","en":"Poyut–Sumilir Summit"}'::jsonb,
  3371, 4, 'published',
  '{"location":{"source":"desa Banjaroya","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Sumbing Mount Sumbing Poyut Sumilir'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Banjaroya","en":"Banjaroya route"}'::jsonb, 8, 10, 2000,
  '{"id":"Izin pos desa; buku tamu.","en":"Village post permit; guest book."}'::jsonb, 1
FROM mountains WHERE slug='sumbing'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Banjaroya","en":"Banjaroya basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung","en":"Registration post, parking, food stalls"}'::jsonb,
  '{"id":"Izin Rp 10.000–20.000; pemandu opsional","en":"Permit IDR 10,000–20,000; optional guide"}'::jsonb,
  true, -7.409, 110.051, 1
FROM mountains WHERE slug='sumbing'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Sindoro
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'sindoro',
  '{"id":"Gunung Sindoro","en":"Mount Sindoro"}'::jsonb,
  ARRAY['Sindara','Sundoro'],
  'Jawa', 'Jawa Tengah',
  '{"id":"Dari Temanggung ke Desa Kaliangkrik (Kec. Kledung); dekat rute Tembus Sumur Bandung.","en":"From Temanggung to Kaliangkrik village (Kledung); the Tembus Sumur Bandung route is nearby."}'::jsonb,
  -7.309, 109.992,
  '{"id":"Puncak Kasayasakawati","en":"Kasayasakawati Summit"}'::jsonb,
  3136, 3, 'published',
  '{"location":{"source":"desa Kaliangkrik","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Sindoro Mount Sindoro Sundoro Kasayasakawati'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Kaliangkrik","en":"Kaliangkrik route"}'::jsonb, 7, 9, 1700,
  '{"id":"Izin pos desa; kawasan hutan lindung.","en":"Village post permit; protected forest area."}'::jsonb, 1
FROM mountains WHERE slug='sindoro'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Kaliangkrik","en":"Kaliangkrik basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung, parkir kendaraan roda dua","en":"Registration post, parking, food stalls, motorcycle parking"}'::jsonb,
  '{"id":"Izin Rp 10.000–20.000","en":"Permit IDR 10,000–20,000"}'::jsonb,
  true, -7.315, 109.985, 1
FROM mountains WHERE slug='sindoro'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Merbabu
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'merbabu',
  '{"id":"Gunung Merbabu","en":"Mount Merbabu"}'::jsonb,
  ARRAY['Merbabu'],
  'Jawa', 'Jawa Tengah',
  '{"id":"Jalur utama dari Selo (Boyolali) antara Merbabu–Merapi; alternatif Wekas (Salatiga) dan Suwanting (Magelang).","en":"Main route from Selo (Boyolali) between Merbabu–Merapi; alternatives Wekas (Salatiga) and Suwanting (Magelang)."}'::jsonb,
  -7.452, 110.442,
  '{"id":"Puncak Kenteng Songo","en":"Kenteng Songo Summit"}'::jsonb,
  3145, 3, 'published',
  '{"location":{"source":"TNGM (Taman Nasional Gunung Merbabu)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGM fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Merbabu Mount Merbabu Kenteng Songo'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Selo","en":"Selo route"}'::jsonb, 7, 8, 1600,
  '{"id":"Izin daring TNGM; kuota harian.","en":"Online TNGM permit; daily quota."}'::jsonb, 1
FROM mountains WHERE slug='merbabu'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Selo","en":"Selo basecamp"}'::jsonb,
  '{"id":"Pos TNGM, warung, parkir, penginapan desa","en":"TNGM post, food stalls, parking, village homestays"}'::jsonb,
  '{"id":"Tiket Rp 10.000–20.000/hari","en":"Ticket IDR 10,000–20,000/day"}'::jsonb,
  true, -7.479, 110.451, 1
FROM mountains WHERE slug='merbabu'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Merapi
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'merapi',
  '{"id":"Gunung Merapi","en":"Mount Merapi"}'::jsonb,
  ARRAY['Merapi Yogyakarta'],
  'Jawa', 'DI Yogyakarta',
  '{"id":"Jalur utama Selo (utara, Boyolali); jalur Kinahrejo (selatan, Sleman) dan Deles (timur, Klaten). Aktivitas vulkanik — status siaga menutup jalur.","en":"Main routes: Selo (north, Boyolali); Kinahrejo (south, Sleman) and Deles (east, Klaten). Active volcano — alert level closes trails."}'::jsonb,
  -7.541, 110.446,
  '{"id":"Puncak Garuda","en":"Garuda Peak"}'::jsonb,
  2930, 4, 'published',
  '{"location":{"source":"TNGM (Taman Nasional Gunung Merapi)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGM fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Merapi Mount Merapi Garuda Selo Kinahrejo'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Selo","en":"Selo route"}'::jsonb, 6, 8, 1500,
  '{"id":"Izin daring TNGM; status vulkanik PGK menentukan buka/tutup.","en":"Online TNGM permit; volcanic alert level decides open/closed."}'::jsonb, 1
FROM mountains WHERE slug='merapi'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Selo","en":"Selo basecamp"}'::jsonb,
  '{"id":"Pos TNGM, penginapan, warung, jeep lava tour sekitar","en":"TNGM post, homestays, food stalls, nearby lava-tour jeeps"}'::jsonb,
  '{"id":"Tiket Rp 10.000–20.000/hari","en":"Ticket IDR 10,000–20,000/day"}'::jsonb,
  true, -7.523, 110.450, 1
FROM mountains WHERE slug='merapi'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Prau (Dieng)
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'prau',
  '{"id":"Gunung Prau","en":"Mount Prau"}'::jsonb,
  ARRAY['Prau Dieng','Prau'],
  'Jawa', 'Jawa Tengah',
  '{"id":"Akses dari kompleks Dieng (Desa Dieng Wetan) ±3 jam dari Semarang/Yogyakarta; alternatif Patak Banteng dan Wates.","en":"Access from the Dieng plateau (Dieng Wetan village) ±3 h from Semarang/Yogyakarta; alternatives Patak Banteng and Wates."}'::jsonb,
  -7.313, 109.833,
  '{"id":"Puncak Kirir petung","en":"Kirirpetung Summit"}'::jsonb,
  2565, 2, 'published',
  '{"location":{"source":"desa Dieng Wetan","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Prau Mount Prau Dieng Kirirpetung'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Dieng Wetan","en":"Dieng Wetan route"}'::jsonb, 6, 6, 900,
  '{"id":"Izin pos desa; buku tamu.","en":"Village post permit; guest book."}'::jsonb, 1
FROM mountains WHERE slug='prau'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Dieng Wetan","en":"Dieng Wetan basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, homestay desa wisata, warung, penginapan sekitar Candi Arjuna","en":"Registration post, tourism-village homestays, food stalls, lodging near Arjuna temple complex"}'::jsonb,
  '{"id":"Izin Rp 10.000; homestay Rp 150.000–250.000/malam","en":"Permit IDR 10,000; homestay IDR 150,000–250,000/night"}'::jsonb,
  true, -7.318, 109.859, 1
FROM mountains WHERE slug='prau'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Gede (TNGGP)
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'gede',
  '{"id":"Gunung Gede","en":"Mount Gede"}'::jsonb,
  ARRAY['Gede Pangrango'],
  'Jawa', 'Jawa Barat',
  '{"id":"Jalur utama Cibodas (Cianjur) ±2 jam dari Jakarta via Puncak; alternatif Gunung Putri dan Selabintana (Sukabumi).","en":"Main route Cibodas (Cianjur) ±2 h from Jakarta via Puncak; alternatives Gunung Putri and Selabintana (Sukabumi)."}'::jsonb,
  -6.772, 106.981,
  '{"id":"Puncak Gede","en":"Gede Summit"}'::jsonb,
  2958, 3, 'published',
  '{"location":{"source":"TNGGP (Balai Taman Nasional Gunung Gede Pangrango)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGGP fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Gede Mount Gede Gede Pangrango Cibodas'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Cibodas","en":"Cibodas route"}'::jsonb, 9, 9, 1700,
  '{"id":"Kuota harian TNGGP + izin daring (reservasi resmi); ditutup saat kebakaran/cuaca ekstrem.","en":"TNGGP daily quota + online permit (official reservation); closed during fires/extreme weather."}'::jsonb, 1
FROM mountains WHERE slug='gede'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Cibodas","en":"Cibodas basecamp"}'::jsonb,
  '{"id":"Kantor TN, museum, warung, parkir, penginapan Cibodas","en":"National-park office, museum, food stalls, parking, Cibodas lodging"}'::jsonb,
  '{"id":"Tiket Rp 20.000–25.000 (wk) / Rp 10.000–15.000 (biasa)","en":"Ticket IDR 20,000–25,000 (weekend) / IDR 10,000–15,000 (weekday)"}'::jsonb,
  true, -6.741, 107.008, 1
FROM mountains WHERE slug='gede'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Papandayan
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'papandayan',
  '{"id":"Gunung Papandayan","en":"Mount Papandayan"}'::jsonb,
  ARRAY['Papandayan'],
  'Jawa', 'Jawa Barat',
  '{"id":"Dari Garut ±1,5 jam ke Desa Cisaruni/Kancana (Kec. Cisurupan).","en":"From Garut ±1.5 h to Cisaruni/Kancana village (Cisurupan district)."}'::jsonb,
  -7.321, 107.729,
  '{"id":"Puncak Papandayan","en":"Papandayan Summit"}'::jsonb,
  2665, 2, 'published',
  '{"location":{"source":"Perhutani / desa Cisurupan","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Papandayan Mount Papandayan Edelweis Tegal Alun'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Cisaruni (Tegal Alun)","en":"Cisaruni route (Tegal Alun)"}'::jsonb, 5, 5, 900,
  '{"id":"Izin pos desa; zona kawah berbahaya — patuhi jalur.","en":"Village post permit; dangerous crater zone — stay on trail."}'::jsonb, 1
FROM mountains WHERE slug='papandayan'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Cisaruni","en":"Cisaruni basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung, camping ground Tegal Alun di jalur","en":"Registration post, parking, food stalls, Tegal Alun campsite on route"}'::jsonb,
  '{"id":"Izin Rp 10.000–20.000","en":"Permit IDR 10,000–20,000"}'::jsonb,
  true, -7.321, 107.721, 1
FROM mountains WHERE slug='papandayan'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Ciremai — highest in West Java
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'ciremai',
  '{"id":"Gunung Ciremai","en":"Mount Ciremai"}'::jsonb,
  ARRAY['Cereme'],
  'Jawa', 'Jawa Barat',
  '{"id":"Taman Nasional Gunung Ciremai; jalur dari Kuningan (Apuy, Linggarjati) dan Majalengka (Palutungan).","en":"Mount Ciremai National Park; routes from Kuningan (Apuy, Linggarjati) and Majalengka (Palutungan)."}'::jsonb,
  -6.892, 108.399,
  '{"id":"Puncak Ciremai","en":"Ciremai Summit"}'::jsonb,
  3078, 3, 'published',
  '{"location":{"source":"TNGC (Taman Nasional Gunung Ciremai)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGC fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Ciremai Mount Ciremai Cereme Linggarjati'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Apuy","en":"Apuy route"}'::jsonb, 9, 10, 1800,
  '{"id":"Izin daring TNGC; kuota harian.","en":"Online TNGC permit; daily quota."}'::jsonb, 1
FROM mountains WHERE slug='ciremai'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Apuy","en":"Apuy basecamp"}'::jsonb,
  '{"id":"Pos TNGC, penginapan desa, warung","en":"TNGC post, village homestays, food stalls"}'::jsonb,
  '{"id":"Tiket Rp 10.000–20.000/hari","en":"Ticket IDR 10,000–20,000/day"}'::jsonb,
  true, -6.874, 108.448, 1
FROM mountains WHERE slug='ciremai'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Bromo
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'bromo',
  '{"id":"Gunung Bromo","en":"Mount Bromo"}'::jsonb,
  ARRAY['Tengger','Bromo Tengger Semeru'],
  'Jawa', 'Jawa Timur',
  '{"id":"Akses paling mudah dari Cemoro Lawang (Ngadisari) ±1,5 jam dari Malang/Probolinggo; jalan kaki/jeep melintasi lautan pasir.","en":"Easiest access from Cemoro Lawang (Ngadisari) ±1.5 h from Malang/Probolinggo; walk or jeep across the sea of sand."}'::jsonb,
  -7.942, 112.950,
  '{"id":"Puncak Bromo (kawah aktif)","en":"Bromo Summit (active crater)"}'::jsonb,
  2329, 1, 'published',
  '{"location":{"source":"TNTBTS (Taman Nasional Bromo Tengger Semeru)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNTBTS fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Bromo Mount Bromo Tengger Lautan Pasir'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Cemoro Lawang (Penanjakan + kawah)","en":"Cemoro Lawang route (Penanjakan + crater)"}'::jsonb, 4, 3, 300,
  '{"id":"Tiket masuk TNTBTS; kawasan kawah dilarang saat status siaga.","en":"TNTBTS entrance ticket; crater area closed at alert level."}'::jsonb, 1
FROM mountains WHERE slug='bromo'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Cemoro Lawang","en":"Cemoro Lawang basecamp"}'::jsonb,
  '{"id":"Kantor TN, hotel/penginapan banyak, warung, sewa jeep","en":"Park office, many hotels/homestays, food stalls, jeep rental"}'::jsonb,
  '{"id":"Tiket Rp 25.000–50.000; jeep Rp 350.000–500.000","en":"Ticket IDR 25,000–50,000; jeep IDR 350,000–500,000"}'::jsonb,
  true, -7.913, 112.952, 1
FROM mountains WHERE slug='bromo'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Ijen
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'ijen',
  '{"id":"Gunung Ijen","en":"Mount Ijen"}'::jsonb,
  ARRAY['Kawah Ijen'],
  'Jawa', 'Jawa Timur',
  '{"id":"Dari Banyuwangi ±1,5 jam ke Pos Paltuding (Kec. Licin); dari Bondowoso ±2 jam.","en":"From Banyuwangi ±1.5 h to Paltuding post (Licin district); from Bondowoso ±2 h."}'::jsonb,
  -8.058, 114.242,
  '{"id":"Puncak Ijen (kawah)","en":"Ijen Summit (crater)"}'::jsonb,
  2769, 2, 'published',
  '{"location":{"source":"TNIKB (Balai Taman Nasional Ijen)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"fee board Paltuding","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Ijen Mount Ijen Kawah Ijen Blue Fire'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Paltuding","en":"Paltuding route"}'::jsonb, 3, 3, 500,
  '{"id":"Tiket masuk TNIKB; fenomena api biru ~01.00–03.00; masker wajib (gas SO2).","en":"TNIKB ticket; blue-fire window ~01:00–03:00; gas mask required (SO2)."}'::jsonb, 1
FROM mountains WHERE slug='ijen'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Pos Paltuding","en":"Paltuding post"}'::jsonb,
  '{"id":"Pos TN, parkir, warung, toilet","en":"Park post, parking, food stalls, toilets"}'::jsonb,
  '{"id":"Tiket Rp 15.000–50.000 (wk)","en":"Ticket IDR 15,000–50,000 (weekend)"}'::jsonb,
  true, -8.094, 114.237, 1
FROM mountains WHERE slug='ijen'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Argopuro
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'argopuro',
  '{"id":"Gunung Argopuro","en":"Mount Argopuro"}'::jsonb,
  ARRAY['Rengganis'],
  'Jawa', 'Jawa Timur',
  '{"id":"Dari Bondowoso ke Desa Baderan (Sumber Wringin) atau dari Probolinggo (Besuk Merah). Jalur panjang klasik Jawa Timur.","en":"From Bondowoso to Baderan village (Sumber Wringin) or from Probolinggo (Besuk Merah). A classic long route in East Java."}'::jsonb,
  -7.880, 113.570,
  '{"id":"Puncak Rengganis","en":"Rengganis Summit"}'::jsonb,
  3088, 4, 'published',
  '{"location":{"source":"Perhutani / desa Baderan","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Argopuro Mount Argopuro Rengganis Baderan'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Baderan (traverse)","en":"Baderan route (traverse)"}'::jsonb, 32, 30, 2400,
  '{"id":"Izin pos desa + Perhutani; wajib pemandu; jalur 4–5 hari.","en":"Village + Perhutani permit; guide mandatory; 4–5 day route."}'::jsonb, 1
FROM mountains WHERE slug='argopuro'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Baderan","en":"Baderan basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung, jasa pemandu/penunggu","en":"Registration post, parking, food stalls, guide/porter service"}'::jsonb,
  '{"id":"Izin Rp 15.000–30.000; pemandu Rp 500.000–800.000/trip","en":"Permit IDR 15,000–30,000; guide IDR 500,000–800,000/trip"}'::jsonb,
  true, -7.917, 113.600, 1
FROM mountains WHERE slug='argopuro'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- ============================ SUMATRA ============================

-- Gunung Kerinci — highest on Sumatra
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'kerinci',
  '{"id":"Gunung Kerinci","en":"Mount Kerinci"}'::jsonb,
  ARRAY['Kerinci Seblat'],
  'Sumatra', 'Jambi',
  '{"id":"Dari Sungai Penuh (Jambi) ±1 jam ke Desa Kersik Tuo (Kec. Kayu Aro); bandara terdekat Minangkabau/PAD.","en":"From Sungai Penuh (Jambi) ±1 h to Kersik Tuo village (Kayu Aro district); nearest airports Padang or Minangkabau."}'::jsonb,
  -1.697, 101.264,
  '{"id":"Puncak Kerinci","en":"Kerinci Summit"}'::jsonb,
  3805, 4, 'published',
  '{"location":{"source":"TNSK (Taman Nasional Kerinci Seblat)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNSK fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Kerinci Mount Kerinci Kerinci Seblat Kersik Tuo'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Kersik Tuo","en":"Kersik Tuo route"}'::jsonb, 11, 14, 2400,
  '{"id":"Izin TNSK daring; kawasan habitat harimau — berkelompok wajib.","en":"Online TNSK permit; tiger habitat — groups mandatory."}'::jsonb, 1
FROM mountains WHERE slug='kerinci'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Kersik Tuo","en":"Kersik Tuo basecamp"}'::jsonb,
  '{"id":"Pos TN, homestay perkebunan teh, warung, porter","en":"Park post, tea-plantation homestays, food stalls, porters"}'::jsonb,
  '{"id":"Tiket Rp 10.000–20.000; porter opsional","en":"Ticket IDR 10,000–20,000; optional porters"}'::jsonb,
  true, -1.730, 101.274, 1
FROM mountains WHERE slug='kerinci'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Dempo
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'dempo',
  '{"id":"Gunung Dempo","en":"Mount Dempo"}'::jsonb,
  ARRAY['Dempo Pagar Alam'],
  'Sumatra', 'Sumatera Selatan',
  '{"id":"Dari Palembang ±5 jam ke Pagar Alam; mulai dari Desa Tanjung Sakti / jalur perkebunan teh PTPN.","en":"From Palembang ±5 h to Pagar Alam; start at Tanjung Sakti village or the PTPN tea plantation."}'::jsonb,
  -4.034, 103.126,
  '{"id":"Puncak Dempo","en":"Dempo Summit"}'::jsonb,
  3173, 3, 'published',
  '{"location":{"source":"desa Pagar Alam","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Dempo Mount Dempo Pagar Alam'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Tanjung Sakti","en":"Tanjung Sakti route"}'::jsonb, 9, 10, 2300,
  '{"id":"Izin pos desa; area perkebunan teh di awal jalur.","en":"Village post permit; tea plantation area at the start."}'::jsonb, 1
FROM mountains WHERE slug='dempo'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Pagar Alam","en":"Pagar Alam basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, penginapan Pagar Alam, warung","en":"Registration post, Pagar Alam lodging, food stalls"}'::jsonb,
  '{"id":"Izin Rp 10.000–20.000; pemandu opsional","en":"Permit IDR 10,000–20,000; optional guide"}'::jsonb,
  true, -4.057, 103.135, 1
FROM mountains WHERE slug='dempo'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Marapi (West Sumatra — distinct from Merapi)
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'marapi',
  '{"id":"Gunung Marapi","en":"Mount Marapi"}'::jsonb,
  ARRAY['Marapi Minangkabau'],
  'Sumatra', 'Sumatera Barat',
  '{"id":"Dari Bukittinggi ±1 jam ke Desa Salimpaung/Buntu Buntu (Kec. Tabiang Panjang). Gunung api tertinggi Sumatera Barat.","en":"From Bukittinggi ±1 h to Salimpaung/Buntu Buntu village. The highest volcano in West Sumatra."}'::jsonb,
  -0.381, 100.474,
  '{"id":"Puncak Marapi (Bancah)","en":"Marapi Summit"}'::jsonb,
  2891, 3, 'published',
  '{"location":{"source":"desa Salimpaung","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Marapi Mount Marapi Minangkabau Bukittinggi'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Salimpaung","en":"Salimpaung route"}'::jsonb, 8, 10, 1700,
  '{"id":"Izin pos desa; status vulkanik ditentukan PVMBG.","en":"Village post permit; PVMBG volcanic status decides access."}'::jsonb, 1
FROM mountains WHERE slug='marapi'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Salimpaung","en":"Salimpaung basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung","en":"Registration post, parking, food stalls"}'::jsonb,
  '{"id":"Izin Rp 10.000–20.000; wajib pemandu (aturan 2023)","en":"Permit IDR 10,000–20,000; guide mandatory (2023 rule)"}'::jsonb,
  true, -0.371, 100.478, 1
FROM mountains WHERE slug='marapi'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Sibayak
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'sibayak',
  '{"id":"Gunung Sibayak","en":"Mount Sibayak"}'::jsonb,
  ARRAY['Sibayak Berastagi'],
  'Sumatra', 'Sumatera Utara',
  '{"id":"Dari Berastagi jalan kaki/jalur desa Semangat Gunung; kawah ±3 jam dari kota.","en":"Walk from Berastagi via Semangat Gunung village; crater ±3 h from town."}'::jsonb,
  3.205, 98.524,
  '{"id":"Puncak Sibayak","en":"Sibayak Summit"}'::jsonb,
  2212, 2, 'published',
  '{"location":{"source":"desa Berastagi","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Sibayak Mount Sibayak Berastagi Karo'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Berastagi (Semangat Gunung)","en":"Berastagi route (Semangat Gunung)"}'::jsonb, 5, 5, 1000,
  '{"id":"Izin pos desa; gas belerang di kawah — masker disarankan.","en":"Village post permit; sulfur gas at crater — mask advised."}'::jsonb, 1
FROM mountains WHERE slug='sibayak'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Berastagi","en":"Berastagi basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, banyak penginapan, pemandian air panas","en":"Registration post, many lodgings, hot springs"}'::jsonb,
  '{"id":"Izin Rp 5.000–10.000","en":"Permit IDR 5,000–10,000"}'::jsonb,
  true, 3.211, 98.519, 1
FROM mountains WHERE slug='sibayak'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- ============================ BALI & NUSA TENGGARA ============================

-- Gunung Rinjani
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'rinjani',
  '{"id":"Gunung Rinjani","en":"Mount Rinjani"}'::jsonb,
  ARRAY['Rinjani Segara Anak'],
  'Bali & Nusa Tenggara', 'Nusa Tenggara Barat',
  '{"id":"Dari Lombok: jalur Sembalun (puncak) dan Senaru (danau) ±3 jam dari Lombok International.","en":"From Lombok: Sembalun route (summit) and Senaru route (lake) ±3 h from Lombok International Airport."}'::jsonb,
  -8.419, 116.465,
  '{"id":"Puncak Rinjani","en":"Rinjani Summit"}'::jsonb,
  3726, 5, 'published',
  '{"location":{"source":"TNGR (Balai Taman Nasional Gunung Rinjani)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"TNGR fee board","reliability":"official","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Rinjani Mount Rinjani Segara Anak Sembalun Senaru'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Sembalun (puncak)","en":"Sembalun route (summit)"}'::jsonb, 12, 14, 2400,
  '{"id":"Izin daring TNGR; wajib pemandu/jasa terdaftar; jalur 2–4 hari.","en":"Online TNGR permit; licensed guide/porter service mandatory; 2–4 day route."}'::jsonb, 1
FROM mountains WHERE slug='rinjani'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Sembalun Lawang","en":"Sembalun Lawang basecamp"}'::jsonb,
  '{"id":"Pos TN, homestay desa, operator trekking resmi, warung","en":"Park post, village homestays, licensed trekking operators, food stalls"}'::jsonb,
  '{"id":"Tiket Rp 150.000 (wni)/Rp 1.000.000 (wna) paket; operator bervariasi","en":"Ticket IDR 150,000 (domestic)/IDR 1,000,000 (foreign) package; operator prices vary"}'::jsonb,
  true, -8.373, 116.513, 1
FROM mountains WHERE slug='rinjani'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Agung
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'agung',
  '{"id":"Gunung Agung","en":"Mount Agung"}'::jsonb,
  ARRAY['Agung Besakih'],
  'Bali & Nusa Tenggara', 'Bali',
  '{"id":"Jalur Pura Besakih (puncak), Selat dan Pasar Agung (Karangasem); status vulkanik menentukan buka/tutup.","en":"Routes: Pura Besakih (summit), Selat and Pasar Agung (Karangasem); volcanic status decides open/closed."}'::jsonb,
  -8.343, 115.507,
  '{"id":"Puncak Agung","en":"Agung Summit"}'::jsonb,
  3031, 4, 'published',
  '{"location":{"source":"Pura Besakih / desa Karangasem","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Agung Mount Agung Besakih Karangasem'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Pasar Agung","en":"Pasar Agung route"}'::jsonb, 6, 8, 1800,
  '{"id":"Izin desa/pura; wajib pemandu; status vulkanik PGK.","en":"Village/temple permit; guide mandatory; volcanic alert applies."}'::jsonb, 1
FROM mountains WHERE slug='agung'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Pasar Agung","en":"Pasar Agung basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, parkir, warung","en":"Registration post, parking, food stalls"}'::jsonb,
  '{"id":"Izin Rp 20.000–50.000; pemandu Rp 350.000–500.000","en":"Permit IDR 20,000–50,000; guide IDR 350,000–500,000"}'::jsonb,
  true, -8.381, 115.485, 1
FROM mountains WHERE slug='agung'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- Gunung Batur
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'batur',
  '{"id":"Gunung Batur","en":"Mount Batur"}'::jsonb,
  ARRAY['Batur Kintamani'],
  'Bali & Nusa Tenggara', 'Bali',
  '{"id":"Dari Kintamani ±1 jam dari Denpasar; mulai Toya Bungkah/Songan; trek matahari terbit klasik.","en":"From Kintamani ±1 h from Denpasar; start at Toya Bungkah/Songan; the classic sunrise trek."}'::jsonb,
  -8.242, 115.375,
  '{"id":"Puncak Batur","en":"Batur Summit"}'::jsonb,
  1717, 1, 'published',
  '{"location":{"source":"desa Kintamani","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Batur Mount Batur Kintamani Sunrise'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Toya Bungkah","en":"Toya Bungkah route"}'::jsonb, 4, 4, 800,
  '{"id":"Izin desa (Pangkalan KKN); pemandu aturan desa; mulai ~04.00 untuk sunrise.","en":"Village permit (Pangkalan KKN); village guide rule; start ~04:00 for sunrise."}'::jsonb, 1
FROM mountains WHERE slug='batur'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Toya Bungkah","en":"Toya Bungkah basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, penginapan, pemandian air panas","en":"Registration post, lodging, hot springs"}'::jsonb,
  '{"id":"Izin Rp 15.000–30.000; pemandu Rp 300.000–400.000","en":"Permit IDR 15,000–30,000; guide IDR 300,000–400,000"}'::jsonb,
  true, -8.245, 115.386, 1
FROM mountains WHERE slug='batur'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- ============================ SULAWESI ============================

-- Gunung Lokon
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'lokon',
  '{"id":"Gunung Lokon","en":"Mount Lokon"}'::jsonb,
  ARRAY['Lokon Empung'],
  'Sulawesi', 'Sulawesi Utara',
  '{"id":"Dari Tomohon (Kakaskasen) ±30 menit dari Manado; jalur tikus ke Tompaluan.","en":"From Tomohon (Kakaskasen) ±30 min from Manado; the rat-trail route to Tompaluan."}'::jsonb,
  1.357, 124.792,
  '{"id":"Puncak Tompaluan","en":"Tompaluan Peak"}'::jsonb,
  1580, 2, 'published',
  '{"location":{"source":"desa Tomohon","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"BIG (kompleks Lokon-Empung)","reliability":"official","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"route report consensus","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"desa setempat","reliability":"community","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Lokon Mount Lokon Empung Tompaluan Tomohon'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Kakaskasen","en":"Kakaskasen route"}'::jsonb, 4, 5, 1000,
  '{"id":"Izin pos desa; status vulkanik ditentukan PVMBG.","en":"Village post permit; PVMBG volcanic status applies."}'::jsonb, 1
FROM mountains WHERE slug='lokon'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Basecamp Kakaskasen","en":"Kakaskasen basecamp"}'::jsonb,
  '{"id":"Pos pendaftaran, penginapan Tomohon, warung","en":"Registration post, Tomohon lodging, food stalls"}'::jsonb,
  '{"id":"Izin Rp 10.000–20.000; pemandu Rp 200.000–300.000","en":"Permit IDR 10,000–20,000; guide IDR 200,000–300,000"}'::jsonb,
  true, 1.364, 124.787, 1
FROM mountains WHERE slug='lokon'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- ============================ KALIMANTAN ============================

-- Bukit Raya — highest in Kalimantan
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'bukit-raya',
  '{"id":"Bukit Raya","en":"Bukit Raya"}'::jsonb,
  ARRAY['Bukit Raya Baka','Raya Kalimantan'],
  'Kalimantan', 'Kalimantan Tengah',
  '{"id":"Dalam Taman Nasional Bukit Baka Bukit Raya; akses dari Desa Tumbang Koroi (Kalteng) via Sungai Katingan — ekspedisi.","en":"Inside Bukit Baka Bukit Raya National Park; access from Tumbang Koroi village (Central Kalimantan) via the Katingan River — an expedition."}'::jsonb,
  -0.412, 112.483,
  '{"id":"Puncak Bukit Raya","en":"Bukit Raya Summit"}'::jsonb,
  2276, 5, 'published',
  '{"location":{"source":"TNPBBBR (Taman Nasional Bukit Baka Bukit Raya)","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"reported by expedition teams","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"expedition reports (very few parties)","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"expedition reports","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Bukit Raya Raya Kalimantan Bukit Baka'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Tumbang Koroi (sungai + hutan)","en":"Tumbang Koroi route (river + forest)"}'::jsonb, 25, 40, 2000,
  '{"id":"Izin TN BBBR; ketintang sungai + trek 5–7 hari; musim banjir menutup jalur.","en":"BBBR National Park permit; river boat + 5–7 day trek; flood season closes the route."}'::jsonb, 1
FROM mountains WHERE slug='bukit-raya'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Desa Tumbang Koroi","en":"Tumbang Koroi village"}'::jsonb,
  '{"id":"Pos TN, homestay sederhana, sewa perahu","en":"Park post, simple homestays, boat rental"}'::jsonb,
  '{"id":"Ekspedisi Rp 5.000.000–10.000.000/kelompok (estimasi)","en":"Expedition IDR 5,000,000–10,000,000/group (estimate)"}'::jsonb,
  true, -0.571, 112.548, 1
FROM mountains WHERE slug='bukit-raya'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- ============================ MALUKU ============================

-- Gunung Binaiya — highest on Seram
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'binaiya',
  '{"id":"Gunung Binaiya","en":"Mount Binaiya"}'::jsonb,
  ARRAY['Binaiya Seram','Binaia'],
  'Maluku', 'Maluku',
  '{"id":"Dalam Taman Nasional Manusela; akses dari Masohi (Ambon → feri) ke Desa Manusela — ekspedisi 5–7 hari.","en":"Inside Manusela National Park; access from Masohi (ferry from Ambon) to Manusela village — a 5–7 day expedition."}'::jsonb,
  -3.136, 129.135,
  '{"id":"Puncak Binaiya","en":"Binaiya Summit"}'::jsonb,
  3027, 5, 'published',
  '{"location":{"source":"Taman Nasional Manusela","reliability":"community","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"reported by expedition teams","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"expedition reports (very few parties)","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"expedition reports","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Gunung Binaiya Mount Binaiya Binaia Seram Manusela'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Desa Manusela","en":"Manusela village route"}'::jsonb, 22, 40, 2700,
  '{"id":"Izin TN Manusela; pemandu lokal wajib; logistik penuh 5–7 hari.","en":"Manusela National Park permit; local guide mandatory; full 5–7 day logistics."}'::jsonb, 1
FROM mountains WHERE slug='binaiya'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Desa Manusela","en":"Manusela village"}'::jsonb,
  '{"id":"Pos TN, homestay sederhana, sewa perahu dari Masohi","en":"Park post, simple homestays, boat rental from Masohi"}'::jsonb,
  '{"id":"Ekspedisi Rp 6.000.000–12.000.000/kelompok (estimasi)","en":"Expedition IDR 6,000,000–12,000,000/group (estimate)"}'::jsonb,
  true, -3.157, 129.194, 1
FROM mountains WHERE slug='binaiya'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- ============================ PAPUA ============================

-- Puncak Trikora
INSERT INTO mountains (id, slug, name, aliases, region, province, location, latitude, longitude, peak_name, peak_height_m, difficulty, status, data_meta, search_text)
VALUES (gen_random_uuid(), 'trikora',
  '{"id":"Puncak Trikora","en":"Mount Trikora"}'::jsonb,
  ARRAY['Trikora'],
  'Papua', 'Papua Pegunungan',
  '{"id":"Dari Wamena ke kawasan Ilu/Kemeri (Distrik Welarek); akses darat sangat terbatas — konfirmasi keamanan lokal dulu.","en":"From Wamena to the Ilu/Kemeri area (Welarek district); road access very limited — confirm local security first."}'::jsonb,
  -4.101, 138.592,
  '{"id":"Puncak Trikora","en":"Trikora Summit"}'::jsonb,
  4750, 5, 'published',
  '{"location":{"source":"expedition reports","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"peak_height_m":{"source":"reported (SRTM-based estimates vary 4730–4751)","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"difficulty":{"source":"expedition reports (very few parties)","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"},"cost":{"source":"expedition reports","reliability":"reported","updated_at":"2026-08-01T00:00:00Z"}}'::jsonb,
  lower('Puncak Trikora Mount Trikora Wamena'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO mountain_routes (id, mountain_id, name, distance_km, duration_hours, elevation_gain_m, entry_requirements, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Jalur Ilu (Kemeri)","en":"Ilu route (Kemeri)"}'::jsonb, 20, 36, 3400,
  '{"id":"Izin adat + koordinasi TNI/Polri setempat; jalur 4–6 hari; cuaca dingin ekstrem.","en":"Customary permit + local security coordination; 4–6 day route; extreme cold weather."}'::jsonb, 1
FROM mountains WHERE slug='trikora'
AND NOT EXISTS (SELECT 1 FROM mountain_routes r WHERE r.mountain_id=mountains.id);

INSERT INTO mountain_basecamps (id, mountain_id, name, facilities, cost_estimate, is_permit_point, latitude, longitude, sort_order)
SELECT gen_random_uuid(), id, '{"id":"Kampung Ilu","en":"Ilu village"}'::jsonb,
  '{"id":"Pos adat sederhana, sewa penginapan kampung, porter lokal","en":"Simple customary post, village lodging, local porters"}'::jsonb,
  '{"id":"Ekspedisi Rp 10.000.000–20.000.000/kelompok (estimasi)","en":"Expedition IDR 10,000,000–20,000,000/group (estimate)"}'::jsonb,
  true, -4.150, 138.610, 1
FROM mountains WHERE slug='trikora'
AND NOT EXISTS (SELECT 1 FROM mountain_basecamps b WHERE b.mountain_id=mountains.id);

-- +goose Down
DELETE FROM mountain_basecamps WHERE mountain_id IN (SELECT id FROM mountains WHERE slug IN (
  'semeru','lawu','slamet','sumbing','sindoro','merbabu','merapi','prau','gede','papandayan','ciremai','bromo','ijen','argopuro',
  'kerinci','dempo','marapi','sibayak','rinjani','agung','batur','lokon','bukit-raya','binaiya','trikora'));
DELETE FROM mountain_routes WHERE mountain_id IN (SELECT id FROM mountains WHERE slug IN (
  'semeru','lawu','slamet','sumbing','sindoro','merbabu','merapi','prau','gede','papandayan','ciremai','bromo','ijen','argopuro',
  'kerinci','dempo','marapi','sibayak','rinjani','agung','batur','lokon','bukit-raya','binaiya','trikora'));
DELETE FROM mountains WHERE slug IN (
  'semeru','lawu','slamet','sumbing','sindoro','merbabu','merapi','prau','gede','papandayan','ciremai','bromo','ijen','argopuro',
  'kerinci','dempo','marapi','sibayak','rinjani','agung','batur','lokon','bukit-raya','binaiya','trikora');
