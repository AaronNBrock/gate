package packet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bxcodec/faker/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.minekube.com/common/minecraft/color"
	"go.minekube.com/common/minecraft/component"
	"go.minekube.com/gate/pkg/edition/java/profile"
	"go.minekube.com/gate/pkg/edition/java/proto/packet/plugin"
	"go.minekube.com/gate/pkg/edition/java/proto/version"
	"go.minekube.com/gate/pkg/gate/proto"
	"go.minekube.com/gate/pkg/util/uuid"
	"io"
	"reflect"
	"testing"
)

// All packets to test.
// Empty packets are being initialized with random fake data at runtime.
var packets = []proto.Packet{
	&plugin.Message{},
	&Chat{},
	&ClientSettings{},
	&Disconnect{},
	&Handshake{},
	&KeepAlive{},
	&ServerLogin{},
	&EncryptionResponse{},
	&LoginPluginResponse{},
	&ServerLoginSuccess{},
	&SetCompression{},
	&LoginPluginMessage{},
	&ResourcePackRequest{},
	&Respawn{},
	&StatusRequest{},
	&StatusResponse{},
	&StatusPing{},
	&HeaderAndFooter{},
	&EncryptionRequest{
		ServerID:    "984hgf8097c4gh8734hr",
		PublicKey:   []byte("9wh90fh23dh203d2b23b3"),
		VerifyToken: []byte("32f8d89dh3di"),
	},
	&Title{
		Action:    SetSubtitle,
		Component: strPtr(`{"text":"sub title"}`),
		FadeIn:    1,
		Stay:      2,
		FadeOut:   3,
	},
	&PlayerListItem{
		Action: UpdateLatencyPlayerListItemAction,
		Items: []PlayerListItemEntry{
			{
				ID:   testUUID,
				Name: "testName",
				Properties: []profile.Property{
					*mustFake(&profile.Property{}).(*profile.Property),
					*mustFake(&profile.Property{}).(*profile.Property),
					*mustFake(&profile.Property{}).(*profile.Property),
				},
				GameMode:    2,
				Latency:     4325,
				DisplayName: &component.Text{Content: "Bob", S: component.Style{Color: color.Red}},
			},
			{
				ID:   testUUID,
				Name: "testName2",
				Properties: []profile.Property{
					*mustFake(&profile.Property{}).(*profile.Property),
					*mustFake(&profile.Property{}).(*profile.Property),
					*mustFake(&profile.Property{}).(*profile.Property),
				},
				GameMode:    1,
				Latency:     42,
				DisplayName: &component.Text{Content: "Alice", S: component.Style{Color: color.Green}},
			},
		},
	},
	&JoinGame{
		EntityID:             4,
		Gamemode:             1,
		Dimension:            4,
		PartialHashedSeed:    1,
		Difficulty:           3,
		Hardcore:             true,
		MaxPlayers:           3,
		LevelType:            strPtr("test"),
		ViewDistance:         3,
		ReducedDebugInfo:     true,
		ShowRespawnScreen:    true,
		DimensionRegistry:    mustFake(&DimensionRegistry{}).(*DimensionRegistry),
		DimensionInfo:        mustFake(&DimensionInfo{}).(*DimensionInfo),
		CurrentDimensionData: mustFake(&DimensionData{}).(*DimensionData),
		PreviousGamemode:     2,
		BiomeRegistry: map[string]interface{}{
			"k": "v",
		},
	},
}

// fill packets with fake data
func init() {
	for _, p := range packets {
		if !reflect.ValueOf(p).Elem().IsZero() {
			continue
		}
		// Fill fake data
		if err := faker.FakeData(p); err != nil {
			panic(fmt.Sprintf("error fake %T: %v", p, err))
		}
	}
}

func TestPackets(t *testing.T) {
	PacketCodings(t,
		[]proto.Direction{proto.ServerBound, proto.ClientBound},
		vRange(version.MinimumVersion, version.MaximumVersion),
		packets...)
}

// Helper - Compares encoding vs. decoding for various versions and packet types
func PacketCodings(t *testing.T,
	directions []proto.Direction,
	versions []*proto.Version,
	samples ...proto.Packet,
) {
	t.Helper()

	message := func(direction proto.Direction, v *proto.Version, packet reflect.Type) string {
		return fmt.Sprintf("Type: %s, Direction: %s, Version: %s", packet.String(), direction, v)
	}

	bufA1, bufA2, bufB1, bufB2 := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
	for _, direction := range directions {
		for _, v := range versions {
			c := &proto.PacketContext{Direction: direction, Protocol: v.Protocol}
			for _, sample := range samples {
				packetType := reflect.TypeOf(sample).Elem()
				msg := message(direction, v, packetType)

				// Encode sample at protocol version drop unnecessary data for that version
				assert.NoError(t, sample.Encode(c, io.MultiWriter(bufA1, bufA2)), msg)
				// Decode that bytes that versioned packet
				a := reflect.New(packetType).Interface().(proto.Packet)
				assert.NoError(t, a.Decode(c, bufA1), msg)

				// Now encode it again
				assert.NoError(t, a.Encode(c, io.MultiWriter(bufB1, bufB2)), msg)
				b := reflect.New(packetType).Interface().(proto.Packet)
				// And decode it again.
				assert.NoError(t, b.Decode(c, bufB1), msg)

				// Both encode buffs should be equal
				if !bytes.Equal(bufA2.Bytes(), bufB2.Bytes()) {
					// Fallback to test json difference.
					jsonA, err := json.MarshalIndent(a, "", "  ")
					require.NoError(t, err)
					jsonB, err := json.MarshalIndent(b, "", "  ")
					require.NoError(t, err)
					assert.Equal(t, string(jsonA), string(jsonB), msg)
				}

				// Both decode buffs should be empty
				assert.Equal(t, 0, bufA1.Len(), msg)
				assert.Equal(t, 0, bufB1.Len(), msg)

				bufA1.Reset()
				bufA2.Reset()
				bufB1.Reset()
				bufB2.Reset()
			}
		}
	}
}

var testUUID, _ = uuid.Parse(`123e4567-e89b-12d3-a456-426614174000`)

func vRange(start, endInclusive *proto.Version) (vers []*proto.Version) {
	for _, v := range version.Versions { // assumes Versions is sorted
		if v.GreaterEqual(start) && v.LowerEqual(endInclusive) {
			vers = append(vers, v)
		}
	}
	return
}
func strPtr(s string) *string { return &s }
func mustFake(a interface{}) interface{} {
	if err := faker.FakeData(a); err != nil {
		panic(fmt.Sprintf("error faking %T: %v", a, err))
	}
	return a
}
