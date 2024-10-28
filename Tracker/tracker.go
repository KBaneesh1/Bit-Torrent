package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sort"
	"time"

	// "errors"
	"encoding/json"
	"sync"
)


type updatePeerStats struct {
	Peer Peer  `json:"peer"`
	Files []string `json:"files"`
}
type Peer struct {
	IP              int     `json:"ip"`
	DownloadedBytes int64   `json:"downloaded_bytes"`
	UploadedBytes   int64   `json:"uploaded_bytes"`
	DownloadingRate float64 `json:"downloading_rate"`
	UploadingRate   float64 `json:"uploading_rate"`
}
var (
	allPeers      = make(map[int]*Peer)
	peerLastUpdate = make(map[int]time.Time) // Stores last update timestamp for each peer
	fileOwners    = make(map[string][]*Peer)
)
var (
    peersMu       sync.RWMutex
    updatesMu     sync.RWMutex
    fileOwnersMu  sync.RWMutex
)
func parseRegisterRequest(r *http.Request) (*Peer, error) {
	var data Peer
	// Read the body data
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading request body")
	}

	// Unmarshal JSON data
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("Error parsing JSON")
	}

	return &data, nil
}

func registerPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse the registration request body
	data, err := parseRegisterRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	peerIP := data.IP
	// Log the parsed data for verification(
	fmt.Printf("Received registration:\nID: %d\n:", peerIP)
	if _,exists := allPeers[data.IP];exists {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		resp_msg := fmt.Sprintf("Peer %d Already Registered \n",peerIP)
		response := map[string]string{"status":resp_msg}
		json.NewEncoder(w).Encode(response)
		return
	}
	peersMu.Lock()
	defer  peersMu.Unlock()

	allPeers[peerIP] = data

	updatesMu.Lock()
	defer  updatesMu.Unlock()
	peerLastUpdate[peerIP] = time.Now()
	
	// Send a response back to the client
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp_msg := fmt.Sprintf("Registration received of peer  with IP %d\n",data.IP)
	response := map[string]string{"status":resp_msg}
	json.NewEncoder(w).Encode(response)
	return
}


func updatePeerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var data updatePeerStats
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	peersMu.RLock()
	peer, exists := allPeers[data.Peer.IP]
	peersMu.RUnlock()
	if !exists {
		http.Error(w, "Peer not registered", http.StatusNotFound)
		return
	}
	updatesMu.Lock()
	peerLastUpdate[data.Peer.IP] = time.Now()
	updatesMu.Unlock()
	// Update peer stats
	peersMu.Lock()
	allPeers[peer.IP] = &data.Peer
	peersMu.Unlock()
	// Update file ownership
	fileOwnersMu.Lock()
	for _, file := range data.Files {
		if _, exists := fileOwners[file]; !exists {
			fileOwners[file] = make([]*Peer, 0)
		}
		fileOwners[file] = append(fileOwners[file], peer)
	}
	fileOwnersMu.Unlock()
	// Send response back to the client
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"status": "Peer status updated successfully"}
	json.NewEncoder(w).Encode(response)
}

func startPeerTimeoutChecker() {
	ticker := time.NewTicker(10 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		peersMu.Lock()
		updatesMu.Lock()
		for ip, lastUpdate := range peerLastUpdate {
			if time.Since(lastUpdate) > 30*time.Minute {
				fmt.Printf("Removing inactive peer with IP: %d\n", ip)
				delete(allPeers, ip)
				delete(peerLastUpdate, ip)
			}
		}
		peersMu.Unlock()
		updatesMu.Unlock()
	}
}

func getPeers (w http.ResponseWriter, r *http.Request)  {
	// Get all peers
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return 
	}
	// Send response back to the client
	//TODO based on the meesage sent from Peer
	fileName := r.URL.Query().Get("file")
	if  fileName == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return 
	}
	fileOwnersMu.RLock()
	peerListPtr , exits :=  fileOwners[fileName]
	defer fileOwnersMu.RUnlock()
	if !exits {
		http.Error(w, "No peers available for the file", http.StatusNotFound)
		return
	}
	
	sort.Slice(peerListPtr,func (i ,j int) bool {
		return peerListPtr[i].UploadingRate > peerListPtr[j].UploadingRate
	})
	if len(peerListPtr) > 50 {
		peerListPtr = peerListPtr[:50]
	}
	peerList := make([]Peer, len(peerListPtr))
	for i, peerPtr := range peerListPtr {
		if peerPtr != nil {
			peerList[i] = *peerPtr
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peerList)
	return
}

func main() {
	// Configure the server to use multiple CPU cores for concurrency
	runtime.GOMAXPROCS(runtime.NumCPU())

	go startPeerTimeoutChecker()
	// Create a custom server with scalable settings
	server := &http.Server{
		Addr:         ":8080", // Server listens on port 8080
		ReadTimeout:  10 * time.Second, // Limits how long the server will take to read the request body
		WriteTimeout: 10 * time.Second, // Limits how long the server will take to write the response
		IdleTimeout:  15 * time.Second, // Limits idle time for connections
	}

	// Set up routes without declaring specific handler functions yet
	http.HandleFunc("/register", registerPeer)

	http.HandleFunc("/updateStatus",updatePeerStatus)

	http.HandleFunc("/getPeers",getPeers)
	// Start the server with efficient logging
	log.Println("Starting scalable server on :8080")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
