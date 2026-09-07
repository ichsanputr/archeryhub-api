-- Performance Composite Indexes Migration for Archeris Platform

-- 1. Notifications composite index
CREATE INDEX IF NOT EXISTS idx_notif_user_read_created ON notifications(user_id, is_read, created_at DESC);

-- 2. Qualification end scores composite index
CREATE INDEX IF NOT EXISTS idx_qes_part_session_end ON qualification_end_scores(participant_uuid, session_uuid, end_number);

-- 3. Elimination matches composite index
CREATE INDEX IF NOT EXISTS idx_elim_bracket_round_match ON elimination_matches(bracket_uuid, round_no, match_no);

-- 4. Payment transactions composite index
CREATE INDEX IF NOT EXISTS idx_pay_user_status_created ON payment_transactions(user_id, status, created_at DESC);

-- 5. Event participants composite index
CREATE INDEX IF NOT EXISTS idx_part_event_pay_cat ON event_participants(event_id, payment_status, category_id);

-- 6. Archer certificates composite index
CREATE INDEX IF NOT EXISTS idx_cert_event_reg ON archer_certificates(event_id, registration_id);

-- 7. Chat messages composite index
CREATE INDEX IF NOT EXISTS idx_chat_conv_created ON chat_messages(conversation_id, created_at ASC);

-- 8. Chat conversations composite indexes
CREATE INDEX IF NOT EXISTS idx_conv_archer_last ON chat_conversations(archer_id, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conv_seller_last ON chat_conversations(seller_id, last_message_at DESC);

