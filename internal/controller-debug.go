package regelverk

import (
	"encoding/json"
	"net/http"
	"sync"
)

type DebugController struct {
	masterController *MasterController
	mu               sync.Mutex
	initialized      bool
}

func (c *DebugController) Lock() {
	c.mu.Lock()
}

func (c *DebugController) Unlock() {
	c.mu.Unlock()
}

func (c *DebugController) IsInitialized() bool {
	return c.initialized
}

func (c *DebugController) Initialize(masterController *MasterController) []MQTTPublish {
	c.masterController = masterController
	http.HandleFunc("/debug/statevaluemap", c.stateValueMapHandler)
	c.initialized = true
	return nil
}

func (c *DebugController) ProcessEvent(_ MQTTEvent) []MQTTPublish {
	return nil
}

func (c *DebugController) stateValueMapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := c.masterController.stateValueMap.Snapshot()

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		http.Error(w, "failed to encode stateValueMap", http.StatusInternalServerError)
		return
	}
}
