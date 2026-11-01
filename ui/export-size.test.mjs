import test from 'node:test';
import assert from 'node:assert/strict';
import { proportionalSize } from './export-size.mjs';
test('locked export uses accepted preview ratio in either direction and rounds to a pixel', () => {
  assert.deepEqual(proportionalSize({ width: 320, height: 180 }, 'width', 641), { width: 641, height: 361 });
  assert.deepEqual(proportionalSize({ width: 320, height: 180 }, 'height', 360), { width: 640, height: 360 });
  assert.deepEqual(proportionalSize({ width: 1024, height: 1 }, 'width', 1), { width: 1, height: 1 });
});
test('locked export leaves incomplete, fractional, unsafe, and invalid inputs for normal validation', () => {
  for (const value of [0, -1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1]) assert.equal(proportionalSize({ width: 2, height: 1 }, 'width', value), null);
  assert.equal(proportionalSize({ width: 0, height: 1 }, 'width', 2), null);
});
