-- 0013_add_requires_password_change.up.sql
-- Adds the requires_password_change column to the user table to enforce mandatory password rotation on first login.
ALTER TABLE `user`
  ADD COLUMN requires_password_change TINYINT(1) NOT NULL DEFAULT 0 AFTER is_active;
