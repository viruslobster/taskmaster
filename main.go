package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Player struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Img    string `json:"img"`
	Hidden bool   `json:"hidden"`

	// Task specific stuff
	Onces float64 `json:"onces"`
}

const (
	GameViewLeaderBoard = "leaderboard"
	GameViewTimer       = "timer"
)

type Game struct {
	Players []*Player `json:"players"`
	Msgs    []string  `json:"msgs"`
}

func (g *Game) nextPlayerId() int {
	var max int
	for _, player := range g.Players {
		if player.Id > max {
			max = player.Id
		}
	}
	return max + 1
}

var mu sync.Mutex

func getGame() *Game {
	data, err := ioutil.ReadFile("game.json")
	if err != nil {
		panic(err)
	}

	var game Game
	err = json.Unmarshal(data, &game)
	if err != nil {
		panic(err)
	}
	return &game
}

func putGame(game *Game) {
	f, err := os.OpenFile("game.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data, err := json.MarshalIndent(game, "", "\t")
	if err != nil {
		panic(err)
	}
	f.Write(data)
}

// TODO: http server
// getGame
// awardPoints
func getGameHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	j, _ := json.Marshal(game)
	w.Write(j)
	game.Msgs = game.Msgs[:0]
	putGame(game)
}

func updateScores(update map[int]int) {
	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	players := game.Players

	for id, delta := range update {
		for _, player := range players {
			if player.Id == id {
				player.Score += delta
				break
			}
		}
	}
	putGame(game)
}

func updateOnces(update map[int]float64) {
	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	players := game.Players

	for id, oz := range update {
		for _, player := range players {
			if player.Id == id {
				player.Onces = oz
				break
			}
		}
	}
	putGame(game)
}

func scoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST requests allowed", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	update := make(map[int]int)
	for key, values := range r.Form {
		if len(values) != 1 {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		valueStr := values[0]
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		idStr := strings.TrimPrefix(key, "player_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		update[id] = value
	}
	updateScores(update)

	http.Redirect(w, r, "/taskmaster.html", http.StatusSeeOther)
}

func sendMsgHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST requests allowed", http.StatusBadRequest)
		return
	}
	msg, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	game.Msgs = append(game.Msgs, string(msg))
	putGame(game)
}

func hidePlayerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST requests allowed", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	idStr, ok := r.Form["player"]
	if !ok || len(idStr) != 1 {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr[0])
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	for _, player := range game.Players {
		if player.Id == id {
			player.Hidden = true
			putGame(game)
			return
		}
	}
	http.Error(w, "no such player", http.StatusBadRequest)
}

func showPlayerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST requests allowed", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	idStr, ok := r.Form["player"]
	if !ok || len(idStr) != 1 {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr[0])
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	for _, player := range game.Players {
		if player.Id == id {
			player.Hidden = false
			putGame(game)
			return
		}
	}
	http.Error(w, "no such player", http.StatusBadRequest)
}

func addPlayerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST requests allowed", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	names, ok := r.Form["name"]
	if !ok || len(names) != 1 {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	name := names[0]

	mu.Lock()
	defer mu.Unlock()

	game := getGame()
	game.Players = append(game.Players, &Player{
		Id:   game.nextPlayerId(),
		Name: name,
		Img:  "/img/derp_framed.png",
	})
	putGame(game)

	http.Error(w, "no such player", http.StatusBadRequest)
}

func setOncesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST requests allowed", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	update := make(map[int]float64)
	for key, values := range r.Form {
		if len(values) != 1 {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		valueStr := values[0]
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		idStr := strings.TrimPrefix(key, "player_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		update[id] = value
	}
	updateOnces(update)

	http.Redirect(w, r, "/disttask.html", http.StatusSeeOther)
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./www")))
	http.HandleFunc("/api/getgame", getGameHandler)
	http.HandleFunc("/api/score", scoreHandler)
	http.HandleFunc("/api/setonces", setOncesHandler)
	http.HandleFunc("/api/sendmsg", sendMsgHandler)
	http.HandleFunc("/api/hideplayer", hidePlayerHandler)
	http.HandleFunc("/api/showplayer", showPlayerHandler)
	http.HandleFunc("/api/addplayer", addPlayerHandler)
	http.ListenAndServe(":8080", nil)
}
