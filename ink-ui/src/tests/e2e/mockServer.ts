import net from 'net';
import { EventEmitter } from 'events';

export class MockEngineServer extends EventEmitter {
    private server: net.Server;
    private sockets: net.Socket[] = [];
    public port: number;

    constructor(port = 0) {
        super();
        this.port = port;
        this.server = net.createServer((socket) => {
            this.sockets.push(socket);
            this.emit('connection', socket);

            socket.on('data', (data) => {
                this.emit('data', data, socket);
            });

            socket.on('close', () => {
                this.sockets = this.sockets.filter(s => s !== socket);
            });
        });
    }

    async start(): Promise<number> {
        return new Promise((resolve) => {
            this.server.listen(this.port, () => {
                const addr = this.server.address() as net.AddressInfo;
                this.port = addr.port;
                resolve(this.port);
            });
        });
    }

    async stop() {
        return new Promise<void>((resolve) => {
            this.sockets.forEach(s => s.destroy());
            this.server.close(() => resolve());
        });
    }

    sendToAll(message: object) {
        const str = JSON.stringify(message) + '\n';
        this.sockets.forEach(s => s.write(str));
    }
}
