.PHONY: api web desktop test smoke

api:
	cd push-be && uvicorn app.main:app --host 0.0.0.0 --port 8080

web:
	npm --prefix push-fe run dev

desktop:
	cd push-fe && cargo tauri dev

test:
	cd push-be && python -m pytest
	npm --prefix push-fe test
	npm --prefix push-fe run lint
	npm --prefix push-fe run typecheck
	npm --prefix push-fe run build

smoke:
	node scripts/smoke.mjs
