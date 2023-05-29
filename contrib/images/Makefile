babylon-relayer:
	@docker build --tag babylonchain/babylon-relayer -f babylon-relayer/Dockerfile \
	$(shell git rev-parse --show-toplevel)

babylon-relayer-rmi:
	@docker rmi babylonchain/babylon-relayer 2>/dev/null

.PHONY: babylon-relayer babylon-relayer-rmi