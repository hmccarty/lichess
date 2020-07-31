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

const streamEventPath = "/api/stream/event"
const seekPath = "/api/board/seek"
const streamBoardPath = "/api/board/game/stream/"
const challengePath = "/api/challenge/"
const gamePath = "/api/board/game/"
const movePath = "/move/"

type Lichess struct {
	client AuthorizedClient
	user User
	currGame Game
}

/*
 * ACCOUNTS
 */

const accountPath = "/api/account"
const emailPath = "/api/account/email"
const prefPath = "/api/account/preferences"
const kidModePath = "api/account/kid"

type Profile struct {
	ID string `json:"id"`
	Username string `json:"username"`
	Title string `json:"title"`
	Online bool `json:"online"`
	Playing bool `json:"playing"`
	Streaming bool `json:"streaming"`
	CreatedAt uint64 `json:"createdAt"`
	SeenAt uint64 `json:"seenAt"`
	Details Details `json:"profile"`
	NbFollowers uint32 `json:"nbFollowers"`
	NbFollowing uint32 `json:"nbFollowing"`
	CompletionRate uint8 `json:"completionRate"`
	Language string `json:"language"`
	Count Count `json:"count"`
	Performance Performance `json:"perfs"`
	Patron bool `json:"patron"`
	Disabled bool `json:"disabled"`
	Engine bool `json:"engine"`
	Booster bool `json:"booster"`
	PlayTime PlayTime `json:"playTime"`
}

type Details struct {
	Bio string `json:"bio"`
	Country string `json:country"`
	FirstName string `json:firstName"`
	LastName string `json:lastName"`
	Links string `json:links"`
	Location string `json:location"`
}

type Count struct {
	AI uint32 `json:"ai"`
	All uint32 `json:"all"`
	Bookmark uint32 `json:"bookmark"`
	Draw uint32 `json:"draw"`
	DrawH uint32 `json:"drawH"`
	Import uint32 `json:"import"`
	Loss uint32 `json:"loss"`
	LossH uint32 `json:"lossH"`
	me uint32 `json:"me"`
	Playing uint32 `json:"playing"`
	Rated uint32 `json:"rated"`
	Win uint32 `json:"win"`
	WinH uint32 `json:"winH"`
}

type Performance struct {
	Blitz PerfType `json:"blitz"`
	Bullet PerfType `json:"bullet"`
	Chess960 PerfType `json:"chess960"`
	Puzzle PerfType `json:"puzzle"`
}

type PerfType struct {
	Games uint32 `json:"games"`
	Progress int16 `json:"prog"`
	Rating uint16 `json:"rating"`
	Rd uint16 `json:"rd"`
}

type PlayTime struct {
	Total uint64 `json:"total"`
	Tv uint64 `json:"tv"`
}

type Preferences struct {
	DarkMode bool `json:"dark"`
	TranspMode bool `json:"transp"`
	BgImg string `json:"bgImg"`
	Is3D bool `json:"is3d"`
	Theme string `json:"theme"`
	PieceSet string `json:"go"`
	Theme3D string `json:"theme3d"`
	PieceSet3D string `json:"pieceSet3d"`
	SoundSet string `json:"soundSet"`
	BlindFold uint8 `json:"blindFold"`
	AutoQueen uint8 `json:"autoQueen"`
	AutoThreeFold uint8 `json:"autoThreefold"`
	Takeback uint8 `json:"takeback"`
	ClockTenths uint8 `json:"clockTenths"`
	ClockBar bool `json:"clockBar"`
	Premove bool `json:"premove"`
	Animation uint8 `json:"animation"` 	
	Captured bool `json:"captured"`	
	Follow bool `json:"follow"`	
	Highlight bool `json:"highlight"`	
	Destination bool `json:"destination"`	
	Coords uint8 `json:"coords"`	
	Replay uint8 `json:"replay"`	
	Challenge uint8 `json:"challenge"`	
	Message uint8 `json:"message"`	
	CoordColor uint8 `json:"coordColor"`	
	SubmitMove uint8 `json:"submitMove"`	
	ConfirmResign uint8`json:"confirmResign"` 	
	InsightShare uint8 `json:"insightShare"`	
	KeyboardMove uint8 `json:"keyboardMove"`	
	Zen uint8 `json:"zen"`	
	MoveEvent uint8 `json:"moveEvent"`	
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