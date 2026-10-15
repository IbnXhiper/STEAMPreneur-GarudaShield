.RECIPEPREFIX = >
.PHONY: build-engine run-engine build-web run-web dev test clean

build-engine:
> go build -o bin/gs ./cmd/garudashield

run-engine: build-engine
> ./bin/gs

build-web:
> cd dashboard && npm install && npm run build

run-web:
> cd dashboard && npm run dev

# jalankan engine (background) + web dev di foreground
dev: build-engine
> ./bin/gs & cd dashboard && npm install && npm run dev

test:
> go test ./internal/... ./cmd/...

clean:
> rm -f bin/gs
