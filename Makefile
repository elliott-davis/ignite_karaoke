# Makefile for Ignite Karaoke

.PHONY: dev

# Run the application in development mode with auto-reloading using air
dev:
	@echo "Starting dev server with hot-reload..."
	@go run github.com/air-verse/air

release:
	@echo "Building and publishing container with ko..."
	@KO_DOCKER_REPO=your-repo-name ko publish -B .

.PHONY: release 