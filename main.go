package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
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
		CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Name:      cmd.args[1],
	})
	if err != nil {
		return err
	}
	fmt.Printf("user %s created\n", user.Name)
	log.Printf("user created: %v\n", user)
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
	}
	if len(users) == 0 {
		println("no users registered")
		return nil
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
		return errors.New("missing positional argument [FEED]")
	}

	rssFeed, err := fetchFeed(context.Background(), cmd.args[1])
	if err != nil {
		print(err)
	}
	fmt.Printf("%+v\n", *rssFeed)
	return nil
}
func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) < 3 {
		return errors.New("missing positional arguments [NAME] [URL]")
	}

	user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Name:      cmd.args[1],
		Url:       cmd.args[2],
		UserID:    user.ID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("added feed %q %s\n", feed.Name, feed.Url)
	return nil
}

func main() {
	f, err := os.OpenFile("db.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close() // unsure if this needs to be called but i'll put it here for now
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
	c.register("add-feed", handlerAddFeed)

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
