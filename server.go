package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Set API version
var version string = "v1"

type Player struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Sport     string `json:"sport"`
	Gender    string `json:"gender"`
	Age       int    `json:"age"`
	Country   string `json:"country"`
	ID        string `json:"id"`
}

type playerHandlers struct {
	//To handle concurrent requests, allows h.lock in GET and POST
	sync.Mutex
	store map[string]Player
}

func (h *playerHandlers) players(w http.ResponseWriter, r *http.Request) {
	// Switch based on the request method
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	default:
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Error, method not allowed!"))
		return
	}
}

func (h *playerHandlers) get(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v1/players playerAPI Players
	// Get list of all players
	// Returns 200 with list of all players
	// ---
	// produces:
	// - application/json
	// responses:
	//     '200':
	//         description: OK
	//     '500':
	//         description: SERVER_ERROR

	// Creates a new list with the length of local store, and populates it with Players
	players := make([]Player, len(h.store))
	h.Lock()
	i := 0
	for _, player := range h.store {
		players[i] = player
		i++
	}
	h.Unlock()
	jsonData, err := json.Marshal(players)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	// Generate Header and write to it
	w.Header().Add("content-type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func (h *playerHandlers) getRandomPlayer(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v1/players/random general getRandomPlayer
	// Get a random player
	// Returns 200 with a random player
	// ---
	// produces:
	// - application/json
	// responses:
	//     '200':
	//         description: OK
	//     '500':
	//         description: SERVER_ERROR
	//     '404':
	//         description: NOT_FOUND

	// Populate a string array of ids
	ids := make([]string, len(h.store))
	h.Lock()
	i := 0
	for id := range h.store {
		ids[i] = id
		i++
	}
	defer h.Unlock()
	// If we have no ids, we have no players. Not found.
	var target string
	if len(ids) == 0 {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusNotFound)
		return
		// if we have one id, we have one player. Return it.
	} else if len(ids) == 1 {
		target = ids[0]
		// else, generate a random integer between 0 and len(ids) -1 and return that index of ids
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}
	w.Header().Add("location", fmt.Sprintf("/%s/players/%s", version, target))
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusFound)
}

func (h *playerHandlers) getPlayer(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v1/players/{player_id} general getPlayer
	// Get player by its id
	// Returns 200 with player matching id.
	// ---
	// produces:
	// - application/json
	// responses:
	//     '200':
	//         description: OK
	//     '500':
	//         description: SERVER_ERROR
	//     '404':
	//        description: NOT_FOUND
	// parameters:
	// - name: player_id
	//   in: path
	//   description: Player id to retrieve
	//   required: true
	//   type: integer

	// get a player by its id

	parts := strings.Split(r.URL.String(), "/")
	// if parts (e.g. players/<id>) is smaller than 3 something has gone wrong
	if len(parts) != 4 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if parts[3] == "random" {
		h.getRandomPlayer(w, r)
		return
	}
	h.Lock()
	player, ok := h.store[parts[3]]
	h.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	jsonData, err := json.Marshal(player)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	// Generate Header and write to it
	w.Header().Add("content-type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func (h *playerHandlers) post(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v1/players general postPlayer
	// Create a Player object in local storage
	// Returns 200 on successfull creation.
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: player
	//   in: body
	//   description: The player to create.
	//   schema:
	//     type: object
	//     required:
	//       - firstname
	//       - lastname
	//       - sport
	//       - gender
	//       - age
	//       - country
	//     properties:
	//       firstname:
	//         type: string
	//       lastname:
	//         type: string
	//       sport:
	//         type: string
	//       gender:
	//         type: string
	//       age:
	//         type: integer
	//       country:
	//         type: string
	// responses:
	//     '201':
	//         description: CREATED
	//     '500':
	//         description: SERVER_ERROR
	//     '400':
	//        description: BAD_REQUEST
	//     '415':
	//        description: UNSUPPORTED_MEDIA_TYPE

	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	// if we can unmarshal the json and there is an error we *assume* there is an internal server error
	if err != nil {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")

	if ct != "application/json" {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content type application/json, got %s", ct)))
		return
	}

	var player Player
	err = json.Unmarshal(bodyBytes, &player)
	// if we can't unmarshal the json and there is an error we *assume* there is a user error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// set a unique id for each player
	player.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	h.Lock()
	h.store[player.ID] = player
	// Unlock at the very end
	defer h.Unlock()
	w.Header().Add("content-type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusCreated)

}

func newPlayerHandlers() *playerHandlers {
	return &playerHandlers{
		store: map[string]Player{},
	}
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("ADMIN_PASSWORD env password required!")
	}
	return &adminPortal{password: password}
}

func (a adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 – Unauthorized"))
		return
	}
	w.Write([]byte("<html><h1> Admin Portal </h1></html>"))
}
func main() {

	admin := newAdminPortal()
	playerHandlers := newPlayerHandlers()
	http.HandleFunc(fmt.Sprintf("/%s/players", version), playerHandlers.players)
	http.HandleFunc(fmt.Sprintf("/%s/players/", version), playerHandlers.getPlayer)
	http.HandleFunc(fmt.Sprintf("/%s/admin", version), admin.handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
