SQLC_VERSION ?= v1.31.1

.PHONY: sqlc-generate tunnel-up tunnel-url tunnel-down
sqlc-generate:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

tunnel-up:
	@grep -Eq '^NGROK_AUTHTOKEN=.+$$' .env || { echo 'Set NGROK_AUTHTOKEN in .env first.'; exit 1; }
	@grep -Eq '^NGROK_DOMAIN=.+$$' .env || { echo 'Set NGROK_DOMAIN to your assigned ngrok dev domain in .env first.'; exit 1; }
	docker compose --profile tunnel up -d ngrok

tunnel-url:
	@docker compose --profile tunnel ps --services --filter status=running ngrok | grep -q '^ngrok$$' || { echo 'ngrok is not running. Start it with make tunnel-up.'; exit 1; }
	@python3 -c 'text=open(".env").read(); domain=next((line.split("=",1)[1].strip().strip("\"") for line in text.splitlines() if line.startswith("NGROK_DOMAIN=")), ""); print(domain if domain.startswith("https://") else "https://" + domain)'

tunnel-down:
	docker compose --profile tunnel stop ngrok
