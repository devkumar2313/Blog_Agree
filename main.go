package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"blog_agree/internal/config"
	"blog_agree/internal/database"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"strconv"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	cont map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cont[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.cont[cmd.name]
	if !ok {
		return fmt.Errorf("command not found: %s", cmd.name)
	}
	return handler(s, cmd)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("a username is required")
	}

	username := cmd.args[0] // Declare this BEFORE using it

	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		fmt.Printf("Error: user %s does not exist\n", username)
		os.Exit(1)
	}

	err = s.cfg.SetUser(username) // Use '=' because err was already declared above
	if err != nil {
		return fmt.Errorf("could not set user: %w", err)
	}

	fmt.Printf("The user has been set to: %s\n", username)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: register <username>")
	}
	name := cmd.args[0]

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
	})
	if err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		os.Exit(1)
	}

	err = s.cfg.SetUser(name)
	if err != nil {
		return fmt.Errorf("could not set current user: %v", err)
	}

	fmt.Println("User created successfully!")
	fmt.Printf("User Details: %+v\n", user)

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		fmt.Printf("Error resetting database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database reset successfully!")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Printf("Error retrieving users: %v\n", err)
		os.Exit(1)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("- %s (current)\n", user.Name)
		} else {
			fmt.Printf("- %s\n", user.Name)
		}
	}

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	// No arguments expected
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: feeds")
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("could not get feeds: %v", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	for _, feed := range feeds {
		fmt.Printf("* %s\n", feed.Name)
		fmt.Printf("  URL: %s\n", feed.Url)
		fmt.Printf("  Added by: %s\n", feed.UserName)
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	log.Printf("Collecting feeds every %s", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	
	// The for loop ensures we trigger immediately upon startup, 
	// then blocks and waits for the next tick.
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddFeed(s *state, cmd command,user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}
	name := cmd.args[0]
	url := cmd.args[1]


	// 2. Create the feed
	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("could not create feed: %v", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not create feed follow: %v", err)
	}

	fmt.Println("Feed created successfully:")
	fmt.Printf("* ID:            %s\n", feed.ID)
	fmt.Printf("* Created At:    %v\n", feed.CreatedAt)
	fmt.Printf("* Updated At:    %v\n", feed.UpdatedAt)
	fmt.Printf("* Name:          %s\n", feed.Name)
	fmt.Printf("* URL:           %s\n", feed.Url)
	fmt.Printf("* User ID:       %s\n", feed.UserID)

	return nil
}

// Add handlerFollow
func handlerFollow(s *state, cmd command,user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: follow <url>")
	}
	url := cmd.args[0]

	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("could not find feed by URL: %v", err)
	}

	ff, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not create feed follow: %v", err)
	}

	fmt.Printf("Feed Follow created:\n")
	fmt.Printf("* User: %s\n", ff.UserName)
	fmt.Printf("* Feed: %s\n", ff.FeedName)

	return nil
}

func handlerFollowing(s *state, cmd command,user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: following")
	}


	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("could not get feed follows: %v", err)
	}

	if len(follows) == 0 {
		fmt.Println("You are not following any feeds.")
		return nil
	}

	fmt.Printf("Feeds %s is following:\n", user.Name)
	for _, follow := range follows {
		fmt.Printf("* %s\n", follow.FeedName)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		// Get the current user from the database
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("could not get current user (are you logged in?): %v", err)
		}

		// Call the provided handler, passing in the user we just found
		return handler(s, cmd, user)
	}
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}
	url := cmd.args[0]

	// First, find the feed by its URL to get its ID
	feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("could not find feed by URL: %v", err)
	}

	// Delete the follow record using the combination of User ID and Feed ID
	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not unfollow feed: %v", err)
	}

	fmt.Printf("%s unfollowed %s successfully!\n", user.Name, feed.Name)
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2 // Default limit
	if len(cmd.args) == 1 {
		if providedLimit, err := strconv.Atoi(cmd.args[0]); err == nil {
			limit = providedLimit
		} else {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return fmt.Errorf("couldn't get posts for user: %v", err)
	}

	fmt.Printf("Found %d posts for user %s:\n", len(posts), user.Name)
	for _, post := range posts {
		fmt.Printf("%s from %s\n", post.PublishedAt.Time.Format("Mon Jan 2"), post.FeedName)
		fmt.Printf("--- %s ---\n", post.Title)
		if post.Description.Valid {
			fmt.Printf("    %s\n", post.Description.String)
		}
		fmt.Printf("Link: %s\n", post.Url)
		fmt.Println("=====================================")
	}

	return nil
}



func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	// 1. Establish database connection FIRST
	db, err := sql.Open("postgres", cfg.DbURL) // Changed to DbURL
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	// 2. Build the state struct with both cfg and db
	appState := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	// 3. Register commands
	cmds := &commands{
		cont: make(map[string]func(*state, command) error),
	}
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)

	cmds.register("feeds", handlerFeeds)
	

	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

	// 4. Parse args and run
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Error: not enough arguments provided")
		os.Exit(1)
	}

	cmdName := args[1]
	cmdArgs := args[2:]

	cmd := command{
		name: cmdName,
		args: cmdArgs,
	}

	err = cmds.run(appState, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}