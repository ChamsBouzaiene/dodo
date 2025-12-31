import { EventEmitter } from "node:events";
import readline from "node:readline";
import type { Readable, Writable } from "node:stream";
import { serializeCommand, type Command, type Event, type StatusEvent } from "./protocol.js";
import { debugLog } from "./utils/debugLogger.js";

/**
 * Options for starting a new session with the engine.
 */
export type StartSessionOptions = {
  sessionId?: string;
  repoRoot: string;
};

/**
 * Events emitted by the EngineClient.
 */
export type EngineClientEvents = {
  event: [Event];
  close: [];
  error: [Error];
};

/**
 * Type-safe event emitter interface.
 */
export interface TypedEventEmitter<TEvents extends Record<string, unknown[]>> {
  on<TKey extends keyof TEvents>(event: TKey, listener: (...args: TEvents[TKey]) => void): this;
  off<TKey extends keyof TEvents>(event: TKey, listener: (...args: TEvents[TKey]) => void): this;
  once<TKey extends keyof TEvents>(event: TKey, listener: (...args: TEvents[TKey]) => void): this;
}

/**
 * EngineClient handles communication with the backend engine.
 * It sends commands via stdin and listens for events via stdout (JSON lines).
 */
export class EngineClient
  extends EventEmitter
  implements TypedEventEmitter<EngineClientEvents> {
  private pendingSession?: {
    resolve: (id: string) => void;
    reject: (err: Error) => void;
  };

  private currentSessionId?: string;
  private rl?: readline.Interface;
  private closed = false;

  constructor(private readonly stdin: Writable, stdout: Readable) {
    super();

    debugLog.state("EngineClient", "initialized");

    this.rl = readline.createInterface({ input: stdout });
    this.rl.on("line", (line) => {
      if (!line.trim() || this.closed) {
        return;
      }

      // Log raw traffic for debugging
      debugLog.log({ category: "traffic", component: "EngineClient", message: "IN", data: { line } });

      try {
        const evt: Event = JSON.parse(line);
        this.handleEvent(evt);
        this.emit("event", evt);
      } catch (err) {
        const error = err instanceof Error ? err : new Error(`Failed to parse engine event: ${String(err)}`);
        debugLog.error("EngineClient", "ParseError", error);
        this.emit("error", error);
      }
    });

    this.rl.on("close", () => {
      this.closed = true;
      debugLog.state("EngineClient", "closed");
      if (this.pendingSession) {
        const err = new Error("Engine closed before session became ready");
        this.pendingSession.reject(err);
        this.pendingSession = undefined;
      }
      this.emit("close");
    });
  }

  /**
   * Logs a user-facing issue or error.
   */
  logIssue(message: string) {
    debugLog.error("EngineClient", "Issue", new Error(message));
  }

  /**
   * Closes the client and the underlying readline interface.
   */
  close(): void {
    if (this.closed) return;
    this.closed = true;
    if (this.rl) {
      this.rl.close();
      this.rl = undefined;
    }
  }

  /**
   * Starts a new session with the engine.
   * Resolves when the session is ready.
   */
  async startSession(opts: StartSessionOptions): Promise<string> {
    if (this.pendingSession) {
      throw new Error("Session already starting");
    }

    if (!opts.sessionId) {
      this.currentSessionId = undefined;
    }

    const command: Command = {
      type: "start_session",
      session_id: opts.sessionId,
      repo_root: opts.repoRoot,
      meta: {},
    };

    debugLog.command("EngineClient", "start_session", command);

    const sessionId = await new Promise<string>((resolve, reject) => {
      this.pendingSession = { resolve, reject };
      this.sendCommand(command).catch((err) => {
        this.pendingSession = undefined;
        reject(err);
      });
    });

    this.currentSessionId = sessionId;
    return sessionId;
  }

  /**
   * Sends a user message to the engine.
   */
  sendUserMessage(sessionId: string, message: string): void {
    const command: Command = {
      type: "user_message",
      session_id: sessionId,
      message,
    };
    debugLog.command("EngineClient", "user_message", { sessionId }); // Omit message content for privacy if needed, or include attempts
    void this.sendCommand(command);
  }

  /**
   * Returns the current session ID, if any.
   */
  getSessionId(): string | undefined {
    return this.currentSessionId;
  }

  /**
   * Low-level method to send a command object.
   */
  public async sendCommand(command: Command): Promise<void> {
    if (this.closed) {
      throw new Error("Engine client is closed");
    }
    const payload = serializeCommand(command);

    // Log outgoing traffic
    debugLog.log({ category: "traffic", component: "EngineClient", message: "OUT", data: { payload } });

    await new Promise<void>((resolve, reject) => {
      this.stdin.write(payload + "\n", (err) => {
        if (err) {
          debugLog.error("EngineClient", "WriteError", err);
          reject(err);
        } else {
          resolve();
        }
      });
    });
  }

  private handleEvent(event: Event) {
    if (event.type === "status") {
      this.handleStatusEvent(event);
    }
  }

  private handleStatusEvent(event: StatusEvent) {
    if (event.status === "session_ready" && event.session_id) {
      if (this.pendingSession) {
        debugLog.state("EngineClient", "session_ready", { sessionId: event.session_id });
        this.pendingSession.resolve(event.session_id);
        this.pendingSession = undefined;
      }
    }
  }
}
