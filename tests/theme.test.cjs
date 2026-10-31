const {test} = require('node:test');
const assert = require('node:assert/strict');
const theme = require('../source/static/js/theme.js');

test('Helsinki night boundaries are independent of visitor timezone', () => {
  const resolve = date => theme.resolve({policy: 'night', start: '21:00', end: '07:00', now: new Date(date)});
  assert.equal(resolve('2026-10-31T18:59:00Z'), 'light');
  assert.equal(resolve('2026-10-31T19:00:00Z'), 'dark');
  assert.equal(resolve('2026-11-01T04:59:00Z'), 'dark');
  assert.equal(resolve('2026-11-01T05:00:00Z'), 'light');
  assert.equal(resolve('2026-07-01T18:00:00Z'), 'dark');
});
test('DST transition and a day interval use the intended local clock', () => {
  assert.equal(theme.helsinkiMinute(new Date('2026-10-25T00:30:00Z')), 210);
  assert.equal(theme.helsinkiMinute(new Date('2026-10-25T01:30:00Z')), 210);
  assert.equal(theme.inSchedule(13*60, 12*60, 15*60), true);
  assert.equal(theme.inSchedule(15*60, 12*60, 15*60), false);
  assert.equal(theme.inSchedule(12*60, 12*60, 12*60), false);
});
test('visitor choice overrides the administrator default; Auto restores it', () => {
  assert.equal(theme.resolve({preference: 'dark', policy: 'light'}), 'dark');
  assert.equal(theme.resolve({preference: 'light', policy: 'dark'}), 'light');
  assert.equal(theme.resolve({preference: 'auto', policy: 'dark'}), 'dark');
  assert.equal(theme.resolve({policy: 'system', systemDark: true}), 'dark');
  assert.equal(theme.resolve({policy: 'system', systemDark: false}), 'light');
});
test('invalid preferences, hours and unavailable Intl fail safely', () => {
  assert.equal(theme.preference('<script>'), 'auto');
  assert.equal(theme.minute('24:00', 1260), 1260);
  assert.equal(theme.minute('00:00', 1260), 0);
  assert.equal(theme.resolve({policy: 'night', now: new Date('invalid')}), 'light');
  assert.equal(theme.resolve({policy: 'unsupported'}), 'light');
});
