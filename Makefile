.PHONY: help up up-d down restart logs test build

# デフォルトターゲット: ヘルプを表示
help:
	@echo "利用可能なコマンド一覧:"
	@echo "  make up          - コンテナをビルドして起動 (フォアグラウンド)"
	@echo "  make up-d        - コンテナをバックグラウンドで起動"
	@echo "  make down        - コンテナを停止・削除"
	@echo "  make restart     - コンテナを再起動"
	@echo "  make logs        - ログを表示 (追跡)"
	@echo "  make test        - サーバーの単体・結合テストを実行"
	@echo "  make build       - ローカル環境で Go バイナリをビルド"

# Docker Compose 関連
up:
	docker compose up --build

up-d:
	docker compose up --build -d

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

# ローカル Go 開発・テスト関連
test:
	@cd server && go test -v ./...

build:
	@cd server && go build -o bin/server ./cmd/server
