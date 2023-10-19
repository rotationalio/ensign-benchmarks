package workload

import (
	"fmt"
	"strings"
	"time"

	api "github.com/rotationalio/go-ensign/api/v1beta1"
	mimetype "github.com/rotationalio/go-ensign/mimetype/v1beta1"
)

type RandomDuplicates struct {
	data           []string
	keyset         map[string][]string
	newKeyProb     float64
	duplicateProb  float64
	versUpdateProb float64
	prev           time.Time
	mimetypes      []mimetype.MIME
	etypes         []*api.Type
}

func NewRandomDuplicates(nKeys int, newKeyProb, duplicateProb float64) *RandomDuplicates {
	workload := &RandomDuplicates{
		data:           make([]string, 0),
		keyset:         make(map[string][]string, nKeys),
		newKeyProb:     newKeyProb,
		duplicateProb:  duplicateProb,
		versUpdateProb: 0.15,
		prev:           time.Now(),
		mimetypes:      []mimetype.MIME{mimetype.ApplicationOctetStream, mimetype.ApplicationOctetStream, mimetype.ApplicationOctetStream, mimetype.ApplicationOctetStream, mimetype.ApplicationOctetStream, mimetype.MIME_USER_SPECIFIED7},
		etypes:         []*api.Type{{Name: "RandomData", MajorVersion: 1}},
	}

	for i := 0; i < nKeys; i++ {
		workload.keyset[MkKey()] = []string{MkVal()}
	}

	return workload
}

func (r *RandomDuplicates) Next() *api.EventWrapper {
	return MkWrap(
		r.Data(),
		r.Metadata(),
		r.Mimetype(),
		r.EventType(),
		r.Created(),
	)
}

func (r *RandomDuplicates) Data() string {
	if flip(r.duplicateProb) {
		i := rnd.Intn(len(r.data))
		return r.data[i]
	}

	data := MkData(0)
	r.data = append(r.data, data)
	return data
}

func (r *RandomDuplicates) Metadata() string {
	n := rnd.Intn(len(r.keyset))
	kvs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		kvs = append(kvs, r.KeyVal())
	}
	return strings.Join(kvs, ",")
}

func (r *RandomDuplicates) KeyVal() string {
	key := r.RandKey()
	if flip(r.newKeyProb) {
		r.keyset[key] = append(r.keyset[key], MkVal())
	}

	val := r.RandVal(key)
	return fmt.Sprintf("%s:%s", key, val)
}

func (r *RandomDuplicates) RandKey() string {
	i := rnd.Intn(len(r.keyset))
	for key := range r.keyset {
		if i == 0 {
			return key
		}
		i--
	}
	panic("iteration ended without finding a random key/val pair")
}

func (r *RandomDuplicates) RandVal(key string) string {
	i := rnd.Intn(len(r.keyset[key]))
	return r.keyset[key][i]
}

func (r *RandomDuplicates) Mimetype() mimetype.MIME {
	if len(r.mimetypes) == 1 {
		return r.mimetypes[0]
	}

	i := rnd.Intn(len(r.mimetypes))
	return r.mimetypes[i]
}

func (r *RandomDuplicates) EventType() string {
	var etype *api.Type
	if len(r.etypes) == 1 {
		etype = r.etypes[0]
	} else {
		i := rnd.Intn(len(r.etypes))
		etype = r.etypes[i]
	}

	if r.versUpdateProb > 0 {
		if flip(r.versUpdateProb) {
			updateType(etype)
		}
	}

	return fmt.Sprintf("%s v%d.%d.%d", etype.Name, etype.MajorVersion, etype.MinorVersion, etype.PatchVersion)
}

func (r *RandomDuplicates) Created() string {
	r.prev = r.prev.Add(time.Duration(rnd.Int63n(int64(15 * time.Minute))))
	return r.prev.Format(time.RFC3339Nano)
}

func (r *RandomDuplicates) SetMimes(mimetypes ...mimetype.MIME) {
	r.mimetypes = mimetypes
}

func (r *RandomDuplicates) SetTypes(etypes ...*api.Type) {
	r.etypes = etypes
}

func (r *RandomDuplicates) SetVersionUpgradeProbability(prob float64) {
	r.versUpdateProb = prob
}

func flip(prob float64) bool {
	return rnd.Float64() <= prob
}

var (
	majorProb float64 = 0.05
	minorProb float64 = 0.45
	patchProb float64 = 0.50
)

func updateType(etype *api.Type) {
	spin := rnd.Float64()
	switch {
	case spin <= majorProb:
		etype.MajorVersion += 1
		etype.MinorVersion = 0
		etype.PatchVersion = 0
		return
	case spin <= minorProb+majorProb:
		etype.MinorVersion += 1
		etype.PatchVersion = 0
		return
	case spin <= patchProb+minorProb+majorProb:
		etype.PatchVersion += 1
		return
	}
}
