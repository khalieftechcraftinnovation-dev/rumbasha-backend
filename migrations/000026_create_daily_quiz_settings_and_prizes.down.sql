-- Riwayat coin hadiah (coin_transactions type reward, reference_type daily_quiz_prize) tetap
-- disimpan; hanya tabel pengaturan & penerima hadiah yang dihapus.
DROP TABLE IF EXISTS daily_quiz_prize_awards;
DROP TABLE IF EXISTS daily_quiz_prize_periods;
DROP TABLE IF EXISTS daily_quiz_settings;
