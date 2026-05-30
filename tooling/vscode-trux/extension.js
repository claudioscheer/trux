const vscode = require("vscode");

let client;

function activate(context) {
  let languageClient;
  try {
    languageClient = require("vscode-languageclient/node");
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    void vscode.window.showErrorMessage(
      `Could not load Trux language server support: ${message}. Run npm install in tooling/vscode-trux or install a packaged extension build.`,
    );
    return;
  }

  const { LanguageClient, TransportKind } = languageClient;
  const command =
    vscode.workspace.getConfiguration("trux").get("languageServer.path") ||
    "trux-lsp";

  client = new LanguageClient(
    "truxLanguageServer",
    "Trux Language Server",
    {
      command,
      args: [],
      transport: TransportKind.stdio,
    },
    {
      documentSelector: [{ scheme: "file", language: "trux" }],
      synchronize: {
        configurationSection: "trux",
      },
    },
  );

  context.subscriptions.push({
    dispose: () => {
      if (client) {
        void client.stop();
      }
    },
  });

  void client.start().catch((error) => {
    const message = error instanceof Error ? error.message : String(error);
    void vscode.window.showErrorMessage(
      `Could not start Trux language server "${command}": ${message}`,
    );
  });
}

function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

module.exports = {
  activate,
  deactivate,
};
