import fs from 'fs';

/**
 * @typedef {{ name: string, value: string }} SessionCookie
 * @typedef {{ cookies: SessionCookie[] }} SessionFile
 * @typedef {{ cookie: string, username: string }} SessionInfo
 */

/**
 * Parse a session file and extract the auth cookie and username.
 * @param {string} filePath
 * @returns {SessionInfo}
 */
export function loadSession(filePath) {
  const raw = fs.readFileSync(filePath, 'utf8').trim();
  const session = JSON.parse(raw);
  const cookie = (session.cookies || []).find((c) => c.name === 'auth_token');
  if (!cookie) return { cookie: '', username: '' };

  const cookieStr = `${cookie.name}=${cookie.value}`;
  const parts = cookie.value.split('.');
  if (parts.length < 3) return { cookie: cookieStr, username: '' };
  const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString());
  return { cookie: cookieStr, username: payload.username };
}
