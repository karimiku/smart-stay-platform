# Smart Stay Platform

Go、gRPC、Cloud Pub/Sub を用いた、スケーラブルなスマートステイ・プラットフォーム

マイクロサービスアーキテクチャ、BFFパターン、Cloud Run上でのイベント駆動設計を採用しています。このプロジェクトは、予約管理からIoTデバイス制御（スマートロック、空調など）までを、高性能かつ高い信頼性で処理する「スマートホテル」管理のためのクラウドネイティブなソリューションです。

## アーキテクチャ

本システムは、**マイクロサービスアーキテクチャ** と **BFF (Backend For Frontend) パターン**を採用しています。

### 通信戦略

- **同期通信 (gRPC)**: ログインやスマートロックの解錠など、即時の一貫性が求められるクリティカルなユーザー操作に使用
- **非同期通信 (Cloud Pub/Sub)**: サービスの疎結合化や、複雑なワークフロー（予約フロー、通知など）の処理に使用

### インフラストラクチャ

すべてのサービスはコンテナ化され、**Google Cloud Run** 上にデプロイされます。永続化層には **Supabase (PostgreSQL)** を使用し、サービスごとに論理的にデータベースを分割しています。

### 構成図 (High-Level Diagram)

```
Client (Web/Mobile)
    |
    | [REST/GraphQL]
    |
    v
API Gateway (BFF)
    |
    +------------------+----------------+------------------+
    |                  |                |                  |
    v                  v                v                  v
Auth Service       Room Service    Reservation Service  Key Service
(gRPC)             (gRPC)          (gRPC/PubSub)         (IoT)
    |                  |                |                  |
    +------------------+----------------+------------------+
                        |
                        v
                  Cloud Pub/Sub
                        |
                        v
              Event-Driven Workflows
```

## 技術スタック

- **言語**: Go (Golang) 1.23+
- **通信プロトコル**: gRPC (Protobuf), REST (HTTP/2)
- **メッセージング**: Google Cloud Pub/Sub
- **インフラ**: Google Cloud Run, Docker
- **データベース**: PostgreSQL (Supabase)
- **CI/CD**: Cloud Build / GitHub Actions

## マイクロサービス一覧

| サービス名 | 責務 | 通信方式 |
|-----------|------|---------|
| **api-gateway** | エントリーポイント、認証、バックエンドサービスへのファンアウト処理 | REST (In), gRPC (Out) |
| **auth-service** | ユーザーID管理、トークンの生成と検証 | gRPC |
| **room-service** | 客室の在庫管理とステータス管理 | gRPC |
| **reservation-service** | 予約管理、Sagaパターンのコーディネーター | gRPC, Pub/Sub |
| **key-service** | スマートロックの制御 (IoT連携) | gRPC, Pub/Sub |

## セットアップ

### 前提条件

- Go 1.23以上
- Protocol Buffers Compiler (protoc)
- Docker (オプション)

### 依存関係のインストール

```bash
go mod download
```

### Protocol Buffersのコンパイル

```bash
protoc --go_out=. --go-grpc_out=. protobuf/*.proto
```

### サーバーの起動

```bash
go run services/server.go
```

### クライアントの実行

```bash
go run client/client.go
```

## ライセンス

このプロジェクトはポートフォリオ用のサンプル実装です。


