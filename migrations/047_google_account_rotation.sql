-- Google 订阅账号的同池轮换方式；升级保持现有队列顺序，均衡需要用户主动选择。
CREATE TABLE google_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    account_rotation_mode TEXT NOT NULL DEFAULT 'queue' CHECK(account_rotation_mode IN ('queue', 'balanced'))
);
INSERT INTO google_settings(id) VALUES(1);
