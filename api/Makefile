BINARY_NAME=api
POSTGRES_DSN='sslmode=disable host=localhost dbname=test_database'
ADMIN_PASSWORD_BCRYPT='$$2a$$12$$/Kav49dP2l31ssbwNeD4vOqWNafvPinjat0P8k/fQbFi3dcP6PQwS'

build:
	go build -ldflags "-X main.GitHash=$(shell git rev-parse HEAD)" -o $(BINARY_NAME)

run: build
	POSTGRES_DSN=$(POSTGRES_DSN) ADMIN_PASSWORD_BCRYPT=$(ADMIN_PASSWORD_BCRYPT) ./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
