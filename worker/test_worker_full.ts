import { handleRequest } from "./worker";
import * as fs from "fs";
import * as path from "path";

// Mock the console methods to capture output
// const originalConsoleLog = console.log;
// const originalConsoleError = console.error;
const originalStdoutWrite = process.stdout.write.bind(process.stdout);

async function main() {
    const filePath = "/Users/flow/dev/juki-builder/projects/test-project/app/page.tsx";
    console.log("Testing with code from:", filePath);

    try {
        const code = fs.readFileSync(filePath, "utf-8");

        const mockReq = {
            jsonrpc: "2.0" as const,
            method: "parse_jsx",
            params: { code },
            id: "test-id"
        };


        // Capture stdout to verify response
        let output = "";
        process.stdout.write = (chunk: any) => {
            output += chunk.toString();
            return true;
        };

        await handleRequest(mockReq);

        // Restore stdout
        process.stdout.write = originalStdoutWrite;

        console.log("Worker Output:", output);

        try {
            const response = JSON.parse(output.trim());
            if (response.error) {
                console.error("Test Failed: Worker returned error", response.error);
                process.exit(1);
            }

            if (!response.result || !Array.isArray(response.result)) {
                console.error("Test Failed: Invalid result format", response);
                process.exit(1);
            }

            if (response.result.length === 0) {
                console.error("Test Failed: Parsed result is empty");
                process.exit(1);
            }

            console.log("Test Passed: Successfully parsed JSX");
            console.log("Result Item Count:", response.result.length);
            console.log("First Item Name:", response.result[0]?.name);

        } catch (e) {
            console.error("Test Failed: Invalid JSON response", output);
            process.exit(1);
        }

    } catch (err) {
        process.stdout.write = originalStdoutWrite;
        console.error("Test Failed with exception:", err);
        process.exit(1);
    }
}

main();