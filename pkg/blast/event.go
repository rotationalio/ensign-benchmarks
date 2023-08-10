package blast

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	benchmarks "github.com/rotationalio/ensign-benchmarks/pkg"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	mimetype "github.com/rotationalio/go-ensign/mimetype/v1beta1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventFactory func() *api.EventWrapper

func MakeEventFactory(size int, topicID ulid.ULID) EventFactory {
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

	return func() *api.EventWrapper {
		count++
		event := &api.Event{
			Data: generateRandomBytes(size),
			Metadata: map[string]string{
				"app":     "enbench",
				"counter": fmt.Sprintf("%x", count),
				"version": version,
			},
			Mimetype: mimetype.ApplicationOctetStream,
			Type:     etype,
			Created:  timestamppb.Now(),
		}

		localID := idgen()
		wrap := &api.EventWrapper{
			TopicId: topicID.Bytes(),
			LocalId: localID.Bytes(),
		}
		if err := wrap.Wrap(event); err != nil {
			panic(err)
		}

		return wrap
	}
}

func generateRandomBytes(n int) (b []byte) {
	b = make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
