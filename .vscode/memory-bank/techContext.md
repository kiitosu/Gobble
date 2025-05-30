# Tech Context

## 使用技術・フレームワーク
- Go 1.x（バックエンド）
  - connect-go, gRPC, Protocol Buffers
- React 19.x（フロントエンド）
  - TypeScript, Vite, ESLint, connect-web
- Protocol Buffers（APIスキーマ管理）
- buf（コード生成・API管理）
- Docker, Docker Compose（開発・実行環境統一）

## 開発環境・セットアップ
- Docker/Docker Composeでバックエンド・フロントエンド・APIスキーマ管理を一括起動
- protoディレクトリでAPI定義→buf/protocで各言語用コード自動生成
- frontend/はViteベースのReact+TypeScript構成
- backend/はGo modules構成

## 技術的制約・注意点
- gRPC+ConnectプロトコルはHTTP/2ベース（ネットワーク設定に注意）
- APIスキーマ変更時はbuf/protocによる再生成が必要
- Docker前提のため、ローカル環境にDockerが必要
- フロントエンド・バックエンドのポート競合に注意

## 主要依存関係（抜粋）
- Go: connect-go, google.golang.org/protobuf, buf
- Frontend: react, react-dom, vite, @connectrpc/connect-web, typescript, eslint
- DevOps: docker, docker-compose

## その他
- ESLintによる静的解析・コード品質担保
- Viteによる高速なフロントエンド開発体験
