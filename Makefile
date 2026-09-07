.PHONY: api web desktop test smoke

api:
	cd push-be && go run .

web:
	npm --prefix push-fe run dev

desktop:
	cd push-fe && npm run tauri dev

test:
	@test -n "$$TEST_DATABASE_URL" || (echo 'Set TEST_DATABASE_URL to a dedicated PostgreSQL test database' && exit 1)
	cd push-be && go test -race -count=1 ./...
	npm --prefix push-fe test
	npm --prefix push-fe run check:fsd
	npm --prefix push-fe run build

smoke:
	node scripts/smoke.mjs
