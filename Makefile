.PHONY: mock-backends-run ingress-run ingress-test-run ingress-test-down

mock-backends-run:
	go run cmd/mock_backends/main.go

ingress-run:
	go run cmd/ingress/main.go

ingress-test-run:
	docker-compose up --build 

ingress-test-down:
	docker-compose down