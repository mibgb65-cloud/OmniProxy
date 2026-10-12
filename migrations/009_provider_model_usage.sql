-- 独立累计，没有资源删除外键，也不依赖有保留上限的请求明细。
CREATE TABLE usage_provider_models (
 provider_id TEXT NOT NULL,
 provider_name TEXT NOT NULL,
 model TEXT NOT NULL,
 requests INTEGER NOT NULL DEFAULT 0,
 completed_usage INTEGER NOT NULL DEFAULT 0,
 input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens>=0),
 output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens>=0),
 PRIMARY KEY(provider_id,model)
);

-- 历史桶没有配置 ID；仅在请求快照或名称映射唯一时归属。
-- 同一个旧桶曾混合多种配置时保留历史分组，不能把全部用量算给其中一家。
WITH bucket_sources AS (
 SELECT b.*,
  CASE WHEN (SELECT COUNT(DISTINCT r.provider_id) FROM request_records r
    WHERE r.day=b.day AND r.subject=b.subject AND r.model=b.model AND r.provider_id!='')>1
   THEN 'legacy:' || b.provider_name
   ELSE COALESCE(
    (SELECT MIN(r.provider_id) FROM request_records r WHERE r.day=b.day AND r.subject=b.subject AND r.model=b.model AND r.provider_id!=''),
    (SELECT MIN(p.id) FROM providers p WHERE p.name=b.provider_name HAVING COUNT(*)=1),
    'legacy:' || b.provider_name)
  END AS matched_provider
 FROM usage_buckets b
 WHERE b.subject!='unassigned'
)
INSERT INTO usage_provider_models(provider_id,provider_name,model,requests,completed_usage,input_tokens,output_tokens)
SELECT matched_provider,MAX(provider_name),model,SUM(requests),SUM(completed_usage),SUM(input_tokens),SUM(output_tokens)
FROM bucket_sources GROUP BY matched_provider,model;
