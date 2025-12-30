import { EngineClient } from "../engineClient.js";
import { PassThrough } from "stream";

const stdin = new PassThrough();
const stdout = new PassThrough();

const client = new EngineClient(stdin, stdout);

client.on("error", (err) => {
    console.log("Caught error:", err.message);
});

console.log("Writing '}' to stdout...");
stdout.write("}\n");
console.log("Done.");
