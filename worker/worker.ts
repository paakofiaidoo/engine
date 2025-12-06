import { Project } from "ts-morph";
import * as readline from "readline";

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false,
});

interface JsonRpcRequest {
    jsonrpc: "2.0";
    method: string;
    params: any;
    id: string | number;
}

interface JsonRpcResponse {
    jsonrpc: "2.0";
    result?: any;
    error?: {
        code: number;
        message: string;
        data?: any;
    };
    id: string | number;
}

const project = new Project({
    // We will load the user's tsconfig later or infer it
    skipAddingFilesFromTsConfig: true,
});

rl.on("line", (line) => {
    if (!line.trim()) return;

    try {
        const request: JsonRpcRequest = JSON.parse(line);
        handleRequest(request);
    } catch (error) {
        console.error("Failed to parse JSON-RPC request:", error);
    }
});

async function handleRequest(req: JsonRpcRequest) {
    try {
        let result;
        switch (req.method) {
            case "ping":
                result = "pong";
                break;
            case "create_component":
                // TODO: Implement create component logic
                result = { success: true, message: "Component created (mock)" };
                break;
            case "install_plugin":
                // TODO: Implement plugin installation logic (AST)
                result = { success: true, message: `Plugin ${req.params.plugin_name} installed (mock)` };
                break;
            case "configure_cms":
                // TODO: Implement CMS configuration logic (AST)
                result = { success: true, message: `CMS ${req.params.cms_type} configured (mock)` };
                break;
            default:
                throw { code: -32601, message: "Method not found" };
        }

        const response: JsonRpcResponse = {
            jsonrpc: "2.0",
            result,
            id: req.id,
        };
        console.log(JSON.stringify(response));
    } catch (error: any) {
        const response: JsonRpcResponse = {
            jsonrpc: "2.0",
            error: {
                code: error.code || -32000,
                message: error.message || "Internal error",
            },
            id: req.id,
        };
        console.log(JSON.stringify(response));
    }
}

// Signal readiness
console.error("Juki Worker Started");
