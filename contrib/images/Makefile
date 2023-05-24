babylon-relayer:
	@docker build --tag babylonchain/babylon-relayer -f babylon-relayer/Dockerfile \
	$(shell git rev-parse --show-toplevel)

babylon-relayer-rmi:
	@docker rmi babylonchain/babylon-relayer 2>/dev/null

ibcsim-gaia:
	@docker build --tag babylonchain/ibcsim-gaia -f ibcsim-gaia/Dockerfile \
		$(shell git rev-parse --show-toplevel)

ibcsim-gaia-rmi:
	@docker rmi babylonchain/ibcsim-gaia 2>/dev/null


.PHONY: babylon-relayer babylon-relayer-rmi ibcsim-gaia ibcsim-gaia-rmi