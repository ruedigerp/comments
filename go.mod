module comment-system

go 1.25

require (
	github.com/gorilla/mux v1.8.0
	github.com/redis/go-redis/v9 v9.17.2
	github.com/rs/cors v1.11.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

replace github.com/go-redis/redis/v9 => github.com/redis/go-redis/v9 v9.17.2
