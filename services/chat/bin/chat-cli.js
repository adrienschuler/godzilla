#!/usr/bin/env node

// Interactive Socket.io chat client. Authenticates via a saved session file
// and provides a readline-based prompt for sending/receiving messages.
// Usage: chat-cli.js <session-file> [discussion-id]

import io from 'socket.io-client';
import readline from 'readline';
import pc from 'picocolors';
import { loadSession } from '../src/session.js';
import { PresenceClient } from '../src/presence-client.js';

function formatTypingText(others) {
  if (others.length === 0) return '';
  const verb = others.length === 1 ? 'is' : 'are';
  return pc.dim(pc.italic(`  ${others.join(', ')} ${verb} typing...`));
}

class ChatCli {
  constructor(endpoint, sessionFile, discussionId) {
    this.endpoint = endpoint;
    this.discussionId = discussionId;
    const { cookie, username } = loadSession(sessionFile);
    this.sessionCookie = cookie;
    this.username = username;

    // Use HTTP proxy for presence when running remotely, direct gRPC when local
    const presenceHost = process.env.PRESENCE_HOST;
    if (presenceHost) {
      this.presence = new PresenceClient({ host: presenceHost });
    } else {
      // When running against remote server, use HTTP proxy endpoints
      this.presence = null;
      this.presenceEndpoint = endpoint;
    }

    this.socket = io(this.endpoint, {
      path: '/socket.io/',
      extraHeaders: this.sessionCookie ? { Cookie: this.sessionCookie } : {},
      transports: ['polling', 'websocket'],
    });

    this.rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      prompt: '',
    });

    this.typingStatus = '';
    this.typingUsers = [];
    this.lastTypingSent = 0;
    this.onlineUsers = new Set();

    this.bindSocketEvents();
    this.bindInputEvents();
    this.startTypingPoll();
    this.bindProcessEvents();
  }

  formatTime(isoString) {
    return pc.dim(
      new Date(isoString || Date.now()).toLocaleTimeString('en-GB', {
        hour12: false,
      }),
    );
  }

  refreshPrompt() {
    this.rl.setPrompt(this.username ? `${pc.blue(this.username)} ` : '');
  }

  clearPromptArea() {
    if (this.typingStatus) {
      readline.moveCursor(process.stdout, 0, -1);
      readline.clearLine(process.stdout, 0);
    }
    readline.cursorTo(process.stdout, 0);
    readline.clearLine(process.stdout, 0);
  }

  printLine(timestamp, ...args) {
    this.clearPromptArea();
    console.log(this.formatTime(timestamp), ...args);
    this.renderTypingStatus();
    this.rl.prompt(true);
  }

  renderTypingStatus() {
    if (this.typingStatus) {
      process.stdout.write(this.typingStatus + '\n');
    }
  }

  updateTypingStatus(usernames) {
    const others = usernames.filter((u) => u !== this.username);
    const newStatus = formatTypingText(others);

    // Only update if the typing status actually changed
    if (newStatus !== this.typingStatus) {
      // Must clear based on oldStatus before updating, since clearPromptArea
      // checks this.typingStatus to decide whether to move up a line.
      this.clearPromptArea();
      this.typingStatus = newStatus;
      this.typingUsers = usernames;
      this.renderTypingStatus();
      this.rl.prompt(true);
    }
  }

  removeTypingUser(username) {
    if (!this.typingStatus) return;
    // Re-derive the list without the user who just sent a message
    this.updateTypingStatus(this.typingUsers.filter((u) => u !== username));
  }

  async loadHistory() {
    if (!this.discussionId) return;
    try {
      const url = `${this.endpoint}/discussion/${this.discussionId}/messages?limit=10`;
      const res = await fetch(url, {
        headers: { Cookie: this.sessionCookie },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const { messages } = await res.json();
      // Messages come newest-first, reverse to display chronologically
      for (const msg of messages.reverse()) {
        const isMe = msg.user_id === this.username;
        const name = isMe ? pc.blue(msg.user_id) : pc.green(msg.user_id);
        console.log(this.formatTime(msg.created_at), name, msg.text);
      }
      if (messages.length > 0) {
        console.log(pc.dim('--- end of history ---'));
      }
    } catch (err) {
      console.log(pc.dim(`Could not load history: ${err.message}`));
    }
  }

  async postMessage(text) {
    if (!this.discussionId) return;
    try {
      const url = `${this.endpoint}/discussion/${this.discussionId}/messages`;
      await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Cookie: this.sessionCookie,
        },
        body: JSON.stringify([{ text }]),
      });
    } catch {
      // Silently ignore history persistence errors to not disrupt chat
    }
  }

  async cleanup() {
    clearInterval(this.typingPoll);
    if (this.presence) {
      await this.presence.userDisconnected(this.username).catch(() => {});
    }
  }

  // HTTP proxy methods for presence
  async getOnlineUsers() {
    if (this.presence) {
      const { usernames } = await this.presence.getOnlineUsers().catch(() => ({ usernames: [] }));
      return usernames || [];
    } else {
      try {
        const url = `${this.presenceEndpoint}/presence/online`;
        const res = await fetch(url, {
          headers: this.sessionCookie ? { Cookie: this.sessionCookie } : {},
        });
        if (!res.ok) return [];
        const { online } = await res.json();
        return online || [];
      } catch {
        return [];
      }
    }
  }

  async getTypingUsers() {
    if (this.presence) {
      const { usernames } = await this.presence.getTypingUsers().catch(() => ({ usernames: [] }));
      return usernames || [];
    } else {
      try {
        const url = `${this.presenceEndpoint}/presence/typing`;
        const res = await fetch(url, {
          headers: this.sessionCookie ? { Cookie: this.sessionCookie } : {},
        });
        if (!res.ok) return [];
        const { typing } = await res.json();
        return typing || [];
      } catch {
        return [];
      }
    }
  }

  async userConnected(username) {
    if (this.presence) {
      await this.presence.userConnected(username).catch(() => {});
    }
    // Note: When using HTTP proxy, userConnected is handled by the socket welcome event
  }

  async userDisconnected(username) {
    if (this.presence) {
      await this.presence.userDisconnected(username).catch(() => {});
    }
    // Note: When using HTTP proxy, userDisconnected is handled by socket disconnect
  }

  async setTyping(username, isTyping) {
    if (this.presence) {
      await this.presence.setTyping(username, isTyping).catch(() => {});
    } else {
      // When using HTTP proxy, send typing events via WebSocket
      try {
        this.socket.emit('typing', { isTyping });
      } catch {
        // Silently ignore typing errors
      }
    }
  }

  bindSocketEvents() {
    this.socket.on('connect', () => {
      console.log('Connected! Socket ID:', this.socket.id);
      this.refreshPrompt();
      this.rl.prompt();
    });

    this.socket.on('connect_error', (error) => {
      console.error('Connection error:', error.message);
      if (
        error.message.includes('401') ||
        error.message.includes('Unauthorized')
      ) {
        console.error('Authentication failed - session cookie may be invalid');
      }
      process.exit(1);
    });

    this.socket.on('welcome', async (data) => {
      if (data.username) this.username = data.username;
      this.refreshPrompt();
      await this.loadHistory();
      try {
        await this.userConnected(this.username);
        const usernames = await this.getOnlineUsers();
        if (usernames?.length) {
          this.onlineUsers = new Set(usernames);
        }
      } catch {
        this.printLine(null, pc.dim('Presence service unavailable'));
      }
    });

    this.socket.on('presence', (data) => {
      const current = new Set(data.online || []);
      for (const user of current) {
        if (!this.onlineUsers.has(user) && user !== this.username) {
          this.printLine(null, pc.yellow(`→ ${user} joined the chat`));
        }
      }
      for (const user of this.onlineUsers) {
        if (!current.has(user) && user !== this.username) {
          this.printLine(null, pc.yellow(`← ${user} left the chat`));
        }
      }
      this.onlineUsers = current;
    });

    this.socket.on('message', (data) => {
      this.printLine(data.timestamp, pc.green(data.from), data.data.text);
      this.removeTypingUser(data.from);
    });

    this.socket.on('typing', (data) => {
      if (data.users) {
        this.updateTypingStatus(data.users);
      }
    });

    this.socket.on('disconnect', async (reason) => {
      await this.cleanup();
      this.printLine(null, 'Disconnected:', reason);
      this.rl.close();
      process.exit(0);
    });
  }

  bindInputEvents() {
    readline.emitKeypressEvents(process.stdin);
    process.stdin.on('keypress', () => {
      const now = Date.now();
      // Send typing events more frequently (every 500ms) to keep status alive
      // This prevents the presence service from cleaning up while user is still typing
      if (now - this.lastTypingSent > 500) {
        this.lastTypingSent = now;
        this.setTyping(this.username, true).catch(() => {});
      }
    });

    this.rl.on('line', async (line) => {
      const text = line.trim();
      if (!text) return this.rl.prompt();
      readline.moveCursor(process.stdout, 0, -1);
      this.printLine(null, pc.blue(this.username), text);
      this.socket.emit('message', { text });
      this.postMessage(text);
      await this.setTyping(this.username, false).catch(() => {});
    });
  }

  startTypingPoll() {
    this.typingPoll = setInterval(async () => {
      try {
        // Only poll if we currently have typing users shown
        // This reduces unnecessary network requests
        if (this.typingStatus) {
          const usernames = await this.getTypingUsers();
          // Update typing status (updateTypingStatus will handle the comparison)
          this.updateTypingStatus(usernames || []);
        }
        // If no typing status is shown, no need to poll
      } catch (err) {
        // If there's an error and we have typing users shown, clear them
        if (this.typingStatus) {
          this.updateTypingStatus([]);
        }
      }
    }, 2000); // Further reduced frequency to 2000ms (2 seconds)
  }

  bindProcessEvents() {
    process.on('SIGINT', async () => {
      console.log('\nDisconnecting...');
      await this.cleanup();
      this.socket.disconnect();
      process.exit(0);
    });
  }
}

// CLI entry point
if (!process.argv[2]) {
  console.error('Usage: chat-cli.js <session-file> [discussion-id]');
  process.exit(1);
}

try {
  const sessionFile = process.argv[2];
  const discussionId = process.argv[3] || process.env.DISCUSSION_ID;
  const endpoint = process.env.SERVER_URL || 'http://127.0.0.1:8080';
  const client = new ChatCli(endpoint, sessionFile, discussionId);
  if (client.sessionCookie) console.log('Cookie:', client.sessionCookie);
  if (discussionId) console.log('Discussion:', discussionId);
} catch (err) {
  console.error(`Failed to start: ${err.message}`);
  process.exit(1);
}
