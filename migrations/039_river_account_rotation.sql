-- 升级保持现有队列顺序；均衡方式需要用户主动选择。
CREATE TABLE river_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    account_rotation_mode TEXT NOT NULL DEFAULT 'queue' CHECK(account_rotation_mode IN ('queue', 'balanced'))
);
INSERT INTO river_settings(id) VALUES(1);
