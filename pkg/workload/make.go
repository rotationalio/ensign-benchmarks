package workload

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rotationalio/ensign/pkg/ensign/rlid"
	api "github.com/rotationalio/go-ensign/api/v1beta1"
	mimetype "github.com/rotationalio/go-ensign/mimetype/v1beta1"
	region "github.com/rotationalio/go-ensign/region/v1beta1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	rnd       *rand.Rand
	seq       rlid.Sequence
	topicID   ulid.ULID
	publisher *api.Publisher
	pubRegion region.Region
)

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	seq = rlid.Sequence(0)
	topicID = ulid.MustParse("01HBETJKP2ES10XXMK27M651GA")
	publisher = &api.Publisher{
		PublisherId: "01HD1KY309F3SHSV1M89GSQBDF",
		Ipaddr:      "192.168.158.211",
		ClientId:    "data-ingestor-alpha",
		UserAgent:   "PyEnsign v0.12.0",
	}
	pubRegion = region.Region_LKE_US_EAST_1A
}

// Helper for quickly making wrapped events for testing purposes. See MkEvent for the
// details about how to use the specified parameters. Wrapper metadata is set using
// global defaults, which can be modified using the setter methods.
func MkWrap(data, kvs string, mime mimetype.MIME, etype, created string) *api.EventWrapper {
	event := MkEvent(data, kvs, mime, etype, created)
	eventID := seq.Next()

	wrap := &api.EventWrapper{
		Id:          eventID.Bytes(),
		TopicId:     topicID.Bytes(),
		Committed:   timestamppb.New(event.Created.AsTime().Add(randDuration(30 * time.Second))),
		Offset:      uint64(eventID.Sequence()),
		Epoch:       uint64(0xbb),
		Region:      pubRegion,
		Publisher:   publisher,
		Encryption:  &api.Encryption{EncryptionAlgorithm: api.Encryption_PLAINTEXT},
		Compression: &api.Compression{Algorithm: api.Compression_NONE},
	}

	wrap.Wrap(event)
	return wrap
}

// Helper to quickly make events for testing purposes. Data should be base64 encoded
// data, kvs should be key:val,key:val pairs, etype should be a semvar for the type,
// e.g. Generic v1.2.3. and finally created should be an RFC3339 string or empty string
// to use a constant timestamp. If data or kvs is empty then empty byte array and empty
// metadata maps will be created. If etype is empty then the unspecified type is used.
func MkEvent(data, kvs string, mime mimetype.MIME, etype, created string) *api.Event {
	e := &api.Event{
		Data:     make([]byte, 0),
		Metadata: make(map[string]string),
		Mimetype: mime,
		Type:     &api.Type{Name: "Unspecified"},
		Created:  timestamppb.New(time.Now().In(time.UTC)),
	}

	if data != "" {
		var err error
		if e.Data, err = base64.RawStdEncoding.DecodeString(data); err != nil {
			e.Data = []byte(data)
		}
	}

	if kvs != "" {
		for _, pair := range strings.Split(kvs, ",") {
			parts := strings.Split(pair, ":")
			switch len(parts) {
			case 0:
				continue
			case 1:
				e.Metadata[parts[0]] = ""
			case 2:
				e.Metadata[parts[0]] = parts[1]
			default:
				panic("could not parse kvs")
			}
		}
	}

	if etype != "" {
		e.Type = &api.Type{}
		parts := strings.Split(etype, " ")
		if len(parts) == 2 {
			e.Type.Name = parts[0]

			semver := strings.Split(strings.TrimPrefix(parts[1], "v"), ".")
			if len(semver) == 3 {
				e.Type.MajorVersion = parseuint32(semver[0])
				e.Type.MinorVersion = parseuint32(semver[1])
				e.Type.PatchVersion = parseuint32(semver[2])
			} else {
				panic("could not parse etype version")
			}
		} else {
			panic("could not parse etype")
		}
	}

	if created != "" {
		if ts, err := time.Parse(time.RFC3339, created); err == nil {
			e.Created = timestamppb.New(ts)
		}
	}

	return e
}

// Generate random data and convert to base64 without error or panic.
func MkData(s int) string {
	if s == 0 {
		s = rnd.Intn(4096)
	}

	data := make([]byte, s)
	crand.Read(data)
	return base64.RawStdEncoding.EncodeToString(data)
}

func SetRand(nrnd *rand.Rand) {
	rnd = nrnd
}

func SetSequence(nseq rlid.Sequence) {
	seq = nseq
}

func SetTopicId(id ulid.ULID) {
	topicID = id
}

func SetPublisher(pub *api.Publisher) {
	publisher = pub
}

func SetRegion(nreg region.Region) {
	pubRegion = nreg
}

func parseuint32(s string) uint32 {
	num, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		panic(err)
	}
	return uint32(num)
}

func randDuration(max time.Duration) time.Duration {
	return time.Duration(rnd.Int63n(int64(max)))
}
