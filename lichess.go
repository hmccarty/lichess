package lichess

import (
	"os"
	"log"
	"sync"
	"fmt"
	"bufio"
	"strings"
	"io"
	"encoding/json"
	"golang.org/x/oauth2"
)

const lichessURL = "https://lichess.org"
const accountPath = "/api/account"
const streamEventPath = "/api/stream/event"
const seekPath = "/api/board/seek"
const streamBoardPath = "/api/board/game/stream/"
const challengePath = "/api/challenge/"
const gamePath = "/api/board/game/"
const movePath = "/move/"

type UserResp struct {
}

type Event struct {
	Type string `json:"type"`
	Challenge Challenge `json:"challenge,omitempty"`
	Game Game `json:"game,omitempty"`
}

type Challenge struct {
	ID string `json:"id"`
	Status string `json:"created"`
	Challenger Challenger `json:"challenger"`
	Variant Variant `json:"variant"`
}

type Challenger struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Title string `json:"title"`
	Rating int `json:"rating"`
	Patron bool `json:"patron"`
	Online bool `json:"online"`
	Lag int `json:"lag"`
}

type Variant struct {
	Key string `json:"key"`
	Name string `json:"name"`
	Short string `json:"short"`
}

type Game struct {
	ID string `json:"id"`
	Board chan Board
}

type Board struct {
	Type string `json:"type"`
	Moves string `json:"moves,omitempty"`
	Status string `json:"status,omitempty"`
	White WhiteSide `json:"white,omitempty"`
	Black BlackSide `json:"black,omitempty"`
	Winner string `json:"winner,omitempty"`
}

type WhiteSide struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type BlackSide struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type Lichess struct {
	client AuthorizedClient
	user User
	currGame Game
}

type User struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Title string `json:"title"`
}

func (l Lichess) authenticateClient() {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("LICHESS_CLIENT_ID"),
		ClientSecret: os.Getenv("LICHESS_CLIENT_SECRET"),
		Scopes:       []string{"preference:read",
		                       "challenge:read", "challenge:write",
							   "bot:play", "board:play"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://oauth.lichess.org/oauth/authorize",
			TokenURL: "https://oauth.lichess.org/oauth",
		},
	}

	resp, err := AuthenticateUser(conf)
	if err != nil {
		log.Fatal(err)
	}

	l.client = *resp
}

func (l Lichess) getUser() User {
	if (User{}) == l.user {
		resp, err := l.client.Get(lichessURL + accountPath)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		user := User{}
		err = dec.Decode(&user)
		if err != nil {
			log.Fatal(err)
		}
		
		l.user = user
	}
	
	return l.user
}

func (l Lichess) getBoardChannel() chan Board {
	return l.currGame.Board
}

func (l Lichess) findGame()  {
	var wg sync.WaitGroup
	wg.Add(1)

	event := Event{}
	watchForGame(l.client, &event, &wg)
	seekGame(l.client)

	wg.Wait()
	l.currGame = event.Game
	l.currGame.Board = make(chan Board)
}

func watchForGame(client AuthorizedClient, event *Event, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := client.Get(lichessURL + streamEventPath)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	eventResp := Event{}
	for {
		err := dec.Decode(&event)
		if err != nil {
			log.Fatal(err)
		}

		switch event.Type {
			case "gameStart":
				*event = eventResp;
				return
			case "challenge":
				fmt.Printf("Challenge from %s\n", event.Challenge.Challenger.Name)
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Do you accept? (y or n): ")
				response, _ := reader.ReadString('\n')

				if response == "y" {
					client.Post(lichessURL + challengePath + event.Challenge.ID + "/accept", "plain/text", strings.NewReader(""))
				} else if response == "n" {
					client.Post(lichessURL + challengePath + event.Challenge.ID + "/decline", "plain/text", strings.NewReader(""))
				} else {
					fmt.Println("Invalid response")
				}
		}
	}
}

func seekGame(client AuthorizedClient) {
	_, err := client.Post(lichessURL + seekPath, "application/x-www-form-urlencoded", strings.NewReader("time=10&increment=0"))
	if err != nil {
		log.Fatal(err)
	}
}

func (l Lichess) watchForBoardUpdates(gameId string, ch chan<- Board, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := l.client.Get(lichessURL + streamBoardPath + gameId)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	for {
		boardResp := Board{}
		err := dec.Decode(&boardResp)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Fatal(err)
		}

		ch <- boardResp
	}
}