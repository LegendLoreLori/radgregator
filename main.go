package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/LegendLoreLori/radgregator/internal/config"
	"github.com/LegendLoreLori/radgregator/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
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
	cmds map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	f, ok := c.cmds[cmd.name]
	if !ok {
		return fmt.Errorf("command: %q not found", cmd.name)
	}
	if err := f(s, cmd); err != nil {
		return err
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional argument [USERNAME]")
	}
	if _, err := s.db.GetUser(context.Background(), cmd.args[1]); err != nil {
		return fmt.Errorf("no user %q registered", cmd.args[1])
	}

	if err := s.cfg.SetUser(cmd.args[1]); err != nil {
		return fmt.Errorf("error calling config method SetUser: %w", err)
	}
	println("username set")
	return nil
}
func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional argument [USERNAME]")
	}

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[1],
	})
	if err != nil {
		return err
	}
	fmt.Printf("user %s created\n", user.Name)
	log.Printf("user created: %+v\n", user)
	return nil
}
func handlerReset(s *state, _ command) error {
	if err := s.db.ResetUsers(context.Background()); err != nil {
		return err
	}
	println("all users deleted")
	return nil
}
func handlerList(s *state, _ command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	} else if len(users) == 0 {
		return errors.New("no users registered")
	}

	for i := 0; i < len(users); i++ {
		if users[i].Name == s.cfg.CurrentUserName {
			fmt.Printf(" * %s (current)\n", users[i].Name)
			continue
		}
		fmt.Printf(" * %s\n", users[i].Name)
	}
	return nil
}
func handlerAggregate(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional argument [INTERVAL]")
	}
	interval, err := time.ParseDuration(cmd.args[1])
	if err != nil {
		return err
	}
	if interval < 5*time.Second {
		return errors.New("input an interval greater than 5 seconds")
	}

	fmt.Printf("begin collecting feeds every %s\n", interval)
	ticker := time.NewTicker(interval)
	for ; ; <-ticker.C {
		scrapeFeeds(context.Background(), s)
	}
}
func handlerFeeds(s *state, _ command) error {
	feeds, err := s.db.GetFeedsUsers(context.Background())
	if err != nil {
		return err
	} else if len(feeds) == 0 {
		return errors.New("no feeds created")
	}

	for i := 0; i < len(feeds); i++ {
		fmt.Printf(" * %s - %q - %s\n", feeds[i].Name, feeds[i].Url, feeds[i].UserName.String)
	}

	return nil
}
func handlerKillFeed(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional arguments [URL]")
	}

	feed, err := s.db.DeleteFeed(context.Background(), cmd.args[1])
	if err != nil {
		return nil
	}

	fmt.Printf("deleted feed: %q - %s\n", feed.Name, feed.Url)

	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 3 {
		return errors.New("missing positional arguments [NAME] [URL]")
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.args[1],
		Url:       cmd.args[2],
		UserID:    user.ID,
	})
	if err != nil {
		return err
	}
	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("added feed %q %s\n", feed.Name, feed.Url)
	log.Printf("feed created: %+v\n", feed)
	return nil
}
func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional argument [URL]")
	}

	feed, err := s.db.GetFeed(context.Background(), cmd.args[1])
	if err != nil {
		return err
	}
	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return err
	}

	// I feel like i've missed the point here but oh well
	fmt.Printf("user: %s now following %q\n", feedFollow.UserName, feedFollow.FeedName)
	log.Printf("feed_follow created: %+v\n", feedFollow)
	return nil
}
func handlerFollowing(s *state, _ command, user database.User) error {
	following, err := s.db.GetFeedFollowsForUser(context.Background(), user.Name)
	if err != nil {
		return err
	}
	fmt.Printf("all feeds %s is following:\n", user.Name)
	if len(following) == 0 {
		fmt.Printf("%+v", following)
		println(" * none!")
	}
	for i := 0; i < len(following); i++ {
		fmt.Printf(" * %q\n", following[i].FeedName)
	}

	return nil
}
func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional argument [URL]")
	}
	deleted, err := s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		Url:    cmd.args[1],
	})
	if err != nil {
		return err
	}

	fmt.Printf("unfollowed: %q", cmd.args[1])
	log.Printf("feed_follow deleted: %+v", deleted)

	return nil
}
func handlerBrowse(s *state, cmd command, user database.User) error {
	var limit int
	if len(cmd.args) < 2 {
		limit = 2
	} else {
		var err error
		limit, err = strconv.Atoi(cmd.args[1])
		if err != nil {
			return err
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Printf("\n * %s\n", post.FeedName)
		fmt.Printf("%s - %s\n", post.Title.String, post.Url)
		println(post.Description.String)
		if post.PublishedAt.Valid {
			fmt.Printf("%s\n", post.PublishedAt.Time)
		}
	}

	return nil
}

func main() {
	f, err := os.OpenFile("db.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close() // unsure if this needs to be called but i'll put it here for now (maybe one day i'll find out)
	log.SetOutput(f)
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	dbUrl := cfg.DbUrl
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	dbQueries := database.New(db)

	s := state{dbQueries, &cfg}
	c := commands{make(map[string]func(*state, command) error)}
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("list", handlerList)
	c.register("agg", handlerAggregate)
	c.register("add-feed", middlewareLoggedIn(handlerAddFeed))
	c.register("kill-feed", handlerKillFeed)
	c.register("feeds", handlerFeeds)
	c.register("follow", middlewareLoggedIn(handlerFollow))
	c.register("following", middlewareLoggedIn(handlerFollowing))
	c.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	c.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 {
		fmt.Println("missing command name\n usage: radgregate COMMAND [...ARGS]")
		os.Exit(1)
	}
	cmd := command{os.Args[1], os.Args[1:]}
	if err := c.run(&s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
