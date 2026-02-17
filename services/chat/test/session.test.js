import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'fs';
import os from 'os';
import path from 'path';
import { loadSession } from '../src/session.js';

/** Build a fake JWT with the given payload */
function fakeJWT(payload) {
  const header = Buffer.from('{"alg":"HS256"}').toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  return `${header}.${body}.fake-signature`;
}

describe('loadSession', () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'session-test-'));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true });
  });

  function writeSession(data) {
    const filePath = path.join(tmpDir, 'session.json');
    fs.writeFileSync(filePath, JSON.stringify(data));
    return filePath;
  }

  it('extracts cookie and username from a valid session', () => {
    const token = fakeJWT({ username: 'alice' });
    const filePath = writeSession({
      cookies: [{ name: 'auth_token', value: token }],
    });

    const result = loadSession(filePath);

    assert.equal(result.cookie, `auth_token=${token}`);
    assert.equal(result.username, 'alice');
  });

  it('returns empty strings when no auth_token cookie exists', () => {
    const filePath = writeSession({
      cookies: [{ name: 'other', value: 'abc' }],
    });

    const result = loadSession(filePath);

    assert.equal(result.cookie, '');
    assert.equal(result.username, '');
  });

  it('returns empty strings when cookies array is missing', () => {
    const filePath = writeSession({});

    const result = loadSession(filePath);

    assert.equal(result.cookie, '');
    assert.equal(result.username, '');
  });

  it('returns cookie but empty username when token has fewer than 3 parts', () => {
    const filePath = writeSession({
      cookies: [{ name: 'auth_token', value: 'not-a-jwt' }],
    });

    const result = loadSession(filePath);

    assert.equal(result.cookie, 'auth_token=not-a-jwt');
    assert.equal(result.username, '');
  });

  it('throws on missing file', () => {
    assert.throws(() => loadSession('/nonexistent/path.json'), {
      code: 'ENOENT',
    });
  });

  it('throws on invalid JSON', () => {
    const filePath = path.join(tmpDir, 'bad.json');
    fs.writeFileSync(filePath, 'not json');

    assert.throws(() => loadSession(filePath));
  });
});
