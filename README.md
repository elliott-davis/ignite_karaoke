# Ignite Karaoke

This is a Go application for a fun presentation game called Ignite Karaoke.

## Demo

![](ignite.gif)

## Setup

1.  **Environment Variables:**
    This application requires API keys for Google and Giphy. These should be stored in an `.envrc` file at the root of the project. The `.gitignore` file is configured to prevent this file from being checked into version control.

    Create a `.envrc` file and add the following, replacing the placeholder text with your actual API keys:

    ```bash
    export GOOGLE_API_KEY="your_google_api_key"
    export GIPHY_API_KEY="your_giphy_api_key"
    ```

    You will need to use a tool like `direnv` to automatically load these environment variables when you are in the project directory. If you don't use `direnv`, you'll need to source the file (`source .envrc`) before running the application.

2.  **Install Dependencies:**
    This project uses Go modules to manage dependencies. Run the following command to ensure all dependencies are downloaded:

    ```bash
    go mod tidy
    ```

## Running the Application

To run the application, execute the following command from the root of the project:

```bash
go run main.go
```

The server will start on port `8080`.

To run the application for development with hot-reloading, use the `dev` target in the Makefile:

```bash
make dev
```
This requires the `air` tool to be installed (`go install github.com/air-verse/air@latest`).

## How to Play

1.  **Admin Page:**
    Navigate to `http://localhost:8080/admin`. Here you can enter the names of all the participants, one per line, into the text area and submit them.

2.  **Index Page:**
    Navigate to `http://localhost:8080/`. This page will show the list of all participants who have been added and will indicate who is next up.

3.  **Game Page:**
    To start the game for a participant, you'll need to manually construct the URL for now. For example, if the next participant is "Alice", you would navigate to `http://localhost:8080/game/Alice`.

    Once on the game page, the 1-minute timer will start automatically. The slides will advance every 15 seconds. Enjoy the show!

## Deployment

This project uses `ko` to build and publish a minimal container image without a Dockerfile.

1.  **Install `ko`:**
    If you don't have it, install `ko`:
    ```bash
    go install github.com/google/ko@latest
    ```

2.  **Set Your Repository:**
    `ko` publishes images to a container registry. You need to specify your registry by setting the `KO_DOCKER_REPO` environment variable. For example:
    ```bash
    export KO_DOCKER_REPO="gcr.io/your-gcp-project"
    ```
    You also need to be authenticated with your chosen registry (e.g., `gcloud auth configure-docker`, `docker login`).

3.  **Update the Makefile:**
    Open the `Makefile` and change the `your-repo-name` placeholder in the `release` target to your actual `KO_DOCKER_REPO` value.

4.  **Build and Publish:**
    Run the release target:
    ```bash
    make release
    ```
    `ko` will build the application, push the container image to your repository, and print the resulting image digest. You can then use this image in your deployment environment (e.g., Kubernetes). 