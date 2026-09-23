import assert from 'node:assert/strict'
import test from 'node:test'

import { resolvePrice } from './pricing.js'

test('resolvePrice recognizes GPT-6 Astra standard, cache, and long-context rates', () => {
  const price = resolvePrice(' GPT-6-Astra ')
  assert.ok(price)
  assert.deepEqual(
    [price.label, price.currency, price.input, price.output, price.cacheCreation, price.cacheRead],
    ['OpenAI GPT-6 Astra', 'USD', 10, 50, 12.5, 1],
  )
  assert.equal(price.longContextInputThreshold, 272000)
  assert.equal(price.longContextInputMultiplier, 2)
  assert.equal(price.longContextOutputMultiplier, 1.5)
  assert.equal(resolvePrice('gpt-6-astral'), null)
})

test('resolvePrice recognizes GPT-6 Sol and Luna standard, cache, and long-context rates', () => {
  assert.deepEqual(
    ['gpt-6-sol', 'gpt-6-luna'].map((model) => {
      const price = resolvePrice(model)
      return [price.label, price.currency, price.input, price.output, price.cacheCreation, price.cacheRead]
    }),
    [
      ['OpenAI GPT-6 Sol', 'USD', 2, 10, 2.5, 0.2],
      ['OpenAI GPT-6 Luna', 'USD', 0.1, 0.5, 0.125, 0.01],
    ],
  )
  for (const model of ['gpt-6-sol', 'gpt-6-luna']) {
    const price = resolvePrice(model)
    assert.equal(price.longContextInputThreshold, 272000)
    assert.equal(price.longContextInputMultiplier, 2)
    assert.equal(price.longContextOutputMultiplier, 1.5)
  }
  assert.equal(resolvePrice('gpt-6-solar'), null)
  assert.equal(resolvePrice('gpt-6-lunar'), null)
})

test('resolvePrice supports all GPT-5.6 tiers', () => {
  assert.deepEqual(
    ['gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna'].map((model) => {
      const price = resolvePrice(model)
      return [price.label, price.input, price.output, price.cacheCreation, price.cacheRead]
    }),
    [
      ['OpenAI GPT-5.6 Sol', 5, 30, 6.25, 0.5],
      ['OpenAI GPT-5.6 Terra', 2.5, 15, 3.125, 0.25],
      ['OpenAI GPT-5.6 Luna', 1, 6, 1.25, 0.1],
    ],
  )
})
