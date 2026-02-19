import { Project, SyntaxKind, JsxOpeningElement, JsxSelfClosingElement, JsxText, JsxExpression, JsxElement } from "ts-morph";
import * as readline from "readline";
import { AnyCanvasItem, ElementCanvasItem } from "./types";
import { randomUUID } from "crypto";

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
    useInMemoryFileSystem: true,
    compilerOptions: {
        jsx: 4, // ReactJSX
        allowJs: true,
    },
});

export function astNodeToCanvasItem(node: any): AnyCanvasItem | string | null {
    const kind = node.getKind();

    if (kind === SyntaxKind.JsxText) {
        const text = node.getText().trim();
        return text.length > 0 ? text : null;
    }

    if (kind === SyntaxKind.JsxElement || kind === SyntaxKind.JsxSelfClosingElement) {
        const opening = kind === SyntaxKind.JsxElement ? node.getOpeningElement() : node;
        const tagName = opening.getTagNameNode().getText();

        const props: Record<string, any> = {};
        opening.getAttributes().forEach((attr: any) => {
            if (attr.getKind() === SyntaxKind.JsxAttribute) {
                const nameNode = attr.getNameNode?.();
                const name = nameNode ? nameNode.getText() : (typeof attr.getName === 'function' ? attr.getName() : "unknown");

                if (name === "unknown") return;

                const initializer = attr.getInitializer();
                if (!initializer) {
                    props[name] = true;
                } else if (initializer.getKind() === SyntaxKind.StringLiteral) {
                    props[name] = initializer.getLiteralValue();
                } else if (initializer.getKind() === SyntaxKind.JsxExpression) {
                    const expr = initializer.getExpression();
                    if (expr?.getKind() === SyntaxKind.ObjectLiteralExpression) {
                        props[name] = "{object}";
                    } else if (expr?.getKind() === SyntaxKind.StringLiteral) {
                        props[name] = expr.getLiteralValue();
                    } else if (expr?.getKind() === SyntaxKind.NumericLiteral) {
                        props[name] = expr.getLiteralValue();
                    } else {
                        props[name] = `{${expr?.getText() || "expression"}}`;
                    }
                }
            } else if (attr.getKind() === SyntaxKind.JsxSpreadAttribute) {
                // Skip spread props for now but mark it
                props["...spread"] = true;
            }
        });

        const children: (AnyCanvasItem | string)[] = [];
        if (kind === SyntaxKind.JsxElement) {
            node.getJsxChildren().forEach((child: any) => {
                const item = astNodeToCanvasItem(child);
                if (item) children.push(item);
            });
        }

        // Check if Component (Capitalized)
        if (/^[A-Z]/.test(tagName)) {
            return {
                id: randomUUID(),
                name: tagName,
                type: 'ELEMENT',
                tag: 'div',
                props: { ...props, className: `${props.className || ''} p-2 border-dashed border-purple-500 text-purple-300 text-xs`.trim() },
                content: `Component<${tagName}>`,
            };
        }

        let content: string | AnyCanvasItem[] | undefined;
        if (children.length === 1 && typeof children[0] === 'string') {
            content = children[0];
        } else if (children.length > 0) {
            content = children.filter((c): c is AnyCanvasItem => typeof c !== 'string');
        }

        return {
            id: randomUUID(),
            name: tagName,
            type: 'ELEMENT',
            tag: tagName,
            props,
            content,
        };
    }

    if (kind === SyntaxKind.ParenthesizedExpression) {
        return astNodeToCanvasItem(node.getExpression());
    }

    if (kind === SyntaxKind.JsxFragment) {
        const children: (AnyCanvasItem | string)[] = [];
        node.getJsxChildren().forEach((child: any) => {
            const item = astNodeToCanvasItem(child);
            if (item) children.push(item);
        });
        return {
            id: randomUUID(),
            name: "Fragment",
            type: 'ELEMENT',
            tag: 'div',
            props: { className: 'border border-gray-500 border-dashed p-4' },
            content: children.filter((c): c is AnyCanvasItem => typeof c !== 'string'),
        };
    }
    if (kind === SyntaxKind.JsxExpression) {
        const expression = node.getExpression();
        const text = expression?.getText();
        if (text === "children") {
            return {
                id: randomUUID(),
                name: "Children Placeholder",
                type: 'ELEMENT',
                tag: 'div',
                props: {
                    "isChildrenPlaceholder": true,
                    className: "p-4 border-2 border-dashed border-blue-400 min-h-[50px] flex items-center justify-center text-blue-400 text-xs"
                },
                content: "Page Content Insertion Point",
            };
        }
        return `{${text || "expression"}}`;
    }

    return null;
}

export function composeItems(layout: AnyCanvasItem[], page: AnyCanvasItem[]): AnyCanvasItem[] {
    const layoutClone = JSON.parse(JSON.stringify(layout));

    let found = false;
    function walk(items: (AnyCanvasItem | string)[]) {
        for (let i = 0; i < items.length; i++) {
            const item = items[i];
            if (typeof item === 'string') continue;

            if (item.props?.isChildrenPlaceholder) {
                // Found! Replace placeholder with page content
                items.splice(i, 1, ...page);
                found = true;
                return true;
            }

            if (Array.isArray(item.content)) {
                if (walk(item.content)) return true;
            }
        }
        return false;
    }

    walk(layoutClone);

    if (!found && layoutClone.length > 0) {
        // Fallback: append to first item's content
        const root = layoutClone[0];
        if (Array.isArray(root.content)) {
            root.content.push(...page);
        } else {
            root.content = page;
        }
    }

    return layoutClone;
}

export async function handleRequest(req: JsonRpcRequest) {
    console.error(`[Worker] Handling request: ${req.method} (ID: ${req.id})`);
    try {
        let result;
        switch (req.method) {
            case "ping":
                result = "pong";
                break;
            case "parse_jsx":
                console.error(`[Worker] Parsing JSX code (length: ${req.params.code?.length})`);
                const sourceFile = project.createSourceFile(`${randomUUID()}.tsx`, req.params.code || "", { overwrite: true });

                const returnStatements = sourceFile.getDescendantsOfKind(SyntaxKind.ReturnStatement);
                console.error(`[Worker] Found ${returnStatements.length} total return statements`);

                if (returnStatements.length === 0) {
                    result = [];
                } else {
                    let mainItem: AnyCanvasItem | null = null;
                    // Try to find a valid JSX tree in any return statement
                    for (const ret of returnStatements) {
                        const expr = ret.getExpression();
                        if (expr) {
                            const item = astNodeToCanvasItem(expr);
                            if (item && typeof item !== 'string') {
                                mainItem = item;
                                console.error(`[Worker] Successfully parsed return statement into ${item.name}`);
                                break;
                            }
                        }
                    }

                    result = mainItem ? [mainItem] : [];
                }
                sourceFile.delete(); // Clean up
                break;
            case "compose":
                console.error(`[Worker] Composing layout and page`);
                const layout = req.params.layout || [];
                const page = req.params.page || [];
                result = composeItems(layout, page);
                break;
            default:
                throw { code: -32601, message: "Method not found" };
        }

        const response: JsonRpcResponse = {
            jsonrpc: "2.0",
            result,
            id: req.id,
        };
        process.stdout.write(JSON.stringify(response) + "\n");
    } catch (error: any) {
        console.error(`[Worker] Error handling ${req.method}:`, error);
        const response: JsonRpcResponse = {
            jsonrpc: "2.0",
            error: {
                code: error.code || -32000,
                message: error.message || "Internal error",
                data: error.stack
            },
            id: req.id,
        };
        process.stdout.write(JSON.stringify(response) + "\n");
    }
}

// Only start the listener if this file is being run directly
if (require.main === module) {
    rl.on("line", (line) => {
        const trimmed = line.trim();
        if (!trimmed) return;
        try {
            const request: JsonRpcRequest = JSON.parse(trimmed);
            handleRequest(request);
        } catch (error) {
            console.error("[Worker] Failed to parse input as JSON:", line);
        }
    });

    console.error("Juki Worker Started");
}
