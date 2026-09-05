import assert from 'node:assert/strict'
import test from 'node:test'

import { gatewayPlatformPresets, inferGatewayProviderForModel, routeDefinitions, routeStrategyChain } from './gatewayRoutePresets.js'
import { codexModelOptions, defaultCodexModels } from '../../constants/claudeModels.js'

test('GPT-6 Astra is selectable without changing existing defaults', () => {
  assert.ok(codexModelOptions.some((model) => model.id === 'gpt-6-astra'))
  assert.deepEqual(defaultCodexModels, ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna'])
  const openai = gatewayPlatformPresets.find((preset) => preset.key === 'openai')
  const astra = openai.models.find((model) => model.routeModels?.codex === 'gpt-6-astra')
  assert.equal(astra?.routeModels.openai, 'gpt-6-astra')
  for (const key of ['codex', 'openai']) {
    assert.ok(routeDefinitions.find((route) => route.key === key).modelPresets.includes('gpt-6-astra'))
  }
  assert.equal(inferGatewayProviderForModel('gpt-6-astra'), 'openai')
})

test('routeDefinitions build stable local gateway endpoints', () => {
  const endpoints = Object.fromEntries(routeDefinitions.map((route) => [route.key, route.endpoint(3899)]))

  assert.equal(endpoints.codex, 'http://127.0.0.1:3899/codex/v1')
  assert.equal(endpoints.claude, 'http://127.0.0.1:3899/anthropic-router')
  assert.equal(endpoints.openai, 'http://127.0.0.1:3899/opencode-router/v1')
  assert.equal(endpoints.gemini, 'http://127.0.0.1:3899/gemini')
})

test('routeDefinitions expose GPT-5.6 role-aware defaults and family presets', () => {
  const codex = routeDefinitions.find((route) => route.key === 'codex')
  const openai = routeDefinitions.find((route) => route.key === 'openai')

  assert.equal(codex.fallback.model, 'gpt-5.6-sol')
  assert.equal(openai.fallback.model, 'gpt-5.6-terra')
  for (const model of ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna']) {
    assert.ok(codex.modelPresets.includes(model))
    assert.ok(openai.modelPresets.includes(model))
  }
})

test('DeepSeek Flash is available to the Codex gateway', () => {
  const codex = routeDefinitions.find((route) => route.key === 'codex')
  const deepseek = gatewayPlatformPresets.find((preset) => preset.key === 'deepseek')
  const flash = deepseek.models.find((model) => model.routeModels?.openai === 'deepseek-v4-flash')

  assert.ok(codex.modelPresets.includes('deepseek-v4-flash'))
  assert.equal(flash.routeModels.codex, 'deepseek-v4-flash')
})

test('inferGatewayProviderForModel keeps provider inference stable', () => {
  const cases = [
    ['claude-sonnet-4-6', 'zo'],
    ['claude-opus-4-7', 'zo'],
    ['claude-sonnet-4-5', 'anthropic'],
    ['deepseek-v4-pro', 'deepseek'],
    ['kimi-for-coding', 'kimi'],
    ['mimo-v2.5-pro', 'xiaomi'],
    ['glm-5.1', 'zhipu'],
    ['MiniMax-M2.7', 'minimax'],
    ['gemini-3-pro-preview', 'gemini'],
    ['auto:balance', 'tokenrouter'],
    ['openai/gpt-5.4', 'openrouter'],
    ['custom-model', 'custom'],
    ['gpt-5.6-sol', 'openai'],
    ['gpt-5.4', 'openai'],
  ]

  for (const [model, provider] of cases) {
    assert.equal(inferGatewayProviderForModel(model), provider)
  }
})

test('routeStrategyChain orders known providers and preserves unknown providers', () => {
  assert.deepEqual(
    routeStrategyChain(['openai', 'deepseek', 'prem', 'local-gateway'], 'cost'),
    ['deepseek', 'prem', 'openai', 'local-gateway'],
  )
  assert.deepEqual(routeStrategyChain(['prem', 'openai', 'zo'], 'speed'), ['prem', 'zo', 'openai'])
})
