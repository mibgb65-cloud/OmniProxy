-- 旧服务商偏好归入逻辑模型；已有逻辑模型配置优先，避免覆盖用户显式设置。
INSERT INTO model_provider_preferences (model, provider_id, updated_at, provider_order)
SELECT CASE model
    WHEN 'deepseek-ai/DeepSeek-V4.1-Flash' THEN 'deepseek-flash'
    WHEN 'zai-org/GLM-5.3-Flash' THEN 'GLM-5.3-Flash'
    WHEN 'nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4' THEN 'NVIDIA-Nemotron-3.5'
END, provider_id, updated_at, provider_order
FROM model_provider_preferences
WHERE model IN ('deepseek-ai/DeepSeek-V4.1-Flash', 'zai-org/GLM-5.3-Flash', 'nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4')
ON CONFLICT(model) DO NOTHING;

DELETE FROM model_provider_preferences
WHERE model IN ('deepseek-ai/DeepSeek-V4.1-Flash', 'zai-org/GLM-5.3-Flash', 'nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4');
