package jsondiff

import (
	"encoding/binary"
	"hash/maphash"
	"math"
	"slices"
)

type hasher struct {
	mh maphash.Hash
}

func (h *hasher) digest(val interface{}, sort bool) uint64 {
	h.mh.Reset()
	h.hash(val, sort)

	return h.mh.Sum64()
}

func (h *hasher) hash(i interface{}, sort bool) {
	switch v := i.(type) {
	case string:
		_, _ = h.mh.WriteString(v)
	case bool:
		if v {
			_ = h.mh.WriteByte('1')
		} else {
			_ = h.mh.WriteByte('0')
		}
	case float64:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(v))
		_, _ = h.mh.Write(buf[:])
	case nil:
		_ = h.mh.WriteByte('0')
	case []interface{}:
		if sort {
			// Hash the element digests in sorted order,
			// so that the digest does not depend on the
			// order of the elements. The array itself is
			// left untouched: it belongs to the document
			// being compared, which the differ walks by
			// index afterwards.
			h.hashUnordered(v)
			return
		}
		for _, e := range v {
			h.hash(e, sort)
		}
	case map[string]interface{}:
		keys := make([]string, 0, len(v))

		// Extract keys first, and sort them
		// in lexicographical order.
		for k := range v {
			keys = append(keys, k)
		}
		sortStrings(keys)

		for _, k := range keys {
			_, _ = h.mh.WriteString(k)
			h.hash(v[k], sort)
		}
	}
}

func (h *hasher) hashUnordered(a []interface{}) {
	eh := hasher{}
	eh.mh.SetSeed(h.mh.Seed())

	digests := make([]uint64, len(a))
	for i, e := range a {
		digests[i] = eh.digest(e, true)
	}
	slices.Sort(digests)

	var buf [8]byte
	for _, d := range digests {
		binary.BigEndian.PutUint64(buf[:], d)
		_, _ = h.mh.Write(buf[:])
	}
}
