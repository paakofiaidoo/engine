import { Project, SyntaxKind } from "ts-morph";
import { randomUUID } from "crypto";

const project = new Project({
    useInMemoryFileSystem: true,
    compilerOptions: {
        jsx: 4, // ReactJSX
    },
});

function astNodeToCanvasItem(node: any): any {
    const kind = node.getKind();
    if (kind === SyntaxKind.JsxText) {
        return node.getText().trim();
    }
    if (kind === SyntaxKind.JsxElement || kind === SyntaxKind.JsxSelfClosingElement) {
        const opening = kind === SyntaxKind.JsxElement ? node.getOpeningElement() : node;
        const tagName = opening.getTagNameNode().getText();
        return { name: tagName, type: 'ELEMENT' };
    }
    if (kind === SyntaxKind.ParenthesizedExpression) {
        return astNodeToCanvasItem(node.getExpression());
    }
    return null;
}

const code = `
import Image from "next/image";

export default function Home() {
  return (
    <div className="flex">
      <h1>Hello</h1>
    </div>
  );
}
`;

const sourceFile = project.createSourceFile("test.tsx", code);
const returnStatements = sourceFile.getDescendantsOfKind(SyntaxKind.ReturnStatement);
console.log("Found return statements:", returnStatements.length);

if (returnStatements.length > 0) {
    const jsx = returnStatements[0].getExpression();
    console.log("Expression kind:", jsx?.getKindName());
    const result = astNodeToCanvasItem(jsx);
    console.log("Parsed result:", JSON.stringify(result, null, 2));
}
