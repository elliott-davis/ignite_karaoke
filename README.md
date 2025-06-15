# Ignite Karaoke

This is a Go application for a fun presentation game called Ignite Karaoke.

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

## How to Play

1.  **Admin Page:**
    Navigate to `http://localhost:8080/admin`. Here you can enter the names of all the participants, one per line, into the text area and submit them.

2.  **Index Page:**
    Navigate to `http://localhost:8080/`. This page will show the list of all participants who have been added and will indicate who is next up.

3.  **Game Page:**
    To start the game for a participant, you'll need to manually construct the URL for now. For example, if the next participant is "Alice", you would navigate to `http://localhost:8080/game/Alice`.

    Once on the game page, the 1-minute timer will start automatically. The slides will advance every 15 seconds. Enjoy the show! 