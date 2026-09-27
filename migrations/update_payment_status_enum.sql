-- migration: update payment_status enum to exactly 4 statuses ('cancelled', 'pending', 'expired', 'paid')
UPDATE tournament_participants SET payment_status = 'expired' WHERE payment_status = 'unpaid';
ALTER TABLE tournament_participants MODIFY COLUMN payment_status ENUM('cancelled', 'pending', 'expired', 'paid') NOT NULL DEFAULT 'pending';
