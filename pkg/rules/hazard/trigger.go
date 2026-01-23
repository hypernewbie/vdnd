package hazard

import (
	"uaa/vdnd/pkg/rules/ability"
)

type TriggerType int

const (
	TriggerEnter     TriggerType = iota // Creature enters area
	TriggerTouch                        // Creature touches object
	TriggerOpen                         // Container/door opened
	TriggerProximity                    // Within X feet
	TriggerPressure                     // Weight on plate
	TriggerTimeBased                    // After X rounds/time
)

type TriggerCondition struct {
	Type      TriggerType
	Area      string // Zone or position ID
	Radius    int    // For proximity
	MinWeight int    // For pressure plates
	Delay     int    // Rounds before activation
}

// Matches checks if an event matches this trigger
func (t TriggerCondition) Matches(event ability.Event, hazardPosition string) bool {
	switch t.Type {
	case TriggerEnter:
		// EventMove to the specified Area
		return event.Type == ability.EventMove && event.Position == t.Area
	case TriggerProximity:
		// In a real system we'd calculate distance. 
		// In zone-based, proximity might just mean "same zone" or "adjacent".
		// For MVP, we'll assume event.Position == hazardPosition means proximity triggered if radius is small.
		return event.Type == ability.EventMove && event.Position == hazardPosition
	case TriggerTouch:
		// EventManipulate targeting the hazard position or ID
		// Details map might contain TargetID
		if event.Type == ability.EventManipulate {
			if id, ok := event.Details["target_id"]; ok && id == hazardPosition {
				return true
			}
		}
		return false
	case TriggerPressure:
		// Similar to Enter but maybe check weight if available in details
		if event.Type == ability.EventMove && event.Position == t.Area {
			if weight, ok := event.Details["weight"].(int); ok {
				return weight >= t.MinWeight
			}
			return true // Default trigger if weight unknown?
		}
		return false
	}
	return false
}
