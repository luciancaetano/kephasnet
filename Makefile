.PHONY: test

test:
	@echo "==> Executando todos os testes..."
	go test ./tests/... -v
