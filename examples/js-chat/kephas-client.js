/**
 * Kephas Protocol Client - Browser JavaScript Implementation
 *
 * Protocol format:
 * - 4 bytes: CommandID (uint32, big-endian)
 * - N bytes: Payload (binary)
 *
 * This is a direct plain-JS conversion of the TypeScript implementation.
 */

/* Reserved command IDs (as defined in the server) */
const ReservedCommands = {
  JSON_RPC: 0xFFFFFFFF,
  JSON_RPC_ERROR: 0xFFFFFFFE,
  INVALID_COMMAND: 0xFFFFFFFD,
  COMMAND_ERROR: 0xFFFFFFFC,
};

/* Connection state */
const ConnectionState = {
  DISCONNECTED: 'DISCONNECTED',
  CONNECTING: 'CONNECTING',
  CONNECTED: 'CONNECTED',
  RECONNECTING: 'RECONNECTING',
  CLOSED: 'CLOSED',
};

class KephasClient {
  constructor(config) {
    this.ws = null;
    this.config = {
      url: config.url,
      autoReconnect: config.autoReconnect !== undefined ? config.autoReconnect : true,
      reconnectDelay: config.reconnectDelay !== undefined ? config.reconnectDelay : 3000,
      maxReconnectAttempts: config.maxReconnectAttempts !== undefined ? config.maxReconnectAttempts : Infinity,
      connectionTimeout: config.connectionTimeout !== undefined ? config.connectionTimeout : 10000,
      debug: config.debug !== undefined ? config.debug : false,
    };
    this.state = ConnectionState.DISCONNECTED;
    this.handlers = new Map();
    this.jsonRpcHandlers = new Map();
    this.reconnectAttempts = 0;
    this.reconnectTimer = null;
    this.connectionPromise = null;
    this.connectionResolve = null;
    this.connectionReject = null;
  }

  getState() {
    return this.state;
  }

  isConnected() {
    return this.state === ConnectionState.CONNECTED &&
           this.ws !== null &&
           this.ws.readyState === WebSocket.OPEN;
  }

  connect() {
    if (this.state === ConnectionState.CONNECTED || this.state === ConnectionState.CONNECTING) {
      this.log('Already connected or connecting');
      return this.connectionPromise || Promise.resolve();
    }

    this.state = ConnectionState.CONNECTING;
    this.log(`Connecting to ${this.config.url}...`);

    this.connectionPromise = new Promise((resolve, reject) => {
      this.connectionResolve = resolve;
      this.connectionReject = reject;

      const timeout = setTimeout(() => {
        if (this.state === ConnectionState.CONNECTING) {
          this.handleConnectionError(new Error('Connection timeout'));
        }
      }, this.config.connectionTimeout);

      try {
        this.ws = new WebSocket(this.config.url);
        this.ws.binaryType = 'arraybuffer';

        this.ws.onopen = () => {
          clearTimeout(timeout);
          this.handleOpen();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event);
        };

        this.ws.onerror = (event) => {
          clearTimeout(timeout);
          this.handleError(event);
        };

        this.ws.onclose = (event) => {
          clearTimeout(timeout);
          this.handleClose(event);
        };
      } catch (error) {
        clearTimeout(timeout);
        this.handleConnectionError(error);
      }
    });

    return this.connectionPromise;
  }

  disconnect() {
    this.log('Disconnecting...');
    this.state = ConnectionState.CLOSED;
    this.clearReconnectTimer();

    if (this.ws) {
      try { this.ws.close(1000, 'Client disconnect'); } catch (e) { /* ignore */ }
      this.ws = null;
    }
  }

  on(commandId, handler) {
    if (commandId >= ReservedCommands.COMMAND_ERROR) {
      throw new Error(`Command ID ${commandId.toString(16)} is reserved`);
    }
    this.handlers.set(commandId, handler);
    this.log(`Registered handler for command 0x${commandId.toString(16).padStart(8, '0')}`);
  }

  off(commandId) {
    this.handlers.delete(commandId);
    this.log(`Unregistered handler for command 0x${commandId.toString(16).padStart(8, '0')}`);
  }

  async send(commandId, payload) {
    payload = payload || new Uint8Array(0);

    if (!this.isConnected()) {
      throw new Error('Not connected to server');
    }

    const message = this.encode(commandId, payload);
    this.ws.send(message);
    this.log(`Sent command 0x${commandId.toString(16).padStart(8, '0')} with ${payload.length} bytes`);
  }

  async sendString(commandId, text) {
    const encoder = new TextEncoder();
    const payload = encoder.encode(text);
    return this.send(commandId, payload);
  }

  async sendJSON(commandId, data) {
    const json = JSON.stringify(data);
    return this.sendString(commandId, json);
  }

  async sendJSONRPC(method, params, id) {
    const request = {
      jsonrpc: '2.0',
      method: method,
      params: params,
      id: id !== undefined ? id : Date.now(),
    };

    return new Promise((resolve, reject) => {
      const responseHandler = (payload) => {
        try {
          const decoder = new TextDecoder();
          const json = decoder.decode(payload);
          const response = JSON.parse(json);

          if (response.id === request.id) {
            this.off(ReservedCommands.JSON_RPC);
            if (response.error) {
              reject(new Error(response.error.message));
            } else {
              resolve(response);
            }
          }
        } catch (error) {
          reject(error);
        }
      };

      this.on(ReservedCommands.JSON_RPC, responseHandler);
      this.sendJSON(ReservedCommands.JSON_RPC, request).catch(reject);

      setTimeout(() => {
        this.off(ReservedCommands.JSON_RPC);
        reject(new Error('JSON-RPC request timeout'));
      }, 30000);
    });
  }

  encode(commandId, payload) {
    const buffer = new ArrayBuffer(4 + payload.length);
    const view = new DataView(buffer);

    // Write command ID (4 bytes, big-endian)
    view.setUint32(0, commandId, false);

    // Write payload
    const payloadView = new Uint8Array(buffer, 4);
    payloadView.set(payload);

    return buffer;
  }

  decode(data) {
    // expecting ArrayBuffer (ws.binaryType = 'arraybuffer')
    if (!data || data.byteLength < 4) {
      throw new Error('Message too short');
    }

    const view = new DataView(data);
    const commandId = view.getUint32(0, false);
    const payload = new Uint8Array(data, 4);

    return { commandId, payload };
  }

  handleOpen() {
    this.log('Connected');
    this.state = ConnectionState.CONNECTED;
    this.reconnectAttempts = 0;

    if (this.connectionResolve) {
      this.connectionResolve();
      this.connectionResolve = null;
      this.connectionReject = null;
    }
  }

  handleMessage(event) {
    try {
      const { commandId, payload } = this.decode(event.data);
      this.log(`Received command 0x${commandId.toString(16).padStart(8, '0')} with ${payload.length} bytes`);

      const handler = this.handlers.get(commandId);
      if (handler) {
        Promise.resolve(handler(payload)).catch((error) => {
          this.log(`Handler error for command 0x${commandId.toString(16)}: ${error && error.message ? error.message : error}`);
        });
      } else {
        this.log(`No handler registered for command 0x${commandId.toString(16).padStart(8, '0')}`);
      }
    } catch (error) {
      this.log(`Failed to decode message: ${error && error.message ? error.message : error}`);
    }
  }

  handleError(event) {
    this.log(`WebSocket error: ${event}`);
  }

  handleClose(event) {
    this.log(`Connection closed (code: ${event.code}, reason: ${event.reason || 'none'})`);

    const wasConnected = this.state === ConnectionState.CONNECTED;
    const shouldReconnect = this.config.autoReconnect &&
                            this.state !== ConnectionState.CLOSED &&
                            this.reconnectAttempts < this.config.maxReconnectAttempts;

    this.state = ConnectionState.DISCONNECTED;
    this.ws = null;

    if (this.connectionReject && !wasConnected) {
      this.connectionReject(new Error(`Connection failed: ${event.reason || 'Unknown error'}`));
      this.connectionResolve = null;
      this.connectionReject = null;
    }

    if (shouldReconnect) {
      this.scheduleReconnect();
    }
  }

  handleConnectionError(error) {
    this.log(`Connection error: ${error.message}`);
    this.state = ConnectionState.DISCONNECTED;

    if (this.connectionReject) {
      this.connectionReject(error);
      this.connectionResolve = null;
      this.connectionReject = null;
    }

    if (this.ws) {
      try { this.ws.close(); } catch (e) { /* ignore */ }
      this.ws = null;
    }

    if (this.config.autoReconnect && this.reconnectAttempts < this.config.maxReconnectAttempts) {
      this.scheduleReconnect();
    }
  }

  scheduleReconnect() {
    if (this.reconnectTimer !== null) {
      return;
    }

    this.reconnectAttempts++;
    this.state = ConnectionState.RECONNECTING;

    const delay = this.config.reconnectDelay * Math.min(this.reconnectAttempts, 5);
    this.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})...`);

    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.connect().catch((error) => {
        this.log(`Reconnection failed: ${error && error.message ? error.message : error}`);
      });
    }, delay);
  }

  clearReconnectTimer() {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  log(message) {
    if (this.config.debug) {
      console.log(`[KephasClient] ${message}`);
    }
  }
}

/* Expose to global scope for browser usage */
window.KephasClient = KephasClient;
window.KephasReservedCommands = ReservedCommands;
window.KephasConnectionState = ConnectionState;

/* Example usage (browser):
const client = new KephasClient({
  url: 'ws://localhost:8080',
  debug: true,
});

client.on(0x0100, async (payload) => {
  const decoder = new TextDecoder();
  console.log('Received:', decoder.decode(payload));
});

await client.connect();
await client.sendString(0x0100, 'Hello, server!');
*/
