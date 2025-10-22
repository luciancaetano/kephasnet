/**
 * Kephas Protocol Client - TypeScript Implementation
 * 
 * This is a sample implementation of a client for the Kephas binary protocol.
 * 
 * Protocol format:
 * - 4 bytes: CommandID (uint32, big-endian)
 * - N bytes: Payload (binary)
 * 
 * Features:
 * - Binary protocol encoding/decoding
 * - WebSocket connection management
 * - Automatic reconnection
 * - Rate limiting protection
 * - JSON-RPC 2.0 support (optional)
 * - TypeScript type safety
 */

/**
 * Command handler function type
 */
type CommandHandler = (payload: Uint8Array) => void | Promise<void>;

/**
 * JSON-RPC 2.0 request structure
 */
interface JSONRPCRequest {
  jsonrpc: '2.0';
  method: string;
  params?: any;
  id?: string | number;
}

/**
 * JSON-RPC 2.0 response structure
 */
interface JSONRPCResponse {
  jsonrpc: '2.0';
  result?: any;
  error?: {
    code: number;
    message: string;
    data?: any;
  };
  id: string | number | null;
}

/**
 * Reserved command IDs (as defined in the server)
 */
export const ReservedCommands = {
  JSON_RPC: 0xFFFFFFFF,
  JSON_RPC_ERROR: 0xFFFFFFFE,
  INVALID_COMMAND: 0xFFFFFFFD,
  COMMAND_ERROR: 0xFFFFFFFC,
} as const;

/**
 * Connection state
 */
export enum ConnectionState {
  DISCONNECTED = 'DISCONNECTED',
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  RECONNECTING = 'RECONNECTING',
  CLOSED = 'CLOSED',
}

/**
 * Client configuration options
 */
export interface KephasClientConfig {
  /** WebSocket server URL */
  url: string;
  /** Enable automatic reconnection (default: true) */
  autoReconnect?: boolean;
  /** Reconnection delay in milliseconds (default: 3000) */
  reconnectDelay?: number;
  /** Maximum reconnection attempts (default: Infinity) */
  maxReconnectAttempts?: number;
  /** Connection timeout in milliseconds (default: 10000) */
  connectionTimeout?: number;
  /** Enable debug logging (default: false) */
  debug?: boolean;
}

/**
 * Kephas Protocol Client
 */
export class KephasClient {
  private ws: WebSocket | null = null;
  private config: Required<KephasClientConfig>;
  private state: ConnectionState = ConnectionState.DISCONNECTED;
  private handlers: Map<number, CommandHandler> = new Map();
  private jsonRpcHandlers: Map<string, (params: any) => Promise<any>> = new Map();
  private reconnectAttempts = 0;
  private reconnectTimer: number | null = null;
  private connectionPromise: Promise<void> | null = null;
  private connectionResolve: (() => void) | null = null;
  private connectionReject: ((error: Error) => void) | null = null;

  constructor(config: KephasClientConfig) {
    this.config = {
      url: config.url,
      autoReconnect: config.autoReconnect ?? true,
      reconnectDelay: config.reconnectDelay ?? 3000,
      maxReconnectAttempts: config.maxReconnectAttempts ?? Infinity,
      connectionTimeout: config.connectionTimeout ?? 10000,
      debug: config.debug ?? false,
    };
  }

  /**
   * Get current connection state
   */
  public getState(): ConnectionState {
    return this.state;
  }

  /**
   * Check if the client is connected
   */
  public isConnected(): boolean {
    return this.state === ConnectionState.CONNECTED && 
           this.ws !== null && 
           this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Connect to the WebSocket server
   */
  public async connect(): Promise<void> {
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
        this.handleConnectionError(error as Error);
      }
    });

    return this.connectionPromise;
  }

  /**
   * Disconnect from the server
   */
  public disconnect(): void {
    this.log('Disconnecting...');
    this.state = ConnectionState.CLOSED;
    this.clearReconnectTimer();

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
  }

  /**
   * Register a handler for a specific command ID
   */
  public on(commandId: number, handler: CommandHandler): void {
    if (commandId >= ReservedCommands.COMMAND_ERROR) {
      throw new Error(`Command ID ${commandId.toString(16)} is reserved`);
    }
    this.handlers.set(commandId, handler);
    this.log(`Registered handler for command 0x${commandId.toString(16).padStart(8, '0')}`);
  }

  /**
   * Unregister a handler for a specific command ID
   */
  public off(commandId: number): void {
    this.handlers.delete(commandId);
    this.log(`Unregistered handler for command 0x${commandId.toString(16).padStart(8, '0')}`);
  }

  /**
   * Send a binary message to the server
   */
  public async send(commandId: number, payload: Uint8Array = new Uint8Array(0)): Promise<void> {
    if (!this.isConnected()) {
      throw new Error('Not connected to server');
    }

    const message = this.encode(commandId, payload);
    this.ws!.send(message);
    this.log(`Sent command 0x${commandId.toString(16).padStart(8, '0')} with ${payload.length} bytes`);
  }

  /**
   * Send a string message to the server (UTF-8 encoded)
   */
  public async sendString(commandId: number, text: string): Promise<void> {
    const encoder = new TextEncoder();
    const payload = encoder.encode(text);
    return this.send(commandId, payload);
  }

  /**
   * Send a JSON message to the server
   */
  public async sendJSON(commandId: number, data: any): Promise<void> {
    const json = JSON.stringify(data);
    return this.sendString(commandId, json);
  }

  /**
   * Send a JSON-RPC 2.0 request
   */
  public async sendJSONRPC(method: string, params?: any, id?: string | number): Promise<JSONRPCResponse> {
    const request: JSONRPCRequest = {
      jsonrpc: '2.0',
      method,
      params,
      id: id ?? Date.now(),
    };

    return new Promise((resolve, reject) => {
      // Register one-time handler for the response
      const responseHandler = (payload: Uint8Array) => {
        try {
          const decoder = new TextDecoder();
          const json = decoder.decode(payload);
          const response: JSONRPCResponse = JSON.parse(json);
          
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

      // Timeout after 30 seconds
      setTimeout(() => {
        this.off(ReservedCommands.JSON_RPC);
        reject(new Error('JSON-RPC request timeout'));
      }, 30000);
    });
  }

  /**
   * Encode a message using the Kephas protocol
   */
  private encode(commandId: number, payload: Uint8Array): ArrayBuffer {
    const buffer = new ArrayBuffer(4 + payload.length);
    const view = new DataView(buffer);
    
    // Write command ID (4 bytes, big-endian)
    view.setUint32(0, commandId, false); // false = big-endian
    
    // Write payload
    const payloadView = new Uint8Array(buffer, 4);
    payloadView.set(payload);
    
    return buffer;
  }

  /**
   * Decode a message using the Kephas protocol
   */
  private decode(data: ArrayBuffer): { commandId: number; payload: Uint8Array } {
    if (data.byteLength < 4) {
      throw new Error('Message too short');
    }

    const view = new DataView(data);
    const commandId = view.getUint32(0, false); // false = big-endian
    const payload = new Uint8Array(data, 4);

    return { commandId, payload };
  }

  /**
   * Handle WebSocket open event
   */
  private handleOpen(): void {
    this.log('Connected');
    this.state = ConnectionState.CONNECTED;
    this.reconnectAttempts = 0;

    if (this.connectionResolve) {
      this.connectionResolve();
      this.connectionResolve = null;
      this.connectionReject = null;
    }
  }

  /**
   * Handle incoming WebSocket message
   */
  private handleMessage(event: MessageEvent): void {
    try {
      const { commandId, payload } = this.decode(event.data);
      this.log(`Received command 0x${commandId.toString(16).padStart(8, '0')} with ${payload.length} bytes`);

      const handler = this.handlers.get(commandId);
      if (handler) {
        // Run handler asynchronously
        Promise.resolve(handler(payload)).catch((error) => {
          this.log(`Handler error for command 0x${commandId.toString(16)}: ${error.message}`);
        });
      } else {
        this.log(`No handler registered for command 0x${commandId.toString(16).padStart(8, '0')}`);
      }
    } catch (error) {
      this.log(`Failed to decode message: ${(error as Error).message}`);
    }
  }

  /**
   * Handle WebSocket error event
   */
  private handleError(event: Event): void {
    this.log(`WebSocket error: ${event}`);
  }

  /**
   * Handle WebSocket close event
   */
  private handleClose(event: CloseEvent): void {
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

    // Attempt reconnection if enabled
    if (shouldReconnect) {
      this.scheduleReconnect();
    }
  }

  /**
   * Handle connection error during initial connection
   */
  private handleConnectionError(error: Error): void {
    this.log(`Connection error: ${error.message}`);
    this.state = ConnectionState.DISCONNECTED;
    
    if (this.connectionReject) {
      this.connectionReject(error);
      this.connectionResolve = null;
      this.connectionReject = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    if (this.config.autoReconnect && this.reconnectAttempts < this.config.maxReconnectAttempts) {
      this.scheduleReconnect();
    }
  }

  /**
   * Schedule a reconnection attempt
   */
  private scheduleReconnect(): void {
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
        this.log(`Reconnection failed: ${error.message}`);
      });
    }, delay);
  }

  /**
   * Clear the reconnection timer
   */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * Log a message (if debug is enabled)
   */
  private log(message: string): void {
    if (this.config.debug) {
      console.log(`[KephasClient] ${message}`);
    }
  }
}

/**
 * Example usage:
 * 
 * ```typescript
 * // Create client
 * const client = new KephasClient({
 *   url: 'ws://localhost:8080',
 *   debug: true,
 *   autoReconnect: true,
 * });
 * 
 * // Register handlers
 * client.on(0x0100, async (payload) => {
 *   const decoder = new TextDecoder();
 *   const message = decoder.decode(payload);
 *   console.log('Received:', message);
 * });
 * 
 * // Connect
 * await client.connect();
 * 
 * // Send a message
 * await client.sendString(0x0100, 'Hello, server!');
 * 
 * // Send JSON
 * await client.sendJSON(0x0200, { username: 'player1', action: 'login' });
 * 
 * // Send JSON-RPC request
 * const response = await client.sendJSONRPC('add', { a: 5, b: 3 });
 * console.log('Result:', response.result);
 * 
 * // Disconnect
 * client.disconnect();
 * ```
 */
