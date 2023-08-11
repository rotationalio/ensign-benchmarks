package sustain

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	"github.com/rotationalio/go-ensign"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	mimetype "github.com/rotationalio/go-ensign/mimetype/v1beta1"
)

type EventFactory func() *ensign.Event

func MakeEventFactory(size int) EventFactory {
	count := uint64(0)
	version := benchmarks.Version()
	etype := &api.Type{
		Name:         "Random",
		MajorVersion: 1,
	}

	entropy := ulid.Monotonic(rand.Reader, 0)
	idgen := func() ulid.ULID {
		ms := ulid.Timestamp(time.Now())
		id, err := ulid.New(ms, entropy)
		if err != nil {
			panic(err)
		}
		return id
	}

	return func() *ensign.Event {
		count++
		event := &ensign.Event{
			Data: generateRandomBytes(size),
			Metadata: map[string]string{
				"app":      "enbench",
				"counter":  fmt.Sprintf("%x", count),
				"version":  version,
				"local_id": idgen().String(),
			},
			Mimetype: mimetype.ApplicationOctetStream,
			Type:     etype,
			Created:  time.Now(),
		}
		return event
	}
}

func generateRandomBytes(n int) (b []byte) {
	b = make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
