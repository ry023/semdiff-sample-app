# semdiff sample Todo app

`ry023/semdiff` の動作確認用に作った、Go API + React Router/React/Vite フロントエンドのモノリポサンプルです。

## 構成

- `api/`: Go の REST API
  - SQLite を利用
  - ユーザー登録、ログイン、ログアウト、現在ユーザー取得
  - Todo の一覧、作成、更新、削除
- `frontend/`: React + TypeScript + Vite + React Router + MUI
- `docker-compose.yml`: API とフロントエンドをまとめて起動

## 起動

Docker Desktop が起動している状態で、リポジトリのルートから実行します。

```bash
docker compose up --build
```

ブラウザで http://localhost:3000 を開いてください。SQLite のデータは Docker volume `todo_data` に保存されます。

API をローカルで直接確認したい場合は http://localhost:8080/api/health を利用できます。

## ローカル開発

API:

```bash
cd api
go run ./cmd/server
```

フロントエンド:

```bash
cd frontend
npm install
npm run dev
```

Vite の開発サーバーは `/api` を `http://localhost:8080` にプロキシします。

## semdiff でレビューする例

```bash
semdiff review --base main --head HEAD
```

小さな変更を加えてレビュー対象を作り、API とフロントエンドの両方にまたがる差分を確認できます。

## 注意

このアプリは semdiff の検証用MVPです。パスワードハッシュは意図的に単純化しており、本番環境で利用できるセキュリティ対策ではありません。実運用では Argon2id/bcrypt、CSRF対策、レート制限、入力検証の強化、HTTPS、適切な秘密情報管理などが必要です。

