.PHONY: cli
cli:
	go install github.com/gogf/gf/cmd/gf/v2@latest

.PHONY: cli.install
cli.install:
	@gf -v > /dev/null 2>&1 || make cli
