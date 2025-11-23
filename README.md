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
├── pkg/                 # 全てのサービスが参照する共通コード
│   └── genproto/        # サービスコード (auth, key, reservation)
├── api-gateway/         # BFFの実装
└── services/            # 各マイクロサービスの実装
    ├── auth/
    ├── key/
    └── reservation/
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

## ライセンス

このプロジェクトはポートフォリオ用のサンプル実装です。
