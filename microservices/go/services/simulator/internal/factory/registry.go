// internal/factory/registry.go
package factory

import (
	"fmt"
)

// Registry holds all available modular parts for household generation.
type Registry struct {
	//emergent
	Devices      map[string]CatalogDevice
	Actions      map[string]CatalogAction
	Personas     map[string]CatalogPersona
	Systems      map[string]CatalogSystem
	Compositions map[string]CatalogComposition

	WaterSystems map[string]CatalogWaterSystem

	//routines -> the societial influence
	Routines         map[string]CatalogRoutine
	Alarms           map[string]CatalogAlarm
	CollectiveEvents map[string]CatalogEvent
}

// NewRegistry initializes an empty catalog store.
func NewRegistry() *Registry {
	return &Registry{
		Devices:      make(map[string]CatalogDevice),
		Actions:      make(map[string]CatalogAction),
		Personas:     make(map[string]CatalogPersona),
		Systems:      make(map[string]CatalogSystem),
		Compositions: make(map[string]CatalogComposition),
		WaterSystems: make(map[string]CatalogWaterSystem),

		Routines:         make(map[string]CatalogRoutine),
		Alarms:           make(map[string]CatalogAlarm),
		CollectiveEvents: make(map[string]CatalogEvent),
	}
}

func (r *Registry) AddWaterSystem(ws CatalogWaterSystem) error {
	if _, exists := r.WaterSystems[ws.ID]; exists {
		return fmt.Errorf("water system ID %s already registered", ws.ID)
	}
	r.WaterSystems[ws.ID] = ws
	return nil
}

func (r *Registry) AddComposition(comp CatalogComposition) error {
	if _, exists := r.Compositions[comp.ID]; exists {
		return fmt.Errorf("composition ID %s already registered", comp.ID)
	}
	r.Compositions[comp.ID] = comp
	return nil
}

// AddDevice safely registers a device.
func (r *Registry) AddDevice(dev CatalogDevice) error {
	if _, exists := r.Devices[dev.ID]; exists {
		return fmt.Errorf("device ID %s already registered", dev.ID)
	}
	r.Devices[dev.ID] = dev
	return nil
}

// AddAction safely registers an action.
func (r *Registry) AddAction(act CatalogAction) error {
	if _, exists := r.Actions[act.ID]; exists {
		return fmt.Errorf("action ID %s already registered", act.ID)
	}
	r.Actions[act.ID] = act
	return nil
}

// AddPersona safely registers a persona.
func (r *Registry) AddPersona(per CatalogPersona) error {
	if _, exists := r.Personas[per.ID]; exists {
		return fmt.Errorf("persona ID %s already registered", per.ID)
	}
	r.Personas[per.ID] = per
	return nil
}

// AddSystem safely registers an ambient system.
func (r *Registry) AddSystem(sys CatalogSystem) error {
	if _, exists := r.Systems[sys.ID]; exists {
		return fmt.Errorf("system ID %s already registered", sys.ID)
	}
	r.Systems[sys.ID] = sys
	return nil
}

func (r *Registry) AddRoutine(rout CatalogRoutine) error {
	if _, exists := r.Routines[rout.ID]; exists {
		return fmt.Errorf("routine ID %s already registered", rout.ID)
	}
	r.Routines[rout.ID] = rout
	return nil
}

func (r *Registry) AddAlarm(al CatalogAlarm) error {
	if _, exists := r.Alarms[al.ID]; exists {
		return fmt.Errorf("alarm ID %s already registered", al.ID)
	}
	r.Alarms[al.ID] = al
	return nil
}

func (r *Registry) AddEvent(ev CatalogEvent) error {
	if _, exists := r.CollectiveEvents[ev.ID]; exists {
		return fmt.Errorf("event ID %s already registered", ev.ID)
	}
	r.CollectiveEvents[ev.ID] = ev
	return nil
}
