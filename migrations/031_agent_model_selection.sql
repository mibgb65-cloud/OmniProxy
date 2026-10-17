-- NULL 表示动态全部模型；显式选择随受管备份生命周期保存，避免刷新重新加入未选模型。
ALTER TABLE agent_backups ADD COLUMN model_ids TEXT;
