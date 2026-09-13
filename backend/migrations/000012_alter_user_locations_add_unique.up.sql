-- UNIQUE (user_id, label) để hỗ trợ upsert atomic bằng ON CONFLICT,
-- tránh race condition khi 2 request đồng thời cùng user_id + label
-- (VD: user bấm nút 2 lần liên tiếp lúc mạng yếu).
ALTER TABLE user_locations
    ADD CONSTRAINT uq_user_locations_user_label UNIQUE (user_id, label);
