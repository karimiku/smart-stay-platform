# Smart Stay Platform

Go、gRPC、Cloud Pub/Sub を用いた、スマートヴィラ予約プラットフォーム。マイクロサービスアーキテクチャ、BFF パターン、Cloud Run 上でのイベント駆動設計を採用しています。このプロジェクトは、会員制/契約ベースの宿泊体験を実現するために、認証、デジタル鍵生成、予約のライフサイクル管理に特化しています。

## アーキテクチャ

本システムは、**マイクロサービスアーキテクチャ** と **BFF (Backend For Frontend) パターン**を採用し、特に鍵の即時発行と予約の確実な連鎖に重点を置いています。

### 通信戦略

- **同期通信 (gRPC)**: 認証・認可、鍵の即時失効など、即時性が求められる処理に使用。
- **非同期通信 (Cloud Pub/Sub)**: 予約の確定、鍵の発行など、複数のサービスにまたがる Saga パターンによる分散トランザクションの制御に使用。

### インフラストラクチャ

すべてのサービスは、独立した **Google Cloud Run** インスタンスとしてデプロイされます。

### 主要な処理フロー

- **ログイン/ダッシュボード表示**: BFF は Auth Service や Reservation Service を Goroutine で並行 呼び出しし、応答時間を短縮します。
- **予約・契約確定**: Reservation Service がイベントを Pub/Sub に発行し、Key Service がこれを購読して鍵を生成する **イベント駆動型** の連鎖を実装します。

## 🛠️ 技術スタック

| カテゴリ       | 技術要素                 | 役割                                           |
| -------------- | ------------------------ | ---------------------------------------------- |
| 言語 / FW      | Go (Golang)              | 高速なマイクロサービス実装のコア言語。         |
| 通信           | gRPC (Protobuf)          | サービス間の高速かつ型安全なバイナリ通信。     |
| メッセージング | Google Cloud Pub/Sub     | 予約フローなど、非同期な処理連鎖を実現。       |
| デプロイ       | Google Cloud Run, Docker | サービスごとの独立デプロイと自動スケーリング。 |
| データベース   | PostgreSQL (Supabase)    | サービスのデータ永続化層。                     |

## マイクロサービス一覧

| サービス名              | 責務                                                                        | 通信方式              |
| ----------------------- | --------------------------------------------------------------------------- | --------------------- |
| **api-gateway**         | REST $\to$ gRPC 変換、認証ミドルウェア、Goroutine によるファンアウト。      | REST (In), gRPC (Out) |
| **auth-service**        | ユーザーの認証、JWT トークンの生成と検証。                                  | gRPC                  |
| **reservation-service** | 予約・契約のライフサイクル管理、Saga パターンの調整役。                     | gRPC, Pub/Sub (発行)  |
| **key-service**         | 予約情報に基づくデジタルキーの発行・無効化（外部 API への抽象化レイヤー）。 | gRPC, Pub/Sub (購読)  |

## ディレクトリ構成

```
smart-stay-platform/
├── go.mod
├── go.sum
├── Makefile             # gRPCコード生成と環境構築を自動化
├── sqlc.yaml            # sqlc設定ファイル（型安全なSQLクエリ生成）
├── docker-compose.yml    # ローカル開発環境の定義
├── .gitignore           # Git除外設定（.envファイルなど）
├── .env.example         # 環境変数のテンプレート
├── proto/               # gRPCの契約 (.protoファイルソース)
│   ├── auth.proto
│   ├── key.proto
│   └── reservation.proto
├── cmd/                 # 実行可能なアプリケーション
│   ├── api-gateway/     # BFFの実装
│   │   ├── handlers/   # HTTPハンドラー
│   │   │   ├── auth.go
│   │   │   ├── key.go
│   │   │   ├── reservation.go
│   │   │   └── user.go
│   │   ├── middleware/ # ミドルウェア
│   │   │   ├── auth.go  # 認証ミドルウェア
│   │   │   └── cors.go  # CORSミドルウェア
│   │   ├── utils/      # ユーティリティ関数
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── auth-service/    # 認証サービス
│   │   ├── jwt/         # JWT生成・検証
│   │   ├── main.go
│   │   ├── service.go
│   │   └── Dockerfile
│   ├── key-service/     # 鍵サービス
│   │   ├── main.go
│   │   ├── service.go
│   │   └── Dockerfile
│   └── reservation-service/  # 予約サービス
│       ├── main.go
│       ├── service.go
│       └── Dockerfile
├── internal/            # プロジェクト内部のみで使うコード
│   ├── database/        # データベース関連
│   │   ├── migrations/  # データベースマイグレーション
│   │   │   └── 001_create_users.sql
│   │   ├── queries/     # SQLクエリ定義（sqlc用）
│   │   │   └── users.sql
│   │   ├── db.go        # データベース接続（sqlc生成）
│   │   ├── models.go    # データモデル（sqlc生成）
│   │   ├── querier.go   # クエリインターフェース（sqlc生成）
│   │   └── users.sql.go # ユーザークエリ実装（sqlc生成）
│   └── events/          # 共通イベント構造体
│       └── payload.go   # EventPayload など
└── pkg/                 # 外部から import 可能な共通コード
    └── genproto/        # 生成されたgRPCコード
        ├── auth/
        ├── key/
        └── reservation/
```

## 環境構築 (Getting Started)

### 前提条件

- Go 1.23+
- Docker & Docker Compose
- Protocol Buffers Compiler (protoc)
- gRPC Go プラグイン（protoc-gen-go, protoc-gen-go-grpc）

### 🛠️ 初期セットアップ

#### リポジトリのクローン

```bash
git clone https://github.com/yourusername/smart-stay-platform.git
cd smart-stay-platform
```

#### 環境変数の設定

機密情報は環境変数で管理します。`.env.example`をコピーして`.env`を作成し、実際の値を設定してください。

```bash
cp .env.example .env
# .envファイルを編集して、実際の値を設定
```

**重要**: `.env`ファイルは絶対にコミットしないでください（`.gitignore`で除外済み）。

**本番環境では、必ず環境変数を設定してください。デフォルト値は開発環境専用です。**

#### コード生成

**gRPC コードの生成**

`proto/` フォルダの定義に基づき、Go コードを生成します。

```bash
make proto
```

→ これにより、`pkg/genproto` 配下に Go のインターフェースとデータ構造が生成されます。

**データベースコードの生成**

`sqlc`を使用して型安全な SQL クエリコードを生成します。

```bash
make sqlc
```

→ これにより、`internal/database` 配下にデータベースモデルとクエリ関数が生成されます。

#### ローカル開発環境の起動

Docker Compose を使用して、すべてのサービスを一度に起動します。

```bash
docker-compose up --build
```

これにより、以下のサービスが起動します：

- **postgres** (ポート 5432): PostgreSQL データベース
- **pubsub-emulator** (ポート 8085): Google Cloud Pub/Sub のローカルエミュレータ
- **auth-service** (ポート 50051): 認証サービス
- **reservation-service** (ポート 50052): 予約サービス
- **key-service** (ポート 50053): 鍵サービス
- **api-gateway** (ポート 8080): API ゲートウェイ（BFF）

個別サービスの起動：

```bash
# 例: API Gateway と Auth Service のみ起動
docker-compose up api-gateway auth-service pubsub-emulator
```

## 📡 API エンドポイント

### API Gateway (BFF) - `http://localhost:8080`

> **注意**: 保護されたエンドポイントは `Authorization: Bearer <token>` ヘッダーが必要です。

#### 認証（公開エンドポイント）

- **POST `/signup`**

  - ユーザー新規登録
  - 認証: 不要
  - リクエスト:
    ```json
    {
      "email": "user@example.com",
      "password": "password123",
      "name": "John Doe"
    }
    ```
  - レスポンス:
    ```json
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "message": "User registered successfully"
    }
    ```
  - エラー:
    - `400 Bad Request`: メール形式が不正、パスワードが 8 文字未満、名前が空
    - `409 Conflict`: メールアドレスが既に登録済み

- **POST `/login`**

  - ユーザーログイン
  - 認証: 不要
  - リクエスト:
    ```json
    {
      "email": "user@example.com",
      "password": "password123"
    }
    ```
  - レスポンス:
    ```json
    {
      "message": "Login successful",
      "expires_in": 3600
    }
    ```
  - 注意: JWT トークンは`httpOnly` Cookie (`auth_token`) として設定されます

- **POST `/logout`**
  - ユーザーログアウト
  - 認証: 不要（Cookie をクリアするため）
  - レスポンス:
    ```json
    {
      "message": "Logout successful"
    }
    ```
  - 注意: `auth_token` Cookie が削除されます

#### ユーザー情報（保護エンドポイント）

- **GET `/me`**
  - 現在のユーザー情報を取得
  - 認証: 必須
  - リクエストヘッダーまたは Cookie:
    ```
    Authorization: Bearer <token>
    ```
    または
    ```
    Cookie: auth_token=<token>
    ```
  - レスポンス:
    ```json
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "role": "guest"
    }
    ```

#### 予約（保護エンドポイント）

- **POST `/reservations`**
  - 予約を作成（Saga パターンの開始）
  - 認証: 必須
  - リクエストヘッダーまたは Cookie:
    ```
    Authorization: Bearer <token>
    ```
    または
    ```
    Cookie: auth_token=<token>
    ```
  - リクエストボディ:
    ```json
    {
      "room_id": 505,
      "start_date": "2024-12-25",
      "end_date": "2024-12-27"
    }
    ```
  - レスポンス:
    ```json
    {
      "reservation_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "PENDING"
    }
    ```
  - 処理フロー:
    1. JWT トークンから user_id を取得
    2. Reservation Service が予約を作成（UUID で一意の ID を生成）
    3. `ReservationCreated` イベントを Pub/Sub に発行
    4. Key Service がイベントを購読し、自動的に鍵を生成

#### 鍵管理（保護エンドポイント）

- **POST `/keys/generate`**
  - 手動で鍵を生成（デバッグ用）
  - 認証: 必須
  - リクエストヘッダーまたは Cookie:
    ```
    Authorization: Bearer <token>
    ```
    または
    ```
    Cookie: auth_token=<token>
    ```
  - リクエストボディ:
    ```json
    {
      "reservation_id": "550e8400-e29b-41d4-a716-446655440000",
      "valid_from": "2024-12-25T00:00:00Z",
      "valid_until": "2024-12-27T23:59:59Z"
    }
    ```
  - レスポンス:
    ```json
    {
      "key_code": "1234",
      "device_id": "smart-lock-device-001"
    }
    ```

## 🔧 環境変数

### 必要な環境変数

| 変数名                | 説明                           | 必須                                              |
| --------------------- | ------------------------------ | ------------------------------------------------- |
| `POSTGRES_USER`       | PostgreSQL ユーザー名          | **必須**                                          |
| `POSTGRES_PASSWORD`   | PostgreSQL パスワード          | **必須**                                          |
| `POSTGRES_DB`         | PostgreSQL データベース名      | **必須**                                          |
| `DATABASE_URL`        | データベース接続 URL           | **必須**                                          |
| `JWT_SECRET`          | JWT トークン署名用シークレット | **必須**                                          |
| `CORS_ALLOWED_ORIGIN` | CORS 許可オリジン              | オプション（デフォルト: `http://localhost:3000`） |

### 環境変数の設定方法

1. **開発環境（ローカル）**

   ```bash
   # .envファイルを作成（.env.exampleをコピー）
   cp .env.example .env
   # 必要に応じて値を変更
   ```

2. **本番環境**
   - 環境変数を直接設定するか、シークレット管理サービスを使用
   - **必ず強力なパスワードと JWT_SECRET を設定してください**
   - JWT_SECRET の生成例: `openssl rand -base64 32`

### セキュリティ注意事項

- ⚠️ `.env`ファイルは絶対にコミットしないでください
- ⚠️ デフォルト値は開発環境専用です
- ⚠️ 本番環境では必ず環境変数を設定してください
- ⚠️ JWT_SECRET は強力なランダム文字列を使用してください

## 🔐 認証

### JWT トークンの使用方法

1. **ユーザー登録（初回のみ）**

   ```bash
   curl -X POST http://localhost:8080/signup \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"password123","name":"John Doe"}'
   ```

2. **ログイン（Cookie にトークンが保存されます）**

   ```bash
   curl -X POST http://localhost:8080/login \
     -H "Content-Type: application/json" \
     -c cookies.txt \
     -d '{"email":"user@example.com","password":"password123"}'
   ```

3. **Cookie を使用して保護されたエンドポイントにアクセス**

   ```bash
   curl -X POST http://localhost:8080/reservations \
     -H "Content-Type: application/json" \
     -b cookies.txt \
     -d '{"room_id":505,"start_date":"2024-12-25","end_date":"2024-12-27"}'
   ```

   または、Authorization ヘッダーを使用：

   ```bash
   curl -X POST http://localhost:8080/reservations \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -d '{"room_id":505,"start_date":"2024-12-25","end_date":"2024-12-27"}'
   ```

4. **ログアウト（Cookie をクリア）**
   ```bash
   curl -X POST http://localhost:8080/logout \
     -H "Content-Type: application/json" \
     -b cookies.txt
   ```

### 認証ミドルウェア

API Gateway では、すべての保護されたエンドポイントで認証ミドルウェアが動作します：

- JWT トークンを検証（Authorization ヘッダーまたは Cookie から取得）
- ユーザー情報（user_id, role）をコンテキストに設定
- 認証失敗時は 401 Unauthorized を返す
- ブラウザクライアントと API クライアントの両方をサポート

## 🔄 イベント駆動フロー

### 予約作成から鍵生成までの流れ

```
1. クライアント → API Gateway (POST /reservations)
   ↓
2. API Gateway → Reservation Service (gRPC: CreateReservation)
   ↓
3. Reservation Service:
   - UUID で予約 ID を生成
   - Pub/Sub に ReservationCreated イベントを発行
     {
       "event_type": "ReservationCreated",
       "reservation_id": "550e8400-...",
       "user_id": "550e8400-e29b-41d4-a716-446655440000",
       "start_date": "2024-12-25T00:00:00Z",
       "end_date": "2024-12-27T23:59:59Z"
     }
   ↓
4. Key Service (Pub/Sub 購読):
   - ReservationCreated イベントを受信
   - 予約の開始日・終了日を使用して鍵を生成
   - 4桁の PIN コードを生成
   ↓
5. クライアントに PENDING ステータスで即座に応答
```

## 🔧 開発コマンド

### Makefile コマンド

```bash
# gRPC コード生成
make proto

# データベースコード生成（sqlc）
make sqlc

# 生成されたコードをクリーンアップ
make clean

# 必要なツールをインストール
make install-tools

# ヘルプを表示
make help
```

### Docker Compose コマンド

```bash
# すべてのサービスを起動（ビルド込み）
docker-compose up --build

# バックグラウンドで起動
docker-compose up -d

# 特定のサービスのみ起動
docker-compose up api-gateway auth-service

# ログを確認
docker-compose logs -f api-gateway

# サービスを停止
docker-compose down
```

## 実装状況

### ✅ 実装済み

- [x] マイクロサービスアーキテクチャの基本構造
- [x] gRPC サービス定義とコード生成
- [x] API Gateway (BFF) による REST → gRPC 変換
- [x] Docker Compose によるローカル開発環境
- [x] Pub/Sub エミュレータの統合
- [x] 予約作成時の UUID 生成
- [x] イベント駆動型の鍵生成フロー
- [x] 予約日付情報を含むイベントペイロード
- [x] Graceful Shutdown の実装
- [x] 認証ミドルウェアの実装（API Gateway）
- [x] JWT トークン検証とユーザーコンテキスト管理
- [x] ハンドラーとミドルウェアの分離（コード整理）
- [x] JWT トークン生成の実装（Auth Service）
- [x] パスワードハッシュ化（bcrypt）
- [x] データベース統合（PostgreSQL + sqlc）
- [x] ユーザー登録機能（POST /signup）
- [x] ログアウト機能（POST /logout）
- [x] Cookie ベースの認証（httpOnly cookies）
- [x] CORS 対応（フロントエンド連携）
- [x] パスワード強度バリデーション（8 文字以上、大文字・小文字・数字・記号）
- [x] 機密情報の環境変数化
- [x] データベースマイグレーション（users テーブル）

### 実装中

（現在進行中の作業はありません）

### 📋 将来実装予定

- [ ] 予約ステータスの更新フロー（PENDING → CONFIRMED）
- [ ] 予約一覧取得（GET /reservations）
- [ ] 予約詳細取得（GET /reservations/:id）
- [ ] 決済処理の統合
- [ ] 外部スマートロック API との統合
- [ ] エラーハンドリングとリトライロジック
- [ ] 分散トレーシング（OpenTelemetry）
- [ ] メトリクス収集とモニタリング
