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
	Name             string
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
	c.Name = "debug"
	http.HandleFunc("/debug/statevalues", c.stateValueMapHandler)
	http.HandleFunc("/debug/devicestate", c.deviceStateHandler)
	c.initialized = true
	return nil
}

func (c *DebugController) ProcessEvent(_ MQTTEvent) []MQTTPublish {
	return nil
}

func (c *DebugController) DebugState() ControllerDebugState {
	return ControllerDebugState{
		Name:        c.Name,
		Initialized: c.initialized,
	}
}

func (c *DebugController) stateValueMapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := c.masterController.stateValueMap.Snapshot()
	controllerStates := []ControllerDebugState{}
	if c.masterController.controllers != nil {
		for _, controller := range *c.masterController.controllers {
			controllerStates = append(controllerStates, controller.DebugState())
		}
	}

	payload := struct {
		StateValueMap map[string]StateValueDebug `json:"stateValueMap"`
		Controllers   []ControllerDebugState     `json:"controllers"`
	}{
		StateValueMap: snapshot,
		Controllers:   controllerStates,
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		http.Error(w, "failed to encode stateValueMap", http.StatusInternalServerError)
		return
	}
}

func (c *DebugController) deviceStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot := c.masterController.deviceStateStore.DebugSnapshot()

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		http.Error(w, "failed to encode device state", http.StatusInternalServerError)
		return
	}
}
