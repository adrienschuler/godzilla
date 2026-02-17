import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
// In Docker: /app/proto/presence.proto. Locally: ../../proto from chat root.
const PROTO_PATH = resolve(
  __dirname,
  process.env.PROTO_DIR || '../../../proto',
  'presence.proto',
);

const packageDef = protoLoader.loadSync(PROTO_PATH, {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});
const proto = grpc.loadPackageDefinition(packageDef).presence;

export class PresenceClient {
  constructor({ host = process.env.PRESENCE_HOST || 'localhost:50051' } = {}) {
    this.client = new proto.PresenceService(
      host,
      grpc.credentials.createInsecure(),
    );
  }

  userConnected(username) {
    return this._call('userConnected', { username });
  }

  userDisconnected(username) {
    return this._call('userDisconnected', { username });
  }

  setTyping(username, isTyping) {
    return this._call('setTyping', { username, isTyping });
  }

  getOnlineUsers() {
    return this._call('getOnlineUsers', {});
  }

  getTypingUsers() {
    return this._call('getTypingUsers', {});
  }

  _call(method, req) {
    return new Promise((resolve, reject) => {
      this.client[method](req, (err, res) => {
        if (err) reject(err);
        else resolve(res);
      });
    });
  }
}
