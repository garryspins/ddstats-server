package ddapi

import (
	"encoding/binary"
	"errors"
	"math"
)

// DeathTypes as defined by the DD API
var DeathTypes = []string{
	"FALLEN", "SWARMED", "IMPALED", "GORED", "INFESTED", "OPENED", "PURGED",
	"DESECRATED", "SACRIFICED", "EVISCERATED", "ANNIHILATED", "INTOXICATED",
	"ENVENMONATED", "INCARNATED", "DISCARNATED", "BARBED",
}

// ErrPlayerNotFound returned when player not found from the DD API
var ErrPlayerNotFound = errors.New("player not found")

// ErrNoPlayersFound is returned when user search produces no users
var ErrNoPlayersFound = errors.New("no players found")

// Player is the struct returned after parsing the binary data
// blob returned from the DD API.
type Player struct {
	PlayerName          string  `json:"player_name"`
	PlayerID            uint64  `json:"player_id"`
	Rank                int32   `json:"rank"`
	Time                float64 `json:"time"`
	Kills               int32   `json:"kills"`
	Gems                int32   `json:"gems"`
	DaggersHit          int32   `json:"daggers_hit"`
	DaggersFired        int32   `json:"daggers_fired"`
	Accuracy            float64 `json:"accuracy"`
	DeathType           string  `json:"death_type"`
	OverallTime         float64 `json:"overall_time"`
	OverallKills        uint64  `json:"overall_kills"`
	OverallGems         uint64  `json:"overall_gems"`
	OverallDeaths       uint64  `json:"overall_deaths"`
	OverallDaggersHit   uint64  `json:"overall_daggers_hit"`
	OverallDaggersFired uint64  `json:"overall_daggers_fired"`
	OverallAccuracy     float64 `json:"overall_accuracy"`
}

// Leaderboard is a struct returned after being converted from bytes
type Leaderboard struct {
	GlobalDeaths       uint64    `json:"global_deaths"`
	GlobalKills        uint64    `json:"global_kills"`
	GlobalTime         float64   `json:"global_time"`
	GlobalGems         uint64    `json:"global_gems"`
	GlobalDaggersFired uint64    `json:"global_daggers_fired"`
	GlobalDaggersHit   uint64    `json:"global_daggers_hit"`
	GlobalAccuracy     float64   `json:"global_accuracy"`
	GlobalPlayerCount  int32     `json:"global_player_count"`
	PlayerCount        int       `json:"player_count"`
	Players            []*Player `json:"players"`
}

// GetScoresBytesToLeaderboard converts the byte array from the DD API
// to a Leaderboard struct
func GetScoresBytesToLeaderboard(b []byte, limit int) (*Leaderboard, error) {
	var leaderboard Leaderboard

	leaderboard.GlobalDeaths = toUint64(b, 11)
	leaderboard.GlobalKills = toUint64(b, 19)
	leaderboard.GlobalTime = roundToNearest(float64(toUint64(b, 35))/1000, 4)
	leaderboard.GlobalGems = toUint64(b, 43)
	leaderboard.GlobalDaggersHit = toUint64(b, 51)
	leaderboard.GlobalDaggersFired = toUint64(b, 27)
	if leaderboard.GlobalDaggersFired > 0 {
		leaderboard.GlobalAccuracy = float64(leaderboard.GlobalDaggersHit) / float64(leaderboard.GlobalDaggersFired)
	}
	leaderboard.GlobalPlayerCount = toInt32(b, 75)

	leaderboard.PlayerCount = int(toInt16(b, 59))
	if limit < leaderboard.PlayerCount {
		leaderboard.PlayerCount = limit
	}

	offset := 83
	for i := 0; i < leaderboard.PlayerCount; i++ {
		p, err := BytesToPlayer(b, offset)
		if err != nil {
			return nil, ErrPlayerNotFound
		}
		offset += len(p.PlayerName) + 90
		leaderboard.Players = append(leaderboard.Players, p)
	}

	return &leaderboard, nil
}

// BytesToPlayer takes a byte array and an initial offset
// and returns a Player object. Will return an error if the
// Player is not found
func BytesToPlayer(b []byte, bytePosition int) (*Player, error) {
	var player Player

	playerNameLength := int(toInt16(b, bytePosition))
	bytePosition += 2
	player.PlayerName = string(b[bytePosition : bytePosition+playerNameLength])
	bytePosition += playerNameLength
	// just figured out this information manually...
	player.PlayerID = toUint64(b, bytePosition+4)
	if player.PlayerID == 0 {
		return nil, ErrPlayerNotFound
	}
	player.Rank = toInt32(b, bytePosition)
	player.Time = roundToNearest(float64(toInt32(b, bytePosition+12))/10000, 4)
	player.Kills = toInt32(b, bytePosition+16)
	player.Gems = toInt32(b, bytePosition+28)
	player.DaggersHit = toInt32(b, bytePosition+24)
	player.DaggersFired = toInt32(b, bytePosition+20)
	if player.DaggersFired > 0 {
		player.Accuracy = roundToNearest(float64(player.DaggersHit)/float64(player.DaggersFired)*100, 2)
	}
	player.DeathType = DeathTypes[toInt16(b, bytePosition+32)]
	player.OverallTime = roundToNearest(float64(toUint64(b, bytePosition+60))/10000, 4)
	player.OverallKills = toUint64(b, bytePosition+44)
	player.OverallGems = toUint64(b, bytePosition+68)
	player.OverallDeaths = toUint64(b, bytePosition+36)
	player.OverallDaggersHit = toUint64(b, bytePosition+76)
	player.OverallDaggersFired = toUint64(b, bytePosition+52)
	if player.OverallDaggersFired > 0 {
		player.OverallAccuracy = roundToNearest(float64(player.OverallDaggersHit)/float64(player.OverallDaggersFired)*100, 2)
	}

	return &player, nil
}

// UserSearchBytesToPlayers converts a byte array to a player slice
func UserSearchBytesToPlayers(b []byte) ([]*Player, error) {
	playerCount := int(toInt16(b, 11))
	if playerCount < 1 {
		return nil, ErrNoPlayersFound
	}
	var players []*Player
	offset := 19
	for i := 0; i < playerCount; i++ {
		p, err := BytesToPlayer(b, offset)
		if err != nil {
			return nil, ErrPlayerNotFound
		}
		offset += len(p.PlayerName) + 90
		players = append(players, p)
	}
	return players, nil
}

func toUint64(b []byte, offset int) uint64 {
	return binary.LittleEndian.Uint64(b[offset : offset+8])
}

func toInt64(b []byte, offset int) int64 {
	return int64(binary.LittleEndian.Uint64(b[offset : offset+4]))
}

func toUint32(b []byte, offset int) uint32 {
	return binary.LittleEndian.Uint32(b[offset : offset+4])
}

func toInt32(b []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
}

func toInt16(b []byte, offset int) int16 {
	return int16(binary.LittleEndian.Uint16(b[offset : offset+2]))
}

func roundToNearest(f float64, numberOfDecimalPlaces int) float64 {
	multiplier := math.Pow10(numberOfDecimalPlaces)
	return math.Round(f*multiplier) / multiplier
}