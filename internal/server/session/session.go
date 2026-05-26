package session

import (
	"math"
	"time"
)

// ClientRole classifies a connected client session.
type ClientRole int32

const (
	ClientRoleUnspecified ClientRole = 0
	ClientRolePlayer      ClientRole = 1
	ClientRoleNPC         ClientRole = 2
	ClientRoleAdmin       ClientRole = 3
	ClientRoleOther       ClientRole = 4
)

func (r ClientRole) String() string {
	switch r {
	case ClientRolePlayer:
		return "PLAYER"
	case ClientRoleNPC:
		return "NPC"
	case ClientRoleAdmin:
		return "ADMIN"
	case ClientRoleOther:
		return "OTHER"
	default:
		return "UNSPECIFIED"
	}
}

// ClientSession holds server-side state for a single connected client.
type ClientSession struct {
	SessionID    string
	ClientUUID   string
	Label        string
	Role         ClientRole
	Color        [3]uint8   // RGB, server-assigned from palette
	Position     [3]float64 // world position in sim units; server-authoritative
	Velocity     [3]float64 // velocity in sim units/s; integrated by N-body (F-022 Ph2)
	POV          [3]float32 // facing direction unit vector; client-authoritative
	MarkerRef    string     // model ref; empty in Phase 1
	ConnectedAt  time.Time
	LastSeen     time.Time
	DamageRating float32 // 0.0 = undamaged, 1.0 = destroyed
	SpawnRef     string  // name of body or belt where client spawned
}

// RegisterRequest is the input to Registry.Register.
type RegisterRequest struct {
	ClientUUID  string
	Label       string     // max 32 UTF-8 chars; server truncates if needed
	Role        ClientRole // requested role; server may downgrade to Player
	AdminSecret string     // required when requesting ClientRoleAdmin
}

// colorPalette is a fixed set of 100 visually distinct RGB colors generated
// by HSL hue rotation at constant saturation (0.75) and lightness (0.55).
// Assigned in registration order; released on disconnect for FIFO reuse.
var colorPalette [100][3]uint8

func init() {
	for i := 0; i < 100; i++ {
		h := float64(i) * 3.6 // 0..356.4 degrees
		colorPalette[i] = hslToRGB(h, 0.75, 0.55)
	}
}

// hslToRGB converts HSL (h in degrees, s and l in [0,1]) to RGB uint8.
func hslToRGB(h, s, l float64) [3]uint8 {
	h /= 360.0
	var r, g, b float64
	if s == 0 {
		r, g, b = l, l, l
	} else {
		q := l + s - l*s
		if l < 0.5 {
			q = l * (1 + s)
		}
		p := 2*l - q
		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}
	return [3]uint8{
		uint8(math.Round(r * 255)),
		uint8(math.Round(g * 255)),
		uint8(math.Round(b * 255)),
	}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 0.5:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
