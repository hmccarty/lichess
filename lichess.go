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

type Lichess struct {
	client AuthorizedClient
	profile Profile
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

/*
 * BOARD
 */

// GET
const streamEventPath = "/api/stream/event"
const streamBoardPath = "/api/board/game/stream/%s" // GameID

// POST
const seekPath = "/api/board/seek"
const boardMovePath = "/api/board/game/%s/move/%s" // GameID, Move
const sendChatPath = "/api/board/game/%s/chat" // GameID
const abortGamePath = "/api/board/game/%s/abort" // GameID
const resignGamePath = "/api/board/game/%s/resign" // GameID
const drawGamePath = "/api/board/game/%s/draw/%s" // GameID, Decision

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
	Rated bool `json:"rated"`
	Color string `json:"color"`
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

type Clock struct {
	Initial uint32 `json:"initial"`
	Increment uint32 `json:"increment"`
}

type Game struct {
	ID string `json:"id"`
	Board chan Board
}

type Board struct {
	Type string `json:"type"`

	// Game Full
	ID string `json:"id", omitempty`
	Rated bool `json:"rated, omitempty"`
	Variant Variant `json:"variant, omitempty"`
	Clock Clock `json:"clock, omitempty"`
	Speed string `json:"speed, omitempty"`
	CreatedAt uint32 `json:"createdAt, omitempty"`
	White WhiteSide `json:"white,omitempty"`
	Black BlackSide `json:"black,omitempty"`
	InitialFen string `json:"initialFen, omitempty"`
	State State `json:"state, omitempty"`

	// Game State
	Moves string `json:"moves,omitempty"`
	Status string `json:"status,omitempty"`
	Winner string `json:"winner,omitempty"`

	// Chat Line
	Username string `json:"username, omitempty"`	
	Text string `json:"text, omitempty"`
	Room string `json:"room, omitempty"`
}

type State struct {
	Type string `json:"gameState"`
	Moves string `json:"moves"`
	WhiteTime uint32 `json:"wtime"`
	BlackTime uint32 `json:"btime"`
	WhiteIncre uint32 `json:"winc"`
	BlackIncre uint32 `json:"binc"`
	Status string `json:"status"`
}

type WhiteSide struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type BlackSide struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

/*
 * CHALLENGE
 */

// GET

// POST
const challengeRespPath = "/api/challenge/%s/%s" // ChallengeID, Resp

func (l Lichess) AuthenticateClient(id string, secret string, scopes []string) {
	conf := &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Scopes:       scopes,
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

func (l Lichess) GetClient() AuthorizedClient {
	return l.client
}

func (l Lichess) GetAccount() Profile {
	fmt.Println(l.profile)	
	fmt.Println("Check 1")
	if (Profile{}) == l.profile {
		fmt.Println("Check 2")
		resp, err := l.client.Get(lichessURL + accountPath)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		fmt.Println("Check 3")

		dec := json.NewDecoder(resp.Body)
		profile := Profile{}
		err = dec.Decode(&profile)
		if err != nil {
			log.Fatal(err)
		}
		
		l.profile = profile
	}
	
	return l.profile
}

func (l Lichess) GetBoardChannel() chan Board {
	return l.currGame.Board
}

func (l Lichess) FindAndStartGame(rated bool, time uint8, incre uint8,
								  variant string, color string, ratingRange string)  {
	var wg sync.WaitGroup
	wg.Add(1)

	event := Event{}
	WatchForGame(l.client, &event, &wg)
	SeekGame(l.client, rated, time, incre, variant, color, ratingRange)

	wg.Wait()
	l.currGame = event.Game
	l.currGame.Board = make(chan Board)
}

func WatchForGame(client AuthorizedClient, event *Event, wg *sync.WaitGroup) {
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
					client.Post(lichessURL + challengeRespPath + event.Challenge.ID + "/accept", "plain/text", strings.NewReader(""))
				} else if response == "n" {
					client.Post(lichessURL + challengeRespPath + event.Challenge.ID + "/decline", "plain/text", strings.NewReader(""))
				} else {
					fmt.Println("Invalid response")
				}
		}
	}
}

func SeekGame(client AuthorizedClient, rated bool, time uint8, incre uint8,
					variant string, color string, ratingRange string) {
	
	params := fmt.Sprintf("rated=%t&time=%d&increment=%d&variant=%s&color=%s&ratingRange=%s",
							rated, time, incre, variant, color, ratingRange)
	_, err := client.Post(lichessURL + seekPath, "application/x-www-form-urlencoded", 
							strings.NewReader(params))
	if err != nil {
		log.Fatal(err)
	}
}

func (l Lichess) WatchForBoardUpdates(gameId string, ch chan<- Board, wg *sync.WaitGroup) {
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