VERSION=$(grep '^VERSION=' .version | cut -d '=' -f 2-)
echo VERSION=$VERSION
COMMIT=$(git rev-parse HEAD)
echo COMMIT=$COMMIT
go build \
  -ldflags="-s -w \
    -X 'gitlab.com/TitanInd/proxy/proxy-router-v3/internal/config.BuildVersion=$VERSION' \
    -X 'gitlab.com/TitanInd/proxy/proxy-router-v3/internal/config.Commit=$COMMIT' \
  " \
  -o bin/hashrouter cmd/main.go 
