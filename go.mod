module github.com/midbel/mule

go 1.21.0

require (
	github.com/midbel/enjoy v0.0.0
	go.etcd.io/bbolt v1.3.7
)

require golang.org/x/sys v0.4.0 // indirect

replace github.com/midbel/enjoy => ../enjoy
