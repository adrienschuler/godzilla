#!/usr/bin/env node

// Interactive Socket.io chat client. Authenticates via a saved session file
// and provides a readline-based prompt for sending/receiving messages.
// Usage: chat-cli.js <session-file> [discussion-id]

import io from 'socket.io-client';
import readline from 'readline';
import pc from 'picocolors';
import { loadSession } from '../src/session.js';

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

    this.socket = io(this.endpoint, {
      path: '/socket.io/',
      extraHeaders: this.sessionCookie ? { Cookie: this.sessionCookie } : {},
      transports: ['websocket'],
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

  setTyping(isTyping) {
    this.socket.emit('typing', { isTyping });
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
          this.removeTypingUser(user);
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

    this.socket.on('disconnect', (reason) => {
      if (this.closing) return;
      this.printLine(null, 'Disconnected:', reason);
      this.close();
    });
  }

  bindInputEvents() {
    readline.emitKeypressEvents(process.stdin);
    process.stdin.on('keypress', (_ch, key) => {
      if (key?.name === 'return' || key?.name === 'enter') return;
      const now = Date.now();
      if (now - this.lastTypingSent > 500) {
        this.lastTypingSent = now;
        this.setTyping(true);
      }
      clearTimeout(this.typingTimeout);
      this.typingTimeout = setTimeout(() => this.setTyping(false), 2000);
    });

    this.rl.on('line', async (line) => {
      const text = line.trim();
      if (!text) return this.rl.prompt();
      readline.moveCursor(process.stdout, 0, -1);
      this.printLine(null, pc.blue(this.username), text);
      this.socket.emit('message', { text });
      this.postMessage(text);
      clearTimeout(this.typingTimeout);
      this.setTyping(false);
    });
  }

  close() {
    if (this.closing) return;
    this.closing = true;
    this.rl.close();
    this.socket.disconnect();
    process.exit(0);
  }

  bindProcessEvents() {
    process.on('SIGINT', () => {
      console.log('\nDisconnecting...');
      this.close();
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
