package graph

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"texxer/internal/img"
)

type hash [32]byte

// Cache shares results across evaluations. Entries are keyed by a hash of the
// node's type, params and everything upstream, so editing a param only
// invalidates the nodes after it.
type Cache struct {
	mu    sync.Mutex
	items map[hash]*img.Buffer
	order []hash
	bytes int
	limit int
}

func NewCache(limitBytes int) *Cache {
	return &Cache{items: map[hash]*img.Buffer{}, limit: limitBytes}
}

func (c *Cache) get(k hash) (*img.Buffer, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	b, ok := c.items[k]
	return b, ok
}

func (c *Cache) put(k hash, b *img.Buffer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[k]; ok {
		return
	}

	c.items[k] = b
	c.order = append(c.order, k)
	c.bytes += len(b.Pix) * 4

	for c.bytes > c.limit && len(c.order) > 1 {
		old := c.order[0]
		c.order = c.order[1:]
		c.bytes -= len(c.items[old].Pix) * 4
		delete(c.items, old)
	}
}

// structHash covers a node and its upstream graph, independent of the render context.
func (ev *Evaluator) structHash(id string, seen map[string]bool) (hash, error) {
	if h, ok := ev.hashes[id]; ok {
		return h, nil
	}
	if seen[id] {
		return hash{}, fmt.Errorf("cycle at node %q", id)
	}
	seen[id] = true

	node, ok := ev.graph.Nodes[id]
	if !ok {
		return hash{}, fmt.Errorf("unknown node %q", id)
	}

	params, err := json.Marshal(node.Params)
	if err != nil {
		return hash{}, err
	}

	s := sha256.New()
	fmt.Fprintf(s, "%s\x00%s\x00%s\x00%d\x00", node.Type, params, ev.graph.Dir, ev.seed)

	names := make([]string, 0, len(node.Inputs))
	for name := range node.Inputs {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		ref := node.Inputs[name]
		up, out, _ := strings.Cut(ref, ".")

		h, err := ev.structHash(up, seen)
		if err != nil {
			return hash{}, err
		}
		fmt.Fprintf(s, "%s\x00%s\x00%x\x00", name, out, h)
	}

	var h hash
	s.Sum(h[:0])
	ev.hashes[id] = h
	return h, nil
}

func (ev *Evaluator) cacheKey(id, out string, ctx Ctx) (hash, error) {
	h, err := ev.structHash(id, map[string]bool{})
	if err != nil {
		return hash{}, err
	}

	var k hash
	s := sha256.New()
	fmt.Fprintf(s, "%x\x00%s\x00%v", h, out, ctx)
	s.Sum(k[:0])
	return k, nil
}
