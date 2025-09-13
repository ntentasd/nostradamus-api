module github.com/ntentasd/nostradamus-api

go 1.24.3

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.15.3

require github.com/gocql/gocql v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)
