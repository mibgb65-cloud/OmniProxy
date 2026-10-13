-- 应用级网关开关；默认开启，保留已有 Codex WebSocket 接入行为。
CREATE TABLE codex_settings (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    websocket_enabled INTEGER NOT NULL CHECK(websocket_enabled IN (0, 1))
);
INSERT INTO codex_settings(id, websocket_enabled) VALUES(1, 1);
