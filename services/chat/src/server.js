import fastify from 'fastify';
import { Server as SocketIOServer } from 'socket.io';
import { PresenceClient } from './presence-client.js';

/**
 * @typedef {{ text: string }} MessageData
 * @typedef {{ from: string, data: MessageData, timestamp: string }} MessagePayload
 * @typedef {{ message: string, timestamp: string }} WelcomePayload
 */

class Server {
  constructor({ port = process.env.PORT || 3000 } = {}) {
    this.port = port;
    this.app = fastify({ logger: true });
    this.io = null;
    this.presence = new PresenceClient();

    this.registerRoutes();
  }

  registerRoutes() {
    this.app.get('/health', async () => {
      return { status: 'ok', service: 'chat' };
    });

    // Presence proxy endpoints
    this.app.get('/presence/online', async () => {
      try {
        const { usernames } = await this.presence.getOnlineUsers();
        return { online: usernames || [] };
      } catch (err) {
        this.app.log.warn(`presence proxy failed: ${err.message}`);
        return { error: 'presence_service_unavailable' };
      }
    });

    this.app.get('/presence/typing', async () => {
      try {
        const { usernames } = await this.presence.getTypingUsers();
        return { typing: usernames || [] };
      } catch (err) {
        this.app.log.warn(`presence proxy failed: ${err.message}`);
        return { error: 'presence_service_unavailable' };
      }
    });
  }

  setupSocketIO() {
    this.io = new SocketIOServer(this.app.server, {
      cors: { origin: '*', methods: ['GET', 'POST'] },
      path: '/socket.io/',
    });

    this.io.use((socket, next) => {
      const username =
        socket.handshake.auth.username ||
        socket.handshake.headers['x-authenticated-user'];

      if (!username) {
        return next(new Error('Authentication required'));
      }

      socket.username = username;
      next();
    });

    this.io.on('connection', (socket) => this.onConnection(socket));
  }

  async onConnection(socket) {
    this.app.log.info(`User ${socket.username} connected via WebSocket`);

    socket.emit('welcome', {
      message: `Welcome ${socket.username}!`,
      username: socket.username,
      timestamp: new Date().toISOString(),
    });

    try {
      const { usernames } = await this.presence.userConnected(socket.username);
      this.io.emit('presence', { online: usernames });
    } catch (err) {
      this.app.log.warn(`presence.userConnected failed: ${err.message}`);
    }

    socket.on('message', (data) => this.onMessage(socket, data));

    socket.on('typing', async (data) => {
      try {
        await this.presence.setTyping(socket.username, !!data?.isTyping);
        const { usernames } = await this.presence.getTypingUsers();
        socket.broadcast.emit('typing', { users: usernames });
      } catch (err) {
        this.app.log.warn(`presence.setTyping failed: ${err.message}`);
      }
    });

    socket.on('disconnect', async () => {
      this.app.log.info(`User ${socket.username} disconnected`);
      try {
        await this.presence.userDisconnected(socket.username);
        const { usernames } = await this.presence.getOnlineUsers();
        this.io.emit('presence', { online: usernames });
      } catch (err) {
        this.app.log.warn(`presence.userDisconnected failed: ${err.message}`);
      }
    });
  }

  /** @param {import('socket.io').Socket} socket @param {MessageData} data */
  onMessage(socket, data) {
    if (!data?.text || typeof data.text !== 'string') return;

    this.app.log.info(
      `Message from ${socket.username}: ${JSON.stringify(data)}`,
    );

    const payload = {
      from: socket.username,
      data,
      timestamp: new Date().toISOString(),
    };

    socket.broadcast.emit('message', payload);
  }

  async start() {
    await this.app.listen({ port: this.port, host: '0.0.0.0' });
    this.setupSocketIO();
    this.app.log.info(`Fastify server listening on port ${this.port}`);
    this.app.log.info(
      `Socket.io server ready on ws://localhost:${this.port}/socket.io/`,
    );
  }
}

const server = new Server();
server.start().catch((err) => {
  console.error(err);
  process.exit(1);
});
