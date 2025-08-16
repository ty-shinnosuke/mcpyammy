# mcpyammy

各生成AI CLIツールのMCP (Model Context Protocol) サーバー設定をyamlファイルで管理するためのCLIツールです。

## 機能

- **設定インポート**: 既存のMCP JSON設定をYAML形式にインポート
- **設定適用**: YAMLファイルからMCP設定をJSON形式に変換・適用

## インストール

```bash
brew install takayanagishinnosuke/tap/mcpyammy
```

もしくは[リリースページ](https://github.com/ty-shinnosuke/mcpyammy/releases)からダウンロード

## 使い方

```bash
mcpyammy
```
1. 初回起動時にyamlファイルが生成されます。
2. `import`を選択するとpath先の各クライアント設定ファイルから既存のMCP設定を取り込みます。
3. mcpを追加する場合は、yamlに記述して`apply`を実行します。

## YAML設定ファイル形式

```yaml:servers.yaml
clients:
  amazonq:
    path: .aws/amazonq/mcp.json
    servers:
    - name: awslabs.aws-documentation-mcp-server
      command: uvx
      args:
      - awslabs.aws-documentation-mcp-server@latest
      env:
        FASTMCP_LOG_LEVEL: ERROR
    - name: fetch
      command: uvx
      args:
      - mcp-server-fetch
  gemini:
    path: .gemini/settings.json
    servers:
    - name: playwright
      command: npx
      args:
      - "@playwright/mcp@latest"
  claude:
    path: .claude.json
    servers:
    - name: aws-knowledge-mcp-server
      command: uvx
      args:
      - mcp-proxy
      - --transport
      - streamablehttp
      - https://knowledge-mcp.global.api.aws
```

> [!WARNING]
> APIキーやトークンなどを記載している場合は外部公開しないように注意してください

### ライセンス
MIT
