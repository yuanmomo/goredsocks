#GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags "-s -w" -o goredsocks
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags "-s -w" -o goredsocks
upx --best goredsocks
scp goredsocks root@10.0.0.1:/usr/bin



