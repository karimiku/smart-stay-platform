# Smart Stay Platform

Go、gRPC、Cloud Pub/Sub を用いた、オーナー/ゲスト向けのスマートステイ・プラットフォーム。マイクロサービスアーキテクチャ、BFF パターン、Cloud Run 上でのイベント駆動設計を採用しています。このプロジェクトは、会員制/契約ベースの宿泊体験を実現するために、認証、デジタル鍵生成、予約のライフサイクル管理に特化しています。

## 🏗️ アーキテクチャ

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

## 📦 マイクロサービス一覧

| サービス名              | 責務                                                                        | 通信方式              |
| ----------------------- | --------------------------------------------------------------------------- | --------------------- |
| **api-gateway**         | REST $\to$ gRPC 変換、認証ミドルウェア、Goroutine によるファンアウト。      | REST (In), gRPC (Out) |
| **auth-service**        | ユーザーの認証、JWT トークンの生成と検証。                                  | gRPC                  |
| **reservation-service** | 予約・契約のライフサイクル管理、Saga パターンの調整役。                     | gRPC, Pub/Sub (発行)  |
| **key-service**         | 予約情報に基づくデジタルキーの発行・無効化（外部 API への抽象化レイヤー）。 | gRPC, Pub/Sub (購読)  |

## 📁 ディレクトリ構成

```
smart-stay-platform/
├── go.mod
├── Makefile             # gRPCコード生成と環境構築を自動化
├── proto/               # gRPCの契約 (.protoファイルソース)
├── cmd/                 # 実行可能なアプリケーション
│   ├── api-gateway/     # BFFの実装
│   ├── auth-service/    # 認証サービス
│   ├── key-service/     # 鍵サービス
│   └── reservation-service/  # 予約サービス
├── internal/            # プロジェクト内部のみで使うコード
│   └── events/          # 共通イベント構造体 (EventPayload など)
└── pkg/                 # 外部から import 可能な共通コード
    └── genproto/        # 生成されたgRPCコード (auth, key, reservation)
```

## 🚀 環境構築 (Getting Started)

### ⚠️ 前提条件

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

#### コード生成

`proto/` フォルダの定義に基づき、Go コードを生成します。

```bash
make proto
```

→ これにより、`pkg/genproto` 配下に Go のインターフェースとデータ構造が生成されます。

#### ローカル開発環境の起動

Docker Compose を使用して、すべてのサービスを一度に起動します。

```bash
docker-compose up --build
```

これにより、以下のサービスが起動します：

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

#### 認証

- **POST `/login`**
  - ユーザーログイン
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
      "token": "dummy-jwt-token-example",
      "expires_in": 3600
    }
    ```

#### 予約

- **POST `/reservations`**
  - 予約を作成（Saga パターンの開始）
  - リクエスト:
    ```json
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
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
    1. Reservation Service が予約を作成（UUID で一意の ID を生成）
    2. `ReservationCreated` イベントを Pub/Sub に発行
    3. Key Service がイベントを購読し、自動的に鍵を生成

#### 鍵管理（デバッグ用）

- **POST `/keys/generate`**
  - 手動で鍵を生成
  - リクエスト:
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

## 📝 実装状況

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

### 🚧 実装中

- [ ] JWT 認証の実装（Auth Service）
- [ ] パスワードハッシュ化
- [ ] データベース統合（PostgreSQL/Supabase）

### 📋 将来実装予定

- [ ] 予約ステータスの更新フロー（PENDING → CONFIRMED）
- [ ] 決済処理の統合
- [ ] 外部スマートロック API との統合
- [ ] 認証ミドルウェアの実装（API Gateway）
- [ ] エラーハンドリングとリトライロジック
- [ ] 分散トレーシング（OpenTelemetry）
- [ ] メトリクス収集とモニタリング
