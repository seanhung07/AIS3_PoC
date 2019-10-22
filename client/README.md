# Client

TAG need to be 27 char long, and only contain `9ABCDEFGHIJKLMNOPQRSTUVWXYZ`
Server TAG Need to be same as Client

1. Modify client.go and replace your server RSA4096 public key

```bash

go get .
go build . 


./client {{TAG}}
```
