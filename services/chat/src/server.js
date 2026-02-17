import fastify from 'fastify';
import { Server as SocketIOServer } from 'socket.io';

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

    this.registerRoutes();
  }

  registerRoutes() {
    this.app.get('/health', async () => {
      return { status: 'ok', service: 'chat' };
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

  onConnection(socket) {
    this.app.log.info(`User ${socket.username} connected via WebSocket`);

    socket.emit('welcome', {
      message: `Welcome ${socket.username}!`,
      username: socket.username,
      timestamp: new Date().toISOString(),
    });

    socket.on('message', (data) => this.onMessage(socket, data));

    socket.on('disconnect', () => {
      this.app.log.info(`User ${socket.username} disconnected`);
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
