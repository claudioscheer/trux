const vscode = require("vscode");
const { LanguageClient, TransportKind } = require("vscode-languageclient/node");

let client;

function activate(context) {
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
