# Gator CLI 🐊

Gator is a custom command-line interface (CLI) tool built in Go that acts as a background RSS feed aggregator. It allows you to follow your favorite RSS feeds, fetch their latest posts continuously in the background, and browse them directly from your terminal.

## Prerequisites

Before you can run Gator, you will need to have the following software installed on your machine:

1. **[Go](https://go.dev/doc/install)** (1.20 or later recommended)
2. **[PostgreSQL](https://www.postgresql.org/download/)**
3. **[Goose](https://github.com/pressly/goose)** (for running database migrations)

## Installation

You can easily install the Gator CLI to your Go binary path using `go install`. 

Open your terminal in the root of this project repository and run:

```bash
go install .
```
*Note: Make sure your `~/go/bin` directory is added to your system's `$PATH` so you can run the `gator` command from anywhere.*

## Setup

Before running the CLI, you need to configure your database and configuration file.

### 1. Database Schema
Create a PostgreSQL database for the project (e.g., `gator`). Then, run the schema migrations to create the required tables. Navigate to the `sql/schema` directory and use `goose`:

```bash
cd sql/schema
goose postgres "postgres://your_username:@localhost:5432/gator" up
```

### 2. Configuration File
Gator expects a configuration file located at `~/.gatorconfig.json` in your home directory. Create this file and add your database connection string:

```json
{
  "db_url": "postgres://your_username:@localhost:5432/gator"
}
```

## Usage & Commands

Once installed and configured, you can start interacting with Gator! Here are the primary commands you can run:

### Account Management
* **`gator register <username>`**: Creates a new user in the database and logs you in.
* **`gator login <username>`**: Logs you in as an existing user.
* **`gator users`**: Lists all users registered in the database.

### Feed Management
* **`gator addfeed <name> <url>`**: Adds a new RSS feed to the database and automatically follows it for your user.
* **`gator feeds`**: Lists all RSS feeds currently in the database.
* **`gator follow <url>`**: Follows an existing RSS feed.
* **`gator following`**: Lists all the feeds your current user is following.
* **`gator unfollow <url>`**: Unfollows a specific RSS feed.

### Reading and Scraping
* **`gator agg <time_duration>`**: Starts the long-running aggregator service. It will continuously fetch posts from all feeds in the database. (Example: `gator agg 1m` will fetch every 1 minute). Press `Ctrl+C` to stop.
* **`gator browse [limit]`**: Views the latest posts saved to the database from the feeds you follow. The limit is optional (defaults to 2). (Example: `gator browse 10`).